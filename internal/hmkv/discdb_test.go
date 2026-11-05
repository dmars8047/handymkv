package hmkv

import (
	"bufio"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSizedFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0644); err != nil {
		t.Fatal(err)
	}
}

func md5OfSizes(sizes ...int64) string {
	h := md5.New()
	buf := make([]byte, 8)
	for _, s := range sizes {
		binary.LittleEndian.PutUint64(buf, uint64(s))
		h.Write(buf)
	}
	return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
}

// Mirrors TheDiscDB's test GenerateAsync_ContentHashUsesOrdinalBareNamesAndExcludesBackupAndSsifFiles.
func TestComputeDiscContentHashBluRay(t *testing.T) {
	root := t.TempDir()
	writeSizedFile(t, filepath.Join(root, "BDMV", "STREAM", "2.m2ts"), 2)
	writeSizedFile(t, filepath.Join(root, "BDMV", "STREAM", "10.M2TS"), 1)
	writeSizedFile(t, filepath.Join(root, "BDMV", "BACKUP", "STREAM", "01.m2ts"), 99)
	writeSizedFile(t, filepath.Join(root, "BDMV", "STREAM", "00001.ssif"), 77)
	writeSizedFile(t, filepath.Join(root, "BDMV", "PLAYLIST", "00800.mpls"), 5)

	got, err := ComputeDiscContentHash(root)
	if err != nil {
		t.Fatal(err)
	}

	if want := md5OfSizes(1, 2); got != want {
		t.Errorf("hash = %s, want %s", got, want)
	}
}

func TestComputeDiscContentHashDVD(t *testing.T) {
	root := t.TempDir()
	// lower case names, as some systems present ISO9660 file systems
	writeSizedFile(t, filepath.Join(root, "video_ts", "vts_01_1.vob"), 30)
	writeSizedFile(t, filepath.Join(root, "video_ts", "video_ts.ifo"), 10)
	writeSizedFile(t, filepath.Join(root, "video_ts", "video_ts.bup"), 11)
	writeSizedFile(t, filepath.Join(root, "video_ts", "vts_01_0.ifo"), 20)
	writeSizedFile(t, filepath.Join(root, "video_ts", "readme.txt"), 99)

	got, err := ComputeDiscContentHash(root)
	if err != nil {
		t.Fatal(err)
	}

	// VIDEO_TS.BUP, VIDEO_TS.IFO, VTS_01_0.IFO, VTS_01_1.VOB
	if want := md5OfSizes(11, 10, 20, 30); got != want {
		t.Errorf("hash = %s, want %s", got, want)
	}
}

func TestComputeDiscContentHashNoVideo(t *testing.T) {
	if _, err := ComputeDiscContentHash(t.TempDir()); err == nil {
		t.Error("expected an error for an empty directory")
	}
}

func TestMountPointFromProcMounts(t *testing.T) {
	content := "sysfs /sys sysfs rw 0 0\n/dev/sr0 /media/user/STAR\\040TREK udf ro 0 0\n"
	got, err := mountPointFromProcMounts(bufio.NewScanner(strings.NewReader(content)), "/dev/sr0")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/media/user/STAR TREK" {
		t.Errorf("got %q", got)
	}

	if _, err := mountPointFromProcMounts(bufio.NewScanner(strings.NewReader(content)), "/dev/sr1"); err == nil {
		t.Error("expected an error for an unmounted drive")
	}
}

