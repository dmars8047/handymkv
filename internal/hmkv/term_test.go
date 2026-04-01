package hmkv

import (
	"strings"
	"testing"
)

func TestPadString(t *testing.T) {
	tests := []struct {
		name        string
		s           string
		width       int
		wantTooLong bool
		// wantSuffix: the trailing spaces expected (for short strings)
		wantLen int // expected len of returned string (rune count irrelevant; byte count for ASCII)
	}{
		{
			name:        "short plain string padded",
			s:           "hello",
			width:       10,
			wantTooLong: false,
			wantLen:     10,
		},
		{
			name:        "exact width plain string",
			s:           "hello",
			width:       5,
			wantTooLong: false,
			wantLen:     5,
		},
		{
			name:        "longer than width",
			s:           "hello world",
			width:       5,
			wantTooLong: true,
			wantLen:     len("hello world"),
		},
		{
			name:        "empty string padded",
			s:           "",
			width:       4,
			wantTooLong: false,
			wantLen:     4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, tooLong := padString(tt.s, tt.width)
			if tooLong != tt.wantTooLong {
				t.Errorf("padString(%q, %d) tooLong = %v, want %v", tt.s, tt.width, tooLong, tt.wantTooLong)
			}
			if len(got) != tt.wantLen {
				t.Errorf("padString(%q, %d) len = %d, want %d (got %q)", tt.s, tt.width, len(got), tt.wantLen, got)
			}
		})
	}
}

func TestPadStringWithANSI(t *testing.T) {
	// A string with ANSI color codes: visible text is "OK" (2 chars), but byte length is longer.
	colored := colorGreen + "OK" + colorReset // "\033[32mOK\033[0m"
	width := 10

	got, tooLong := padString(colored, width)

	if tooLong {
		t.Errorf("padString with ANSI codes reported tooLong=true, want false")
	}

	// Visible length should be 2; total padding added = 8 spaces.
	if !strings.HasSuffix(got, strings.Repeat(" ", 8)) {
		t.Errorf("padString with ANSI codes should end with 8 spaces, got %q", got)
	}

	// Must still start with the original colored string.
	if !strings.HasPrefix(got, colored) {
		t.Errorf("padString with ANSI codes should preserve original prefix, got %q", got)
	}
}
