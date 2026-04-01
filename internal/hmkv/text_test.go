package hmkv

import (
	"testing"
	"time"
)

func TestFormatTimeElapsedString(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{0, "0m0s"},
		{time.Second, "0m1s"},
		{65 * time.Second, "1m5s"},
		{60 * time.Second, "1m0s"},
		{3661 * time.Second, "61m1s"},
		{2*time.Hour + 3*time.Minute + 4*time.Second, "123m4s"},
	}

	for _, tt := range tests {
		got := formatTimeElapsedString(tt.duration)
		if got != tt.want {
			t.Errorf("formatTimeElapsedString(%v) = %q, want %q", tt.duration, got, tt.want)
		}
	}
}

func TestFormatSavedSpace(t *testing.T) {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 Bytes"},
		{1, "1 Bytes"},
		{512, "512 Bytes"},
		{KB - 1, "1023 Bytes"},
		{KB, "1.00 KB"},
		{int64(1.5 * KB), "1.50 KB"},
		{MB - 1, "1024.00 KB"},
		{MB, "1.00 MB"},
		{int64(2.5 * MB), "2.50 MB"},
		{GB, "1.00 GB"},
		{int64(3.75 * GB), "3.75 GB"},
	}

	for _, tt := range tests {
		got := formatSavedSpace(tt.bytes)
		if got != tt.want {
			t.Errorf("formatSavedSpace(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}