func sampleMatch() DiscDBMatch {
	return DiscDBMatch{
		MediaTitle:   "Star Trek: The Next Generation",
		MediaYear:    1987,
		MediaType:    "Series",
		ReleaseTitle: "Season 4",
		Disc: discDBDisc{
			Index: 2,
			Name:  "Disc 2",
			Titles: []discDBTitle{
				{Index: 0, SourceFile: "00800.mpls", SegmentMap: "1,2", Size: 1000, Duration: "0:45:10", Item: &discDBItem{Title: "Remember Me", Type: "Episode", Season: "4", Episode: "5"}},
				{Index: 1, SourceFile: "00801.mpls", Size: 2000, Duration: "0:45:20", Item: &discDBItem{Title: "Legacy", Type: "Episode", Season: "4", Episode: "6"}},
				{Index: 2, SourceFile: "00800.mpls", SegmentMap: "3,4", Size: 3000, Duration: "0:44:00", Item: &discDBItem{Title: "Reunion", Type: "Episode", Season: "4", Episode: "7"}},
				{Index: 3, SourceFile: "00900.mpls", Size: 500, Item: &discDBItem{Title: "Making of: Season 4", Type: "Extra"}},
				{Index: 4, SourceFile: "00901.mpls", Size: 400},
			},
		},
	}
}

func TestApplyDiscDBMatch(t *testing.T) {
	titles := []TitleInfo{
		{Index: 0, FileName: "title_t00.mkv", SourceFile: "00800.mpls", SegmentMap: "3,4", FileSizeBytes: 3000},
		{Index: 1, FileName: "title_t01.mkv", SourceFile: "00801.mpls", FileSizeBytes: 2000},
		{Index: 2, FileName: "title_t02.mkv", SourceFile: "00800.mpls", SegmentMap: "1,2", FileSizeBytes: 1000},
		{Index: 3, FileName: "title_t03.mkv", SourceFile: "00900.mpls", FileSizeBytes: 500},
		{Index: 4, FileName: "title_t04.mkv", SourceFile: "00901.mpls", FileSizeBytes: 400},
		{Index: 5, FileName: "title_t05.mkv", SourceFile: "00999.mpls", FileSizeBytes: 1},
	}

	named := ApplyDiscDBMatch(titles, sampleMatch())
	if named != 4 {
		t.Errorf("named = %d, want 4", named)
	}

	want := []string{
		"Star Trek - The Next Generation - S04E07 - Reunion.mkv",
		"Star Trek - The Next Generation - S04E06 - Legacy.mkv",
		"Star Trek - The Next Generation - S04E05 - Remember Me.mkv",
		"Star Trek - The Next Generation - Making of - Season 4.mkv",
		"title_t04.mkv",
		"title_t05.mkv",
	}
	for i, w := range want {
		if got := titles[i].OutputFileName(); got != w {
			t.Errorf("title %d: OutputFileName() = %q, want %q", i, got, w)
		}
	}

	if titles[0].DiscDBName != "Star Trek: The Next Generation - S04E07 - Reunion" {
		t.Errorf("DiscDBName = %q", titles[0].DiscDBName)
	}
	if titles[3].DiscDBName != "Star Trek: The Next Generation - Making of: Season 4 [Extra]" {
		t.Errorf("DiscDBName = %q", titles[3].DiscDBName)
	}
	if got := titles[0].ProgressName(); got != "S04E07 - Reunion" {
		t.Errorf("ProgressName() = %q", got)
	}
	if got := titles[3].ProgressName(); got != "Making of: Season 4" {
		t.Errorf("ProgressName() = %q", got)
	}
	if got := titles[4].ProgressName(); got != "title_t04" {
		t.Errorf("ProgressName() = %q", got)
	}
	if titles[4].DiscDBName != "" {
		t.Errorf("unexpected DiscDBName %q", titles[4].DiscDBName)
	}
}

