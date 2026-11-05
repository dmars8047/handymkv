package hmkv

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TheDiscDB (https://thediscdb.com) is a community database that maps the titles of a disc
// (as MakeMKV sees them) to the movie, episode or extra they contain. handymkv uses its public
// GraphQL API to show the real name of each title in the selection list and to name the ripped
// files after it.

const (
	defaultDiscDBEndpoint = "https://thediscdb.com/graphql/"
	discDBRequestTimeout  = 20 * time.Second
)

// ErrDiscDBNoMatch is returned when TheDiscDB does not know the disc.
var ErrDiscDBNoMatch = errors.New("disc not found in TheDiscDB")

// DiscDBClient talks to the public TheDiscDB GraphQL endpoint.
type DiscDBClient struct {
	Endpoint   string
	APIKey     string
	UserAgent  string
	HTTPClient *http.Client
}

func NewDiscDBClient(apiKey, userAgent string) *DiscDBClient {
	return &DiscDBClient{
		Endpoint:   defaultDiscDBEndpoint,
		APIKey:     apiKey,
		UserAgent:  userAgent,
		HTTPClient: &http.Client{Timeout: discDBRequestTimeout},
	}
}

// Data model of the GraphQL responses (only the fields handymkv requests).

type discDBItem struct {
	Title   string `json:"title"`
	Type    string `json:"type"`
	Season  string `json:"season"`
	Episode string `json:"episode"`
}

type discDBTitle struct {
	Index      int         `json:"index"`
	SourceFile string      `json:"sourceFile"`
	SegmentMap string      `json:"segmentMap"`
	Duration   string      `json:"duration"`
	Size       int64       `json:"size"`
	Item       *discDBItem `json:"item"`
}

type discDBDisc struct {
	Index       int           `json:"index"`
	Name        string        `json:"name"`
	Format      string        `json:"format"`
	ContentHash string        `json:"contentHash"`
	Titles      []discDBTitle `json:"titles"`
}

type discDBRelease struct {
	Title string       `json:"title"`
	Year  int          `json:"year"`
	Discs []discDBDisc `json:"discs"`
}

type discDBMediaItem struct {
	Title    string          `json:"title"`
	Year     int             `json:"year"`
	Type     string          `json:"type"`
	Releases []discDBRelease `json:"releases"`
}

type discDBBoxset struct {
	Title   string         `json:"title"`
	Release *discDBRelease `json:"release"`
}

type discDBQueryData struct {
	MediaItems *struct {
		Nodes []discDBMediaItem `json:"nodes"`
	} `json:"mediaItems"`
	Boxsets *struct {
		Nodes []discDBBoxset `json:"nodes"`
	} `json:"boxsets"`
}

// DiscDBMatch is one disc of one release in TheDiscDB.
type DiscDBMatch struct {
	// Title of the movie or series, or of the boxset when the disc belongs to one.
	MediaTitle string
	MediaYear  int
	MediaType  string
	// Title of the release (e.g. "Season 1 Blu-ray").
	ReleaseTitle string
	Disc         discDBDisc
}

func (m DiscDBMatch) String() string {
	var sb strings.Builder
	sb.WriteString(m.MediaTitle)
	if m.MediaYear > 0 {
		fmt.Fprintf(&sb, " (%d)", m.MediaYear)
	}
	if m.ReleaseTitle != "" {
		fmt.Fprintf(&sb, " - %s", m.ReleaseTitle)
	}
	disc := m.Disc.Name
	if disc == "" {
		disc = fmt.Sprintf("Disc %d", m.Disc.Index)
	}
	fmt.Fprintf(&sb, " - %s", disc)
	if m.Disc.Format != "" {
		fmt.Fprintf(&sb, " [%s]", m.Disc.Format)
	}
	return sb.String()
}

const discDBTitleFields = `
	index
	name
	format
	contentHash
	titles(order: { index: ASC }) {
		index
		sourceFile
		segmentMap
		duration
		size
		item { title type season episode }
	}`

var discDBHashQuery = `query HandyMKVDiscByHash($hash: String) {
  mediaItems(where: { releases: { some: { discs: { some: { contentHash: { eq: $hash } } } } } }) {
    nodes {
      title
      year
      type
      releases(where: { discs: { some: { contentHash: { eq: $hash } } } }) {
        title
        year
        discs(where: { contentHash: { eq: $hash } }) {` + discDBTitleFields + `
        }
      }
    }
  }
  boxsets(where: { release: { discs: { some: { contentHash: { eq: $hash } } } } }) {
    nodes {
      title
      release {
        title
        year
        discs(where: { contentHash: { eq: $hash } }) {` + discDBTitleFields + `
        }
      }
    }
  }
}`

var discDBSearchQuery = `query HandyMKVSearch($term: String) {
  mediaItems(where: { title: { contains: $term } }, first: 15, order: { title: ASC }) {
    nodes {
      title
      year
      type
      releases(order: { year: ASC }) {
        title
        year
        discs(order: { index: ASC }) {` + discDBTitleFields + `
        }
      }
    }
  }
  boxsets(where: { title: { contains: $term } }, first: 10) {
    nodes {
      title
      release {
        title
        year
        discs(order: { index: ASC }) {` + discDBTitleFields + `
        }
      }
    }
  }
}`

func (c *DiscDBClient) query(ctx context.Context, query string, variables map[string]any) (*discDBQueryData, error) {
	body, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "ApiKey "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("TheDiscDB request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, fmt.Errorf("reading TheDiscDB response failed: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("TheDiscDB rejected the request (HTTP %d) - an API key may be required (config: thediscdb_api_key)", resp.StatusCode)
	}

	var result struct {
		Data   *discDBQueryData `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("TheDiscDB returned HTTP %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("could not parse TheDiscDB response: %w", err)
	}

	if len(result.Errors) > 0 {
		msgs := make([]string, 0, len(result.Errors))
		for _, e := range result.Errors {
			msgs = append(msgs, e.Message)
		}
		return nil, fmt.Errorf("TheDiscDB returned errors: %s", strings.Join(msgs, "; "))
	}

	if result.Data == nil {
		return nil, fmt.Errorf("TheDiscDB returned no data (HTTP %d)", resp.StatusCode)
	}

	return result.Data, nil
}

// flattenDiscDBMatches turns the nested GraphQL response into a flat list of discs.
// If onlyHash is set, discs with a different content hash are dropped.
func flattenDiscDBMatches(data *discDBQueryData, onlyHash string) []DiscDBMatch {
	matches := make([]DiscDBMatch, 0)

	keep := func(d discDBDisc) bool {
		return onlyHash == "" || strings.EqualFold(d.ContentHash, onlyHash)
	}

	if data.MediaItems != nil {
		for _, mi := range data.MediaItems.Nodes {
			for _, rel := range mi.Releases {
				for _, d := range rel.Discs {
					if !keep(d) {
						continue
					}
					matches = append(matches, DiscDBMatch{
						MediaTitle:   mi.Title,
						MediaYear:    mi.Year,
						MediaType:    mi.Type,
						ReleaseTitle: rel.Title,
						Disc:         d,
					})
				}
			}
		}
	}

	if data.Boxsets != nil {
		for _, bs := range data.Boxsets.Nodes {
			if bs.Release == nil {
				continue
			}
			for _, d := range bs.Release.Discs {
				if !keep(d) {
					continue
				}
				matches = append(matches, DiscDBMatch{
					MediaTitle:   bs.Title,
					MediaYear:    bs.Release.Year,
					MediaType:    "Boxset",
					ReleaseTitle: bs.Release.Title,
					Disc:         d,
				})
			}
		}
	}

	return matches
}

// LookupByHash returns all discs in TheDiscDB with the given content hash. The same disc can be
// part of several releases (e.g. a single season and a complete-series boxset).
func (c *DiscDBClient) LookupByHash(ctx context.Context, hash string) ([]DiscDBMatch, error) {
	hash = strings.ToUpper(hash)
	data, err := c.query(ctx, discDBHashQuery, map[string]any{"hash": hash})
	if err != nil {
		return nil, err
	}

	matches := flattenDiscDBMatches(data, hash)
	if len(matches) == 0 {
		return nil, ErrDiscDBNoMatch
	}

	return matches, nil
}

// Search returns all discs of movies, series and boxsets whose title contains the term.
func (c *DiscDBClient) Search(ctx context.Context, term string) ([]DiscDBMatch, error) {
	data, err := c.query(ctx, discDBSearchQuery, map[string]any{"term": term})
	if err != nil {
		return nil, err
	}

	return flattenDiscDBMatches(data, ""), nil
}

// ---------------------------------------------------------------------------
// Content hash
// ---------------------------------------------------------------------------

// ComputeDiscContentHash computes TheDiscDB's "content hash" of a mounted disc.
//
// It mirrors TheDiscDB's own implementation (DiscFileCatalog.ComputeContentHash in
// github.com/TheDiscDb/web): an MD5 over the sizes (int64, little endian) of
//   - Blu-ray/UHD: every BDMV/STREAM/*.m2ts file (not in subdirectories)
//   - DVD:         every VIDEO_TS/*.VOB, *.IFO and *.BUP file
//
// sorted by file name, rendered as upper case hex.
func ComputeDiscContentHash(root string) (string, error) {
	type file struct {
		name string
		size int64
	}

	collect := func(dir string, match func(name string) bool) ([]file, error) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		files := make([]file, 0, len(entries))
		for _, e := range entries {
			if e.IsDir() || !match(strings.ToLower(e.Name())) {
				continue
			}
			info, err := e.Info()
			if err != nil {
				return nil, err
			}
			files = append(files, file{name: e.Name(), size: info.Size()})
		}
		return files, nil
	}

	var files []file
	var err error

	if dir, ok := findDirCaseInsensitive(root, "BDMV", "STREAM"); ok {
		files, err = collect(dir, func(n string) bool { return strings.HasSuffix(n, ".m2ts") })
	} else if dir, ok := findDirCaseInsensitive(root, "VIDEO_TS"); ok {
		files, err = collect(dir, func(n string) bool {
			return strings.HasSuffix(n, ".vob") || strings.HasSuffix(n, ".ifo") || strings.HasSuffix(n, ".bup")
		})
	} else {
		return "", fmt.Errorf("no BDMV/STREAM or VIDEO_TS directory found in %s", root)
	}

	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no video files found on disc at %s", root)
	}

	// TheDiscDB sorts ordinally by the file name as stored on the disc, which is upper case for
	// DVDs and digits only for Blu-rays. Some systems present ISO9660 names in lower case, so
	// compare upper case names to get the same order everywhere.
	sort.Slice(files, func(i, j int) bool {
		return strings.ToUpper(files[i].name) < strings.ToUpper(files[j].name)
	})

	h := md5.New()
	buf := make([]byte, 8)
	for _, f := range files {
		binary.LittleEndian.PutUint64(buf, uint64(f.size))
		h.Write(buf)
	}

	return strings.ToUpper(hex.EncodeToString(h.Sum(nil))), nil
}

// findDirCaseInsensitive resolves a path below root, matching each element case-insensitively.
func findDirCaseInsensitive(root string, elems ...string) (string, bool) {
	cur := root
	for _, elem := range elems {
		entries, err := os.ReadDir(cur)
		if err != nil {
			return "", false
		}
		found := false
		for _, e := range entries {
			if e.IsDir() && strings.EqualFold(e.Name(), elem) {
				cur = filepath.Join(cur, e.Name())
				found = true
				break
			}
		}
		if !found {
			return "", false
		}
	}
	return cur, true
}

// ---------------------------------------------------------------------------
// Title matching
// ---------------------------------------------------------------------------

// normalizeDuration turns "1:02:03" / "01:02:03" into seconds; returns -1 if unparseable.
func normalizeDuration(d string) int {
	parts := strings.Split(strings.TrimSpace(d), ":")
	if len(parts) == 0 || len(parts) > 3 {
		return -1
	}
	total := 0
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return -1
		}
		total = total*60 + n
	}
	return total
}

func normalizeSegmentMap(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), " ", "")
}

// matchDiscDBTitle finds the TheDiscDB title that corresponds to a MakeMKV title.
//
// TheDiscDB records its titles from MakeMKV logs, so the strongest key is MakeMKV's source file
// (e.g. "00800.mpls" on Blu-ray, the title number on DVD). Playlists can be reused with different
// segment maps, so the segment map and the size break ties. Titles without a source file
// fall back to an exact size match.
func matchDiscDBTitle(t *TitleInfo, candidates []discDBTitle) *discDBTitle {
	var pool []*discDBTitle

	if t.SourceFile != "" {
		for i := range candidates {
			if strings.EqualFold(strings.TrimSpace(candidates[i].SourceFile), t.SourceFile) {
				pool = append(pool, &candidates[i])
			}
		}
	}

	if len(pool) == 0 {
		// No source file information - only accept an exact size match.
		if t.FileSizeBytes <= 0 {
			return nil
		}
		for i := range candidates {
			if candidates[i].Size == int64(t.FileSizeBytes) {
				pool = append(pool, &candidates[i])
			}
		}
		if len(pool) == 0 {
			return nil
		}
	}

	narrow := func(pred func(c *discDBTitle) bool) {
		if len(pool) < 2 {
			return
		}
		var next []*discDBTitle
		for _, c := range pool {
			if pred(c) {
				next = append(next, c)
			}
		}
		if len(next) > 0 {
			pool = next
		}
	}

	if t.SegmentMap != "" {
		narrow(func(c *discDBTitle) bool {
			return normalizeSegmentMap(c.SegmentMap) == normalizeSegmentMap(t.SegmentMap)
		})
	}
	if t.FileSizeBytes > 0 {
		narrow(func(c *discDBTitle) bool { return c.Size == int64(t.FileSizeBytes) })
	}
	if d := normalizeDuration(t.Length); d >= 0 {
		narrow(func(c *discDBTitle) bool { return normalizeDuration(c.Duration) == d })
	}
	narrow(func(c *discDBTitle) bool { return c.Index == t.Index })

	return pool[0]
}

// ApplyDiscDBMatch attaches TheDiscDB names to the MakeMKV titles. It returns the number of
// titles that received a name.
func ApplyDiscDBMatch(titles []TitleInfo, match DiscDBMatch) int {
	named := 0
	used := make(map[string]int)

	for i := range titles {
		titles[i].DiscDBName = ""
		titles[i].DiscDBType = ""
		titles[i].DiscDBFileName = ""

		dt := matchDiscDBTitle(&titles[i], match.Disc.Titles)
		if dt == nil || dt.Item == nil || strings.TrimSpace(dt.Item.Title) == "" {
			continue
		}

		titles[i].DiscDBName = discDBDisplayName(match, dt.Item)
		titles[i].DiscDBType = dt.Item.Type

		base := sanitizeFileName(discDBBaseName(match, dt.Item))
		if base == "" {
			continue
		}

		// Avoid two titles being renamed to the same file.
		key := strings.ToLower(base)
		used[key]++
		if used[key] > 1 {
			base = fmt.Sprintf("%s (%d)", base, used[key])
		}

		titles[i].DiscDBFileName = base + ".mkv"
		named++
	}

	return named
}

func formatSeasonEpisode(season, episode string) string {
	s, errS := strconv.Atoi(strings.TrimSpace(season))
	e, errE := strconv.Atoi(strings.TrimSpace(episode))
	if errS == nil && errE == nil {
		return fmt.Sprintf("S%02dE%02d", s, e)
	}
	if errE == nil && strings.TrimSpace(season) == "" {
		return fmt.Sprintf("E%02d", e)
	}
	if strings.TrimSpace(season) == "" && strings.TrimSpace(episode) == "" {
		return ""
	}
	return fmt.Sprintf("S%sE%s", strings.TrimSpace(season), strings.TrimSpace(episode))
}

// discDBDisplayName is the name shown in the title selection list.
func discDBDisplayName(match DiscDBMatch, item *discDBItem) string {
	name := discDBBaseName(match, item)
	if item.Type != "" && !strings.EqualFold(item.Type, "Episode") && !strings.EqualFold(item.Type, "MainMovie") {
		name = fmt.Sprintf("%s [%s]", name, item.Type)
	}
	return name
}

// discDBBaseName builds the output file name (without extension):
//
//	Episode:    "<Series> - S01E02 - <Episode title>"
//	Main movie: "<Movie title> (<Year>)"
//	Other:      "<Movie/Series> - <Item title>"
func discDBBaseName(match DiscDBMatch, item *discDBItem) string {
	itemTitle := strings.TrimSpace(item.Title)
	media := strings.TrimSpace(match.MediaTitle)

	switch {
	case strings.EqualFold(item.Type, "Episode"):
		se := formatSeasonEpisode(item.Season, item.Episode)
		parts := make([]string, 0, 3)
		if media != "" && !strings.EqualFold(media, itemTitle) {
			parts = append(parts, media)
		}
		if se != "" {
			parts = append(parts, se)
		}
		parts = append(parts, itemTitle)
		return strings.Join(parts, " - ")
	case strings.EqualFold(item.Type, "MainMovie"):
		if match.MediaYear > 0 && match.MediaType != "Boxset" {
			return fmt.Sprintf("%s (%d)", itemTitle, match.MediaYear)
		}
		return itemTitle
	default:
		if media != "" && !strings.EqualFold(media, itemTitle) {
			return fmt.Sprintf("%s - %s", media, itemTitle)
		}
		return itemTitle
	}
}

// sanitizeFileName removes characters that are not allowed in file names on Windows, macOS or
// Linux and trims the result to a sensible length.
func sanitizeFileName(name string) string {
	var sb strings.Builder
	for _, r := range name {
		switch {
		case r < 32:
			continue
		case strings.ContainsRune(`<>:"/\|?*`, r):
			if r == ':' {
				sb.WriteString(" -")
			} else {
				sb.WriteRune(' ')
			}
		default:
			sb.WriteRune(r)
		}
	}

	result := strings.Join(strings.Fields(sb.String()), " ")
	result = strings.TrimRight(result, ". ")

	const maxLen = 180
	if len(result) > maxLen {
		// cut on a rune boundary
		cut := 0
		for i := range result {
			if i > maxLen {
				break
			}
			cut = i
		}
		result = strings.TrimRight(result[:cut], ". ")
	}

	// Reserved device names on Windows
	switch strings.ToUpper(result) {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		result = "_" + result
	}

	return result
}

