package handy

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[36m"
)

// Prints the application logo
func PrintLogo() {
	var sb strings.Builder

	sb.WriteString("██╗  ██╗ █████╗ ███╗   ██╗██████╗ ██╗   ██╗\n")
	sb.WriteString("██║  ██║██╔══██╗████╗  ██║██╔══██╗╚██╗ ██╔╝\n")
	sb.WriteString("███████║███████║██╔██╗ ██║██║  ██║ ╚████╔╝ \n")
	sb.WriteString("██╔══██║██╔══██║██║╚██╗██║██║  ██║  ╚██╔╝  \n")
	sb.WriteString("██║  ██║██║  ██║██║ ╚████║██████╔╝   ██║   \n")
	sb.WriteString("╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═════╝    ╚═╝   \n")
	sb.WriteString("A MakeMKV + HandBrake productivity tool by Herbzy\n")

	fmt.Printf("\n%s\n", sb.String())
}

// clear clears the terminal
func clear() {
	fmt.Print("\033[H\033[2J") // Clear the terminal
}

// colorize wraps a string in the specified color
func colorize(text statusValue, color string) string {
	return fmt.Sprintf("%s%s%s", color, text.String(), colorReset)
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

// Prompts the user for a string value. If the user provides an empty string, the default value is returned.
func promptForString(prompt, explain, defaultValue string, validValues map[string]struct{}) string {
	var input string
	fmt.Printf("%s\n\n", prompt)

	if explain != "" {
		fmt.Printf("%s\n", explain)
	}

	if len(validValues) > 0 {
		fmt.Printf("Valid values:\n\n")
		for k := range validValues {
			fmt.Printf("%s\n", k)
		}
	}

	if defaultValue != "" {
		fmt.Printf("\nDefault: %s\n\n", defaultValue)
	}

	fmt.Scanln(&input)
	fmt.Println()

	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue
	}

	if validValues != nil {
		if _, valid := validValues[input]; !valid {
			fmt.Printf("Invalid value. Using default value: %s\n", defaultValue)
			return defaultValue
		}
	}
	return input
}

// Prompts the user for an integer value. If the user provides an empty string, the default value is returned.
func promptForInt(prompt, explain string, defaultValue int) int {
	var input string
	fmt.Printf("%s\n\n", prompt)

	if explain != "" {
		fmt.Printf("%s\n", explain)
	}

	fmt.Printf("Default: %d\n\n", defaultValue)

	fmt.Scanln(&input)
	fmt.Println()

	if input == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(input)

	if err != nil || value < 0 {
		fmt.Printf("Invalid value. Using default value: %d\n", defaultValue)
		return defaultValue
	}

	return value
}

// Prompts the user for a string slice value. If the user provides an empty string, the default value is returned.
func promptForStringSlice(prompt, explain, defaultValue string) []string {
	var input string
	fmt.Printf("%s\n\n", prompt)

	if explain != "" {
		fmt.Printf("%s\n\n", explain)
	}

	if defaultValue != "" {
		fmt.Printf("Default: %s\n\n", defaultValue)
	}

	fmt.Scanln(&input)
	fmt.Println()

	if input == "" {
		return []string{defaultValue}
	}

	return strings.Split(input, ",")
}

// Prompts the user for a boolean value. If the user provides an empty string, the default value is returned.
func promptForBool(prompt, explain string, defaultValue bool) bool {
	var input string

	defaultStr := "N"

	if defaultValue {
		defaultStr = "Y"
	}

	fmt.Printf("%s\n\n", prompt)

	if explain != "" {
		fmt.Printf("%s\n", explain)
	}

	fmt.Printf("\nDefault: %s\n\n", defaultStr)

	fmt.Scanln(&input)
	fmt.Println()

	input = strings.ToLower(strings.TrimSpace(input))

	if input == "y" {
		return true
	}

	return defaultValue
}