func TestApplyDiscDBMatchMovieAndDuplicates(t *testing.T) {
	match := DiscDBMatch{
		MediaTitle: "Heat",
		MediaYear:  1995,
		MediaType:  "Movie",
		Disc: discDBDisc{Titles: []discDBTitle{
			{Index: 0, SourceFile: "00003.mpls", Size: 10, Item: &discDBItem{Title: "Heat", Type: "MainMovie"}},
			{Index: 1, SourceFile: "00011.mpls", Size: 20, Item: &discDBItem{Title: "Trailer", Type: "Trailer"}},
			{Index: 2, SourceFile: "00012.mpls", Size: 30, Item: &discDBItem{Title: "Trailer", Type: "Trailer"}},
		}},
	}
	titles := []TitleInfo{
		{Index: 0, FileName: "a.mkv", SourceFile: "00003.mpls"},
		{Index: 1, FileName: "b.mkv", SourceFile: "00011.mpls"},
		{Index: 2, FileName: "c.mkv", SourceFile: "00012.mpls"},
	}
	ApplyDiscDBMatch(titles, match)

	want := []string{"Heat (1995).mkv", "Heat - Trailer.mkv", "Heat - Trailer (2).mkv"}
	for i, w := range want {
		if got := titles[i].OutputFileName(); got != w {
			t.Errorf("title %d = %q, want %q", i, got, w)
		}
	}
}

func TestMatchDiscDBTitleFallsBackToSize(t *testing.T) {
	candidates := sampleMatch().Disc.Titles
	got := matchDiscDBTitle(&TitleInfo{Index: 9, FileSizeBytes: 2000}, candidates)
	if got == nil || got.Index != 1 {
		t.Errorf("expected match on size, got %+v", got)
	}
	if got := matchDiscDBTitle(&TitleInfo{Index: 9, FileSizeBytes: 12345}, candidates); got != nil {
		t.Errorf("expected no match, got %+v", got)
	}
}

func TestSanitizeFileName(t *testing.T) {
	tests := map[string]string{
		`Who? What/When "Now"`: "Who What When Now",
		"Title: Subtitle":      "Title - Subtitle",
		"Ends with dot.":       "Ends with dot",
		"CON":                  "_CON",
		"  many   spaces  ":    "many spaces",
	}
	for in, want := range tests {
		if got := sanitizeFileName(in); got != want {
			t.Errorf("sanitizeFileName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDiscDBClientLookupByHash(t *testing.T) {
	var gotAuth, gotHash string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		_ = json.Unmarshal(body, &req)
		gotHash, _ = req.Variables["hash"].(string)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{
			"mediaItems":{"nodes":[{"title":"Heat","year":1995,"type":"Movie","releases":[
				{"title":"Blu-ray 2009","year":2009,"discs":[
					{"index":1,"name":"Blu-ray","format":"Blu-ray","contentHash":"ABC","titles":[
						{"index":0,"sourceFile":"00003.mpls","segmentMap":"1","duration":"2:50:00","size":123,"item":{"title":"Heat","type":"MainMovie","season":null,"episode":null}}
					]}
				]}
			]}]},
			"boxsets":{"nodes":[]}
		}}`))
	}))
	defer srv.Close()

	c := NewDiscDBClient("secret", "handymkv/test")
	c.Endpoint = srv.URL

	matches, err := c.LookupByHash(context.Background(), "abc")
	if err != nil {
		t.Fatal(err)
	}

	if gotHash != "ABC" {
		t.Errorf("hash sent = %q, want upper case ABC", gotHash)
	}
	if gotAuth != "ApiKey secret" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if len(matches) != 1 || matches[0].MediaTitle != "Heat" || len(matches[0].Disc.Titles) != 1 {
		t.Fatalf("unexpected matches %+v", matches)
	}
	if matches[0].Disc.Titles[0].Item.Type != "MainMovie" {
		t.Errorf("unexpected item %+v", matches[0].Disc.Titles[0].Item)
	}
}

func TestDiscDBClientNoMatchAndErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/err" {
			_, _ = w.Write([]byte(`{"errors":[{"message":"boom"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"mediaItems":{"nodes":[]},"boxsets":{"nodes":[]}}}`))
	}))
	defer srv.Close()

	c := NewDiscDBClient("", "")
	c.Endpoint = srv.URL
	if _, err := c.LookupByHash(context.Background(), "X"); err != ErrDiscDBNoMatch {
		t.Errorf("err = %v, want ErrDiscDBNoMatch", err)
	}

	c.Endpoint = srv.URL + "/err"
	if _, err := c.LookupByHash(context.Background(), "X"); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("err = %v, want GraphQL error", err)
	}
}
