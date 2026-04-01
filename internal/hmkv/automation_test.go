package hmkv

import (
	"fmt"
	"testing"
	"time"
)

func TestValidateAutomationName(t *testing.T) {
	valid := []string{"backup", "test-123", "my_script", "notify", "a"}
	for _, name := range valid {
		if err := validateAutomationName(name); err != nil {
			t.Errorf("validateAutomationName(%q) returned unexpected error: %v", name, err)
		}
	}

	invalid := []string{
		"my/script",   // forward slash
		"my\\script",  // backslash
		"my..script",  // double dot
		"../escape",   // parent dir traversal
		"..\\escape",  // Windows parent dir traversal
	}
	for _, name := range invalid {
		if err := validateAutomationName(name); err == nil {
			t.Errorf("validateAutomationName(%q) expected error, got nil", name)
		}
	}
}

func TestBuildRunOutputData(t *testing.T) {
	hbDir := "/out/hb"
	mkvDir := "/out/mkv"
	duration := 65 * time.Second // 1m5s
	titleCount := 3
	totalRaw := int64(1024 * 1024 * 500) // 500 MB
	totalEncoded := int64(1024 * 1024 * 200) // 200 MB

	t.Run("rawDeleted=true", func(t *testing.T) {
		data := buildRunOutputData(hbDir, mkvDir, duration, titleCount, true, totalRaw, totalEncoded)

		if data["hb_output_dir"] != hbDir {
			t.Errorf("hb_output_dir = %q, want %q", data["hb_output_dir"], hbDir)
		}
		if data["mkv_output_dir"] != mkvDir {
			t.Errorf("mkv_output_dir = %q, want %q", data["mkv_output_dir"], mkvDir)
		}
		if data["raw_files_deleted"] != "true" {
			t.Errorf("raw_files_deleted = %q, want %q", data["raw_files_deleted"], "true")
		}
		if data["run_duration"] != "1m5s" {
			t.Errorf("run_duration = %q, want %q", data["run_duration"], "1m5s")
		}
		if data["title_count"] != fmt.Sprintf("%d", titleCount) {
			t.Errorf("title_count = %q, want %q", data["title_count"], fmt.Sprintf("%d", titleCount))
		}
		if data["total_raw_size"] != fmt.Sprintf("%d", totalRaw) {
			t.Errorf("total_raw_size = %q, want %q", data["total_raw_size"], fmt.Sprintf("%d", totalRaw))
		}
		if data["total_encoded_size"] != fmt.Sprintf("%d", totalEncoded) {
			t.Errorf("total_encoded_size = %q, want %q", data["total_encoded_size"], fmt.Sprintf("%d", totalEncoded))
		}

		// Verify all 7 expected keys are present
		expectedKeys := []string{
			"hb_output_dir", "mkv_output_dir", "run_duration",
			"raw_files_deleted", "title_count", "total_raw_size", "total_encoded_size",
		}
		for _, key := range expectedKeys {
			if _, ok := data[key]; !ok {
				t.Errorf("missing key %q in buildRunOutputData result", key)
			}
		}
	})

	t.Run("rawDeleted=false", func(t *testing.T) {
		data := buildRunOutputData(hbDir, mkvDir, duration, titleCount, false, totalRaw, totalEncoded)
		if data["raw_files_deleted"] != "false" {
			t.Errorf("raw_files_deleted = %q, want %q", data["raw_files_deleted"], "false")
		}
	})
}
