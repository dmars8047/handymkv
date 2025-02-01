package hmkv

import (
	"fmt"
	"strings"
	"sync"
)

// Keeps track of the progress of the ripping and encoding processes.
// Outputs the progress to the terminal.
type progressTracker struct {
	statuses []titleStatus
	mutex    sync.Mutex
	err      error
}

type statusValue uint8

const (
	Pending statusValue = iota
	InProgress
	Complete
)

// String representation of the statusValue.
func (s statusValue) String() string {
	switch s {
	case Pending:
		return "Pending"
	case InProgress:
		return "In Progress"
	case Complete:
		return "Complete"
	default:
		return "Unknown"
	}
}

// Represents the status of a title.
type titleStatus struct {
	// The index of the title on the disc.
	TitleIndex int
	// The name of the title.
	Title string
	// The disc index.
	DiscId int
	// The status of the ripping process.
	Ripping statusValue
	// The status of the encoding process.
	Encoding statusValue
}

func (pt *progressTracker) applyChangeAndDisplay(titleIndex int, applyChangeFunc func(*titleStatus)) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	for i, status := range pt.statuses {
		if status.TitleIndex == titleIndex {
			applyChangeFunc(&pt.statuses[i])
			break
		}
	}

	pt.refreshDisplay()
}

func (pt *progressTracker) refreshDisplay() {
	clear()
	PrintLogo()
	fmt.Printf("%-30s%-10s%-20s%-20s\n", "Title", "Disc", "Ripping", "Encoding")
	fmt.Println(strings.Repeat("-", 80))

	for _, status := range pt.statuses {
		rippingColor := getColor(status.Ripping)
		encodingColor := getColor(status.Encoding)

		// Format and pad each column
		displayTitle := strings.TrimSuffix(status.Title, ".mkv")
		titleCol, titleTooLong := padString(displayTitle, 30)
		discIdCol, _ := padString(fmt.Sprintf("%d", status.DiscId), 10)
		rippingCol, _ := padString(colorize(status.Ripping, rippingColor), 20)
		encodingCol, _ := padString(colorize(status.Encoding, encodingColor), 20)

		if titleTooLong {
			titleSpillOver := titleCol[27:]
			titleCol = fmt.Sprintf("%s   ", titleCol[0:27])
			fmt.Printf("%s%s%s%s\n", titleCol, discIdCol, rippingCol, encodingCol)
			fmt.Printf("%s\n", titleSpillOver)
			continue
		}

		// Print the row
		fmt.Printf("%s%s%s%s\n", titleCol, discIdCol, rippingCol, encodingCol)
	}
}

func (pt *progressTracker) setError(err error) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	pt.err = err
}

func getColor(status statusValue) string {
	switch status {
	case Pending:
		return colorYellow
	case InProgress:
		return colorBlue
	case Complete:
		return colorGreen
	default:
		return colorReset
	}
}
