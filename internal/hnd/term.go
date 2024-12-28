package hnd

import (
	"fmt"
	"regexp"
	"strings"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[36m"
)

func PrintLogo() {
	logoText := "██╗  ██╗ █████╗ ███╗   ██╗██████╗ ██╗   ██╗\n"
	logoText += "██║  ██║██╔══██╗████╗  ██║██╔══██╗╚██╗ ██╔╝\n"
	logoText += "███████║███████║██╔██╗ ██║██║  ██║ ╚████╔╝ \n"
	logoText += "██╔══██║██╔══██║██║╚██╗██║██║  ██║  ╚██╔╝  \n"
	logoText += "██║  ██║██║  ██║██║ ╚████║██████╔╝   ██║   \n"
	logoText += "╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═════╝    ╚═╝   \n"
	logoText += "A MakeMKV + HandBrake productivity tool by Herbzy\n"

	fmt.Printf("\n%s\n", logoText)
}

// colorize wraps a string in the specified color
func colorize(text statusValue, color string) string {
	return fmt.Sprintf("%s%s%s", color, text, colorReset)
}

// Pad a string to a specific width, accounting for visible length.
// Returns true if the given visible length of the string is longer than the width.
func padString(s string, width int) (string, bool) {
	var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`) // Regex to match ANSI escape codes

	// Strip ANSI escape codes and return the visible width of the string
	visibleLen := len(ansiEscape.ReplaceAllString(s, ""))

	if visibleLen < width {
		return s + strings.Repeat(" ", width-visibleLen), false
	}

	return s, visibleLen > width
}
