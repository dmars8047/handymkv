package hmkv

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

type manifestTitle struct {
	TitleIndex           int    `json:"title_index"`
	MediaDuration        string `json:"media_duration_raw"`
	RippingDuration      string `json:"ripping_duration"`
	RippedFile           string `json:"ripped_file"`
	RippedFileSizeBytes  int64  `json:"ripped_file_size_bytes"`
	EncodedFile          string `json:"encoded_file"`
	EncodedFileSizeBytes int64  `json:"encoded_file_size_bytes"`
}

type manifestDisc struct {
	DiscId   int             `json:"disc_id"`
	DiscName string          `json:"disc_name"`
	Titles   []manifestTitle `json:"titles"`
}

type manifestAutomationParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type manifestAutomation struct {
	Name     string                    `json:"name"`
	Command  string                    `json:"command"`
	Params   []manifestAutomationParam `json:"params,omitempty"`
	ExitCode int                       `json:"exit_code"`
}

type manifest struct {
	AppVersion      string                `json:"app_version"`
	Date            time.Time             `json:"date"`
	Duration        string                `json:"duration"`
	RawFilesDeleted bool                  `json:"raw_files_deleted"`
	Discs           []manifestDisc        `json:"discs"`
	Automations     []manifestAutomation  `json:"automations,omitempty"`
}

// getManifestDir returns ~/.config/handymkv/manifests/ on Unix or %APPDATA%\handymkv\manifests\ on Windows.
// Always returns the user-level path regardless of config source.
func getManifestDir() (string, error) {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable is not set")
		}
		return filepath.Join(appData, "handymkv", "manifests"), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "handymkv", "manifests"), nil
}

// normalizeHHMMSS parses MakeMKV's "H:MM:SS" duration format and returns a
// zero-padded "HH:MM:SS" string. Returns the input unchanged if it cannot be parsed.
func normalizeHHMMSS(s string) string {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return s
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	sec, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return s
	}
	return fmt.Sprintf("%02d:%02d:%02d", h, m, sec)
}

// buildManifest groups entries by disc (preserving order from processTitles) and builds a manifest struct.
func buildManifest(processTitles []TitleInfo, entries []EncodingParams, startTime time.Time, duration time.Duration, rawDeleted bool, appVersion string, automations []manifestAutomation) *manifest {
	// Build disc name and duration lookups from processTitles
	discNames := make(map[int]string)
	type discTitle struct{ discId, titleIndex int }
	durations := make(map[discTitle]string)
	// Preserve disc order from processTitles
	var discOrder []int
	seen := make(map[int]bool)
	for _, t := range processTitles {
		discNames[t.DiscId] = t.DiscTitle
		if !seen[t.DiscId] {
			discOrder = append(discOrder, t.DiscId)
			seen[t.DiscId] = true
		}
		durations[discTitle{t.DiscId, t.Index}] = normalizeHHMMSS(t.Length)
	}

	// Group entries by disc id, preserving insertion order
	discTitles := make(map[int][]manifestTitle)
	for _, e := range entries {
		discTitles[e.DiscId] = append(discTitles[e.DiscId], manifestTitle{
			TitleIndex:           e.TitleIndex,
			MediaDuration:        durations[discTitle{e.DiscId, e.TitleIndex}],
			RippingDuration:      e.RippingDuration,
			RippedFile:           e.MKVOutputPath,
			RippedFileSizeBytes:  e.RippedFileSizeBytes,
			EncodedFile:          e.HandBrakeOutputPath,
			EncodedFileSizeBytes: e.EncodedFileSizeBytes,
		})
	}

	var discs []manifestDisc
	for _, discId := range discOrder {
		titles := discTitles[discId]
		if len(titles) == 0 {
			continue
		}
		discs = append(discs, manifestDisc{
			DiscId:   discId,
			DiscName: discNames[discId],
			Titles:   titles,
		})
	}

	return &manifest{
		AppVersion:      appVersion,
		Date:            startTime.UTC(),
		Duration:        duration.String(),
		RawFilesDeleted: rawDeleted,
		Discs:           discs,
		Automations:     automations,
	}
}

// writeManifest JSON-marshals the manifest, ensures the directory exists, and writes the file.
// Returns the path written.
func writeManifest(dir string, startTime time.Time, m *manifest) (string, error) {
	if err := os.MkdirAll(dir, 0740); err != nil {
		return "", fmt.Errorf("could not create manifest directory: %w", err)
	}

	fileName := fmt.Sprintf("manifest_%s.json", startTime.Format("2006-01-02_15-04-05"))
	filePath := filepath.Join(dir, fileName)

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", fmt.Errorf("could not marshal manifest: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0640); err != nil {
		return "", fmt.Errorf("could not write manifest file: %w", err)
	}

	return filePath, nil
}

// resolveManifestDir returns the manifest directory from config (if available)
// or falls back to getManifestDir().
func resolveManifestDir() (string, error) {
	cfg, err := ReadConfig()
	if err == nil && cfg.ManifestDirectory != "" {
		return cfg.ManifestDirectory, nil
	}
	return getManifestDir()
}