// ---------------------------------------------------------------------------
// Interactive identification
// ---------------------------------------------------------------------------

// identifyTitlesWithDiscDB looks the disc up in TheDiscDB and attaches the names to the titles.
// Every failure is non-fatal: the titles simply keep the MakeMKV names.
func identifyTitlesWithDiscDB(client *DiscDBClient, titles []TitleInfo) {
	if client == nil || len(titles) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*discDBRequestTimeout)
	defer cancel()

	fmt.Printf("Looking up disc in TheDiscDB...\n")

	match, err := lookupDiscByHash(ctx, client, titles[0].DevicePath)
	if err != nil {
		fmt.Printf("TheDiscDB: %v\n\n", err)
		match = searchDiscDBInteractive(client, titles[0].DiscTitle)
		if match == nil {
			fmt.Println()
			return
		}
	}

	named := ApplyDiscDBMatch(titles, *match)
	fmt.Printf("TheDiscDB: %s - %d of %d titles identified.\n\n", match.String(), named, len(titles))
}

func lookupDiscByHash(ctx context.Context, client *DiscDBClient, devicePath string) (*DiscDBMatch, error) {
	mountPoint, err := findDiscMountPoint(devicePath)
	if err != nil {
		return nil, fmt.Errorf("could not access the disc file system (%w)", err)
	}

	hash, err := ComputeDiscContentHash(mountPoint)
	if err != nil {
		return nil, fmt.Errorf("could not compute the disc hash (%w)", err)
	}

	matches, err := client.LookupByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrDiscDBNoMatch) {
			return nil, fmt.Errorf("no disc with hash %s found", hash)
		}
		return nil, err
	}

	// All matches share the same content hash and therefore the same titles. A disc can be part
	// of several releases; movie/series releases come first and give better names than boxsets.
	return &matches[0], nil
}

