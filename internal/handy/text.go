package handy

import (
	"fmt"
	"math"
	"time"
)

func formatTimeElapsedString(m time.Duration) string {
	minutes := int(math.Floor(m.Minutes()))
	seconds := int(math.Floor(m.Seconds())) - (minutes * 60)
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}

// Formats the given size in bytes into a human-readable string (GB, MB, KB, or Bytes).
func formatSavedSpace(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d Bytes", bytes)
	}
}
