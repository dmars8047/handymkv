package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dmars8047/handy/internal/fs"
	"github.com/dmars8047/handy/internal/hb"
	"github.com/dmars8047/handy/internal/mkv"
)

const (
	INPUT_DIR_NAME  = "Input"
	OUTPUT_DIR_NAME = "Output"
)

type TitleStatus struct {
	TitleIndex int
	Title      string
	Ripping    string
	Encoding   string
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[36m"
)

// colorize wraps a string in the specified color
func colorize(text, color string) string {
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

func main() {
	var quality int
	var encoder string

	setupFlag := flag.Bool("s", false, fmt.Sprintf(`Setup. 
	Creates the %s and %s directory as well as a %s file with sensible defaults in the current working directory. 
	Will not overwrite if they exist.`, INPUT_DIR_NAME, OUTPUT_DIR_NAME, fs.CONFIG_FILE_NAME))

	listFlag := flag.Bool("l", false, `List Encoders. 
	Displays a list of all valid encoder options.`)

	flag.IntVar(&quality, "q", 0, fmt.Sprintf(`Quality. 
	Sets the quality value to be used for each encoding task. If not provided then the value will be read from the '%s' file. 
	If the config file cannot be found then then a default value of '%d' will be used.`, fs.CONFIG_FILE_NAME, hb.DEFAULT_QUALITY))

	flag.StringVar(&encoder, "e", "", fmt.Sprintf(`Encoder. 
	If not provided then the value will be read from the '%s' file. 
	If the config file cannot be found then then a default value of '%s' will be used.`, fs.CONFIG_FILE_NAME, hb.DEFAULT_ENCODER))

	flag.Parse()

	setup := *setupFlag

	if setup {
		fmt.Println("Creating setup directories...")
		setupErr := fs.CreateDirectories(INPUT_DIR_NAME, OUTPUT_DIR_NAME)

		if setupErr != nil {
			fmt.Printf("An error occurred while setting up directories.\nError: %v\n", setupErr)
			return
		}

		fmt.Println("Creating config file...")

		setupErr = fs.CreateConfigFile()

		if setupErr != nil {
			fmt.Printf("An error occurred while setting up the config file.\nError: %v\n", setupErr)
			return
		}

		fmt.Printf("Directory setup complete.\n")
		return
	}

	list := *listFlag

	if list {
		fmt.Println("Valid encoders:")
		for k := range hb.PossibleEncoderValues {
			fmt.Printf("%s\n", k)
		}

		fmt.Printf("\nNote. This is a list of all possible values. Your installation may or may not be configured with all of these.\n")
		return
	}

	if encoder != "" {
		_, encoderAllowed := hb.PossibleEncoderValues[encoder]

		if !encoderAllowed {
			fmt.Printf("The provided encoder '%s' does not match any known encoder. Use handy -l to get a list of all valid encoder options.\n", encoder)
			return
		}
	}

	if quality < 0 {
		fmt.Printf("Invalid quality value detected. Setting to default value - %d\n", hb.DEFAULT_QUALITY)
		quality = hb.DEFAULT_QUALITY
	}

	_, err := exec.LookPath("makemkvcon")

	if err != nil {
		fmt.Println("makemkvcon not found in $PATH. Please install the makemkvcon. Exiting.")
		return
	}

	_, err = exec.LookPath("HandBrakeCLI")

	if err != nil {
		fmt.Println("HandBrakeCLI not found in $PATH. Please install the HandBrakeCLI. Exiting.")
		return
	}

	config, err := fs.ReadConfig()

	if err != nil {
		fmt.Printf("An error occurred reading config.\nError:%v\n", err)
	}

	fmt.Println()
	logoText := "██╗  ██╗ █████╗ ███╗   ██╗██████╗ ██╗   ██╗\n"
	logoText += "██║  ██║██╔══██╗████╗  ██║██╔══██╗╚██╗ ██╔╝\n"
	logoText += "███████║███████║██╔██╗ ██║██║  ██║ ╚████╔╝ \n"
	logoText += "██╔══██║██╔══██║██║╚██╗██║██║  ██║  ╚██╔╝  \n"
	logoText += "██║  ██║██║  ██║██║ ╚████║██████╔╝   ██║   \n"
	logoText += "╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═════╝    ╚═╝   \n"
	logoText += "A MakeMKV + HandBrake productivity tool by Herbzy\n"

	fmt.Printf("%s\n", logoText)

	titles, err := mkv.GetTitlesFromDisc()

	if err != nil {
		fmt.Printf("An error occurred while reading titles from disc\nError: %v\n", err)
	}

	if len(titles) < 1 {
		fmt.Printf("No titles could be read from disc. Exiting.\n")
		return
	}

	fmt.Printf("The following titles were read from the disc: %s\n\n", titles[0].DiscTitle)

	for _, title := range titles {
		fmt.Printf("ID: %d, Title Name: %s, Size: %s, Length: %s\n", title.Index, title.FileName, title.FileSize, title.Length)
	}

	var titleSelections string

	// Prompt the user for input
	fmt.Print("\nEnter the IDs of the titles to process (0,1,2...) or enter 'all' to process all titles: \n\n")
	fmt.Scanln(&titleSelections)

	if titleSelections != "all" {
		rawIds := strings.Split(titleSelections, ",")
		selectedIds := make([]int, 0)

		for _, rawIds := range rawIds {
			id, err := strconv.Atoi(rawIds)

			if err != nil {
				fmt.Printf("Invalid title selection input detected. Exiting.\n")
				return
			}

			selectedIds = append(selectedIds, id)
		}

		if len(selectedIds) < 1 {
			fmt.Printf("No selected titles detected. Exiting.\n")
			return
		}

		titles = slices.DeleteFunc(titles, func(x mkv.TitleInfo) bool {
			for _, sd := range selectedIds {
				if sd == x.Index {
					return false
				}
			}

			return true
		})
	}

	if len(titles) < 1 {
		fmt.Printf("No titles to process. Exiting.\n")
		return
	}

	// Titles progress tracking
	var progressLock sync.Mutex

	progress := make([]*TitleStatus, len(titles))
	for i, title := range titles {
		progress[i] = &TitleStatus{
			TitleIndex: title.Index,
			Title:      title.FileName,
			Ripping:    "Pending",
			Encoding:   "Pending",
		}
	}

	processStartTime := time.Now()

	refreshDisplay := func(changeFunc func()) {
		progressLock.Lock()
		defer progressLock.Unlock()

		changeFunc()

		fmt.Print("\033[H\033[2J") // Clear the terminal
		fmt.Printf("%s\n", logoText)
		fmt.Printf("%-30s%-20s%-20s\n", "Title", "Ripping", "Encoding")
		fmt.Println(strings.Repeat("-", 70))

		for _, status := range progress {
			rippingColor := colorReset
			encodingColor := colorReset

			// Set color based on status
			switch status.Ripping {
			case "Pending":
				rippingColor = colorYellow
			case "In Progress":
				rippingColor = colorBlue
			case "Complete":
				rippingColor = colorGreen
			}

			switch status.Encoding {
			case "Pending":
				encodingColor = colorYellow
			case "In Progress":
				encodingColor = colorBlue
			case "Complete":
				encodingColor = colorGreen
			}

			// Format and pad each column
			displayTitle := strings.TrimSuffix(status.Title, ".mkv")
			titleCol, titleTooLong := padString(displayTitle, 30)
			rippingCol, _ := padString(colorize(status.Ripping, rippingColor), 20)
			encodingCol, _ := padString(colorize(status.Encoding, encodingColor), 20)

			if titleTooLong {
				titleSpillOver := titleCol[29:]
				titleCol := titleCol[0:28]
				fmt.Printf("%s%s%s\n", titleCol, rippingCol, encodingCol)
				fmt.Printf("%s\n", titleSpillOver)
				return
			}

			// Print the row
			fmt.Printf("%s%s%s\n", titleCol, rippingCol, encodingCol)
		}
	}

	fmt.Println()

	ctx, cancelProcessing := context.WithCancel(context.Background())
	var encChannel = make(chan hb.EncodingParams, 1)
	var wg sync.WaitGroup

	wg.Add(1)

	// MKV
	go func() {
		defer wg.Done()
		defer close(encChannel)

		for i, title := range titles {
			applyInProgressChangeFunc := func() {
				// Update progress for ripping
				progress[i].Ripping = "In Progress"
			}

			refreshDisplay(applyInProgressChangeFunc)

			ripErr := mkv.RipTitle(ctx, &title, config.MKVOutputDirectory)

			if ripErr != nil {
				fmt.Printf("An error occurred ripping title: %s\nError: %v\n", title.FileName, ripErr)
				cancelProcessing()
				return
			}

			// Update progress for ripping completion
			applyCompletedChangeFunc := func() {
				progress[i].Ripping = "Complete"
			}

			refreshDisplay(applyCompletedChangeFunc)

			// Replace spaces with underscores for encoding run.
			encodingOutputFileName := strings.ReplaceAll(title.FileName, " ", "_")

			encChannel <- hb.EncodingParams{
				TitleIndex:        title.Index,
				InputFilePath:     fmt.Sprintf("%s/%s", config.MKVOutputDirectory, title.FileName),
				OutputFilePath:    fmt.Sprintf("%s/%s", config.HBOutputDirectory, encodingOutputFileName),
				Quality:           config.EncodeConfig.Quality,
				Encoder:           config.EncodeConfig.Encoder,
				SubtitleLanguages: config.EncodeConfig.SubtitleLanguages,
				AudioLanguages:    config.EncodeConfig.AudioLanguages,
			}
		}
	}()

	// HB
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			select {
			case params, ok := <-encChannel:
				if !ok {
					return
				}

				// Find the index of the title being encoded
				var titleIndex int
				for i, status := range progress {
					if status.TitleIndex == params.TitleIndex {
						titleIndex = i
						break
					}
				}

				// Update progress for encoding
				applyInProgressChangeFunc := func() {
					progress[titleIndex].Encoding = "In Progress"
				}

				refreshDisplay(applyInProgressChangeFunc)

				encErr := hb.Encode(ctx, &params)

				if encErr != nil {
					fmt.Printf("An error occurred while encoding %s\nError: %v\n", params.InputFilePath, encErr)
					cancelProcessing()
					return
				}

				// Update progress for encoding completion
				applyCompletedChangeFunc := func() {
					progress[titleIndex].Encoding = "Complete"
				}

				refreshDisplay(applyCompletedChangeFunc)

			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Wait()

	processDuration := time.Since(processStartTime).Round(time.Second)

	fmt.Printf("\nOperation Complete. Time Elapsed: %s\n", formatTimeElapsedString(processDuration))

	var confirmDeleteString string

	totalSizeRaw := int64(0)
	totalSizeEncoded := int64(0)

	for _, title := range titles {
		rawFilePath := fmt.Sprintf("%s/%s", config.MKVOutputDirectory, title.FileName)

		rawFileSize, err := fs.GetFileSize(rawFilePath)

		if err != nil {
			fmt.Printf("An error occurred reading the size of the raw MKV file - %s\nError: %v\n", rawFilePath, err)
			return
		}

		totalSizeRaw += rawFileSize

		encodedFilePath := fmt.Sprintf("%s/%s", config.HBOutputDirectory, strings.ReplaceAll(title.FileName, " ", "_"))

		encodedFileSize, err := fs.GetFileSize(encodedFilePath)

		if err != nil {
			fmt.Printf("An error occurred reading the size of the encoded MKV file - %s\nError: %v\n", encodedFilePath, err)
			return
		}

		totalSizeEncoded += encodedFileSize
	}

	fmt.Printf("\nTotal size of raw unencoded files: %s\n", formatSavedSpace(totalSizeRaw))
	fmt.Printf("Total size of encoded files: %s\n", formatSavedSpace(totalSizeEncoded))
	fmt.Printf("Total disk space saved via encoding: %s\n", formatSavedSpace(totalSizeRaw-totalSizeEncoded))

	fmt.Printf("\nDelete raw unencoded files? (y/N)\n\n")

	fmt.Scanln(&confirmDeleteString)

	fmt.Println()

	if strings.ToLower(confirmDeleteString) == "y" {
		for _, title := range titles {
			filePath := fmt.Sprintf("%s/%s", config.MKVOutputDirectory, title.FileName)

			fmt.Printf("Deleting %s\n", filePath)

			err := fs.DeleteFile(filePath)

			if err != nil {
				fmt.Printf("An error occured while deleting %s - Error: %v\n", filePath, err)
				return
			}
		}
	}
}