const maxDiscDBSearchResults = 40

func searchDiscDBInteractive(client *DiscDBClient, discTitle string) *DiscDBMatch {
	for {
		fmt.Printf("Enter a movie or series name to search TheDiscDB (leave empty to skip): ")
		term := readLine()
		if term == "" {
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), discDBRequestTimeout)
		matches, err := client.Search(ctx, term)
		cancel()

		if err != nil {
			fmt.Printf("\nTheDiscDB: %v\n\n", err)
			return nil
		}

		if len(matches) == 0 {
			fmt.Printf("\nNo results for '%s'.\n\n", term)
			continue
		}

		if len(matches) > maxDiscDBSearchResults {
			fmt.Printf("\n%d discs found, please use a more specific search term.\n\n", len(matches))
			continue
		}

		fmt.Printf("\nSelect the disc '%s' from TheDiscDB:\n\n", discTitle)
		fmt.Printf("0. None of these (search again)\n")
		for i, m := range matches {
			fmt.Printf("%d. %s\n", i+1, m.String())
		}
		fmt.Println()

		for {
			fmt.Printf("Selection: ")
			n, err := strconv.Atoi(readLine())
			if err != nil || n < 0 || n > len(matches) {
				fmt.Printf("Invalid selection.\n")
				continue
			}
			if n == 0 {
				fmt.Println()
				break
			}
			return &matches[n-1]
		}
	}
}