// ClearHistory prompts the user and deletes all manifest files from the manifest directory.
func ClearHistory() error {
	manifestDir, err := resolveManifestDir()
	if err != nil {
		return fmt.Errorf("could not determine manifest directory: %w", err)
	}

	pattern := filepath.Join(manifestDir, "manifest_*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("error scanning manifest directory: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No run history to clear.")
		return nil
	}

	fmt.Printf("This will permanently delete %d manifest file(s) from:\n  %s\n\n", len(files), manifestDir)
	fmt.Printf("Are you sure? [y/N]: ")

	choice := readLine()

	if strings.ToLower(choice) != "y" {
		fmt.Println("Aborted.")
		return nil
	}

	var deleteErrors []string
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			deleteErrors = append(deleteErrors, fmt.Sprintf("%s: %v", f, err))
		}
	}

	if len(deleteErrors) > 0 {
		return fmt.Errorf("errors deleting files:\n%s", strings.Join(deleteErrors, "\n"))
	}

	fmt.Printf("Cleared %d manifest file(s).\n", len(files))
	return nil
}

// PrintHistory prints a summary list (index == -1) or detail view (index > 0) of past runs.
func PrintHistory(index int) error {
	dir, err := resolveManifestDir()
	if err != nil {
		return fmt.Errorf("could not determine manifest directory: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No run history found.")
			return nil
		}
		return fmt.Errorf("could not read manifest directory: %w", err)
	}

	// Filter to manifest JSON files and sort newest-first
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "manifest_") && strings.HasSuffix(e.Name(), ".json") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}

	if len(files) == 0 {
		fmt.Println("No run history found.")
		return nil
	}

	// Sort newest-first (filenames are timestamped, so lexicographic descending works)
	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	if index == -1 {
		// Summary list
		fmt.Printf("Run History\n\n")
		fmt.Printf("  %-4s %-22s %-12s %-14s %s\n", "#", "Date", "Duration", "Raw Deleted", "Discs")

		for i, path := range files {
			m, err := readManifestFile(path)
			if err != nil {
				fmt.Printf("  %-4d (error reading manifest: %v)\n", i+1, err)
				continue
			}

			dateStr := m.Date.Local().Format("2006-01-02 15:04:05")
			rawDeleted := "No"
			if m.RawFilesDeleted {
				rawDeleted = "Yes"
			}

			var discSummaries []string
			for _, d := range m.Discs {
				discSummaries = append(discSummaries, fmt.Sprintf("%s (%d titles)", d.DiscName, len(d.Titles)))
			}

			fmt.Printf("  %-4d %-22s %-12s %-14s %s\n", i+1, dateStr, m.Duration, rawDeleted, strings.Join(discSummaries, ", "))
		}

		fmt.Printf("\nRun 'handymkv history <number>' to view details for a specific run.\n\n")
		return nil
	}

	// Detail view
	if index < 1 || index > len(files) {
		fmt.Printf("Invalid run number %d. Valid range: 1-%d.\n\n", index, len(files))
		return nil
	}

	m, err := readManifestFile(files[index-1])
	if err != nil {
		return fmt.Errorf("could not read manifest: %w", err)
	}

	rawDeleted := "No"
	if m.RawFilesDeleted {
		rawDeleted = "Yes"
	}

	fmt.Printf("Run #%d — %s\n", index, m.Date.Local().Format("2006-01-02 15:04:05"))
	fmt.Printf("App Version:       %s\n", m.AppVersion)
	fmt.Printf("Duration:          %s\n", m.Duration)
	fmt.Printf("Raw Files Deleted: %s\n\n", rawDeleted)

	for _, d := range m.Discs {
		fmt.Printf("Disc %d: %s\n", d.DiscId, d.DiscName)
		for _, t := range d.Titles {
			fmt.Printf("  [%d] %s\n", t.TitleIndex, filepath.Base(t.RippedFile))
			fmt.Printf("    Media Duration: %s\n", t.MediaDuration)
			fmt.Printf("    Rip Duration:   %s\n", t.RippingDuration)
			fmt.Printf("    Raw File:       %s (%s)\n", t.RippedFile, formatSavedSpace(t.RippedFileSizeBytes))
			fmt.Printf("    Encoded File:   %s (%s)\n", t.EncodedFile, formatSavedSpace(t.EncodedFileSizeBytes))
		}
	}

	if len(m.Automations) > 0 {
		fmt.Printf("\nAutomations:\n")
		for _, a := range m.Automations {
			outcome := "OK"
			if a.ExitCode != 0 {
				outcome = fmt.Sprintf("FAILED (exit code %d)", a.ExitCode)
			}
			fmt.Printf("  %s (%s) — %s\n", a.Name, a.Command, outcome)
			for _, p := range a.Params {
				fmt.Printf("    %s = %s\n", p.Name, p.Value)
			}
		}
	}

	fmt.Println()
	return nil
}

func readManifestFile(path string) (*manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
