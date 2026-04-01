package hmkv

import (
	"strings"
	"testing"
)

func TestStatusValueString(t *testing.T) {
	tests := []struct {
		status statusValue
		want   string
	}{
		{Pending, "Pending"},
		{InProgress, "In Progress"},
		{Complete, "Complete"},
		{statusValue(99), "Unknown"},
	}

	for _, tt := range tests {
		got := tt.status.String()
		if got != tt.want {
			t.Errorf("statusValue(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestGetColor(t *testing.T) {
	tests := []struct {
		status statusValue
		want   string
	}{
		{Pending, colorYellow},
		{InProgress, colorBlue},
		{Complete, colorGreen},
		{statusValue(99), colorReset},
	}

	for _, tt := range tests {
		got := getColor(tt.status)
		if got != tt.want {
			t.Errorf("getColor(%d) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestGetAnimatedEllipsis(t *testing.T) {
	tests := []struct {
		frame int
		want  string
	}{
		{0, ".  "},
		{1, ".. "},
		{2, "..."},
		{3, ".  "}, // default
		{-1, ".  "}, // default
	}

	for _, tt := range tests {
		got := getAnimatedEllipsis(tt.frame)
		if got != tt.want {
			t.Errorf("getAnimatedEllipsis(%d) = %q, want %q", tt.frame, got, tt.want)
		}
	}
}

func TestFormatProgressStatus(t *testing.T) {
	tests := []struct {
		status   statusValue
		progress int
		frame    int
		contains []string // substrings that must appear in output
	}{
		{Pending, 0, 0, []string{"Pending"}},
		{Complete, 0, 0, []string{"Complete"}},
		{InProgress, 50, 0, []string{"50%", ".  "}},
		{InProgress, 100, 2, []string{"100%", "..."}},
		{InProgress, -1, 1, []string{"Working", ".. "}},
	}

	for _, tt := range tests {
		got := formatProgressStatus(tt.status, tt.progress, tt.frame)
		for _, sub := range tt.contains {
			if !strings.Contains(got, sub) {
				t.Errorf("formatProgressStatus(%v, %d, %d) = %q, want it to contain %q",
					tt.status, tt.progress, tt.frame, got, sub)
			}
		}
	}
}
