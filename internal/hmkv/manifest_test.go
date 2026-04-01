package hmkv

import (
	"os"
	"testing"
	"time"
)

func TestNormalizeHHMMSS(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1:02:03", "01:02:03"},
		{"0:0:0", "00:00:00"},
		{"23:59:59", "23:59:59"},
		{"10:10:10", "10:10:10"},
		{"1:1:1", "01:01:01"},
		// Invalid inputs returned unchanged
		{"bad", "bad"},
		{"a:b:c", "a:b:c"},
		{"", ""},
		{"1:2", "1:2"},         // only 2 parts
		{"1:2:3:4", "1:2:3:4"}, // 4 parts
	}

	for _, tt := range tests {
		got := normalizeHHMMSS(tt.input)
		if got != tt.want {
			t.Errorf("normalizeHHMMSS(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBuildManifestEmpty(t *testing.T) {
	m := buildManifest(nil, nil, time.Now(), 0, false, "1.0.0", nil)
	if m == nil {
		t.Fatal("buildManifest returned nil")
	}
	if len(m.Discs) != 0 {
		t.Errorf("expected 0 discs, got %d", len(m.Discs))
	}
	if m.AppVersion != "1.0.0" {
		t.Errorf("AppVersion = %q, want %q", m.AppVersion, "1.0.0")
	}
}

func TestBuildManifestSingleDisc(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	duration := 5 * time.Minute

	processTitles := []TitleInfo{
		{Index: 0, DiscId: 0, DiscTitle: "My Movie", Length: "1:30:00"},
		{Index: 1, DiscId: 0, DiscTitle: "My Movie", Length: "0:5:00"},
	}

	entries := []EncodingParams{
		{TitleIndex: 0, DiscId: 0, MKVOutputPath: "/tmp/t0.mkv", HandBrakeOutputPath: "/out/t0.mp4"},
		{TitleIndex: 1, DiscId: 0, MKVOutputPath: "/tmp/t1.mkv", HandBrakeOutputPath: "/out/t1.mp4"},
	}

	m := buildManifest(processTitles, entries, start, duration, true, "1.2.3", nil)

	if len(m.Discs) != 1 {
		t.Fatalf("expected 1 disc, got %d", len(m.Discs))
	}

	disc := m.Discs[0]
	if disc.DiscId != 0 {
		t.Errorf("DiscId = %d, want 0", disc.DiscId)
	}
	if disc.DiscName != "My Movie" {
		t.Errorf("DiscName = %q, want %q", disc.DiscName, "My Movie")
	}
	if len(disc.Titles) != 2 {
		t.Fatalf("expected 2 titles, got %d", len(disc.Titles))
	}

	// Duration should be normalized
	if disc.Titles[0].MediaDuration != "01:30:00" {
		t.Errorf("Titles[0].MediaDuration = %q, want %q", disc.Titles[0].MediaDuration, "01:30:00")
	}
	if disc.Titles[1].MediaDuration != "00:05:00" {
		t.Errorf("Titles[1].MediaDuration = %q, want %q", disc.Titles[1].MediaDuration, "00:05:00")
	}

	if !m.RawFilesDeleted {
		t.Error("RawFilesDeleted = false, want true")
	}
	if m.AppVersion != "1.2.3" {
		t.Errorf("AppVersion = %q, want %q", m.AppVersion, "1.2.3")
	}
}

func TestBuildManifestDiscOrder(t *testing.T) {
	// disc order should follow processTitles insertion order
	processTitles := []TitleInfo{
		{Index: 0, DiscId: 1, DiscTitle: "Disc One", Length: "1:0:0"},
		{Index: 0, DiscId: 0, DiscTitle: "Disc Zero", Length: "2:0:0"},
	}
	entries := []EncodingParams{
		{TitleIndex: 0, DiscId: 1, MKVOutputPath: "/a.mkv"},
		{TitleIndex: 0, DiscId: 0, MKVOutputPath: "/b.mkv"},
	}

	m := buildManifest(processTitles, entries, time.Now(), 0, false, "v", nil)

	if len(m.Discs) != 2 {
		t.Fatalf("expected 2 discs, got %d", len(m.Discs))
	}
	if m.Discs[0].DiscId != 1 {
		t.Errorf("Discs[0].DiscId = %d, want 1 (insertion order)", m.Discs[0].DiscId)
	}
	if m.Discs[1].DiscId != 0 {
		t.Errorf("Discs[1].DiscId = %d, want 0 (insertion order)", m.Discs[1].DiscId)
	}
}

func TestWriteAndReadManifestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	start := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)

	original := &manifest{
		AppVersion:      "1.0.0",
		Date:            start,
		Duration:        "5m0s",
		RawFilesDeleted: true,
		Discs: []manifestDisc{
			{
				DiscId:   0,
				DiscName: "Test Disc",
				Titles: []manifestTitle{
					{TitleIndex: 0, MediaDuration: "01:30:00", RippedFile: "/tmp/a.mkv", EncodedFile: "/out/a.mp4"},
				},
			},
		},
	}

	path, err := writeManifest(dir, start, original)
	if err != nil {
		t.Fatalf("writeManifest error: %v", err)
	}

	got, err := readManifestFile(path)
	if err != nil {
		t.Fatalf("readManifestFile error: %v", err)
	}

	if got.AppVersion != original.AppVersion {
		t.Errorf("AppVersion = %q, want %q", got.AppVersion, original.AppVersion)
	}
	if got.RawFilesDeleted != original.RawFilesDeleted {
		t.Errorf("RawFilesDeleted = %v, want %v", got.RawFilesDeleted, original.RawFilesDeleted)
	}
	if len(got.Discs) != 1 {
		t.Fatalf("expected 1 disc, got %d", len(got.Discs))
	}
	if got.Discs[0].DiscName != "Test Disc" {
		t.Errorf("DiscName = %q, want %q", got.Discs[0].DiscName, "Test Disc")
	}
	if len(got.Discs[0].Titles) != 1 {
		t.Fatalf("expected 1 title, got %d", len(got.Discs[0].Titles))
	}
	if got.Discs[0].Titles[0].MediaDuration != "01:30:00" {
		t.Errorf("MediaDuration = %q, want %q", got.Discs[0].Titles[0].MediaDuration, "01:30:00")
	}
}

func TestReadManifestFileInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	badFile := dir + "/bad.json"

	if err := os.WriteFile(badFile, []byte("not json"), 0640); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := readManifestFile(badFile)
	if err == nil {
		t.Error("readManifestFile expected error for invalid JSON, got nil")
	}
}

func TestReadManifestFileMissing(t *testing.T) {
	_, err := readManifestFile("/nonexistent/path/manifest.json")
	if err == nil {
		t.Error("readManifestFile expected error for missing file, got nil")
	}
}
