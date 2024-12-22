package main

import (
	"context"
	"flag"
	"fmt"
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

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`) // Regex to match ANSI escape codes

// Strip ANSI escape codes and return the visible width of the string
func visibleWidth(s string) int {
	return len(ansiEscape.ReplaceAllString(s, ""))
}

// Pad a string to a specific width, accounting for visible length
func padString(s string, width int) string {
	visibleLen := visibleWidth(s)
	if visibleLen < width {
		return s + strings.Repeat(" ", width-visibleLen)
	}
	return s
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

	refreshDisplay := func() {
		progressLock.Lock()
		defer progressLock.Unlock()

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
			titleCol := padString(status.Title, 30)
			rippingCol := padString(colorize(status.Ripping, rippingColor), 20)
			encodingCol := padString(colorize(status.Encoding, encodingColor), 20)

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
			// Update progress for ripping
			progressLock.Lock()
			progress[i].Ripping = "In Progress"
			progressLock.Unlock()

			ripErr := mkv.RipTitle(ctx, &title, config.MKVOutputDirectory)

			if ripErr != nil {
				fmt.Printf("An error occurred ripping title: %s\nError: %v\n", title.FileName, ripErr)
				cancelProcessing()
				return
			}

			// Update progress for ripping completion
			progressLock.Lock()
			progress[i].Ripping = "Complete"
			progressLock.Unlock()

			encChannel <- hb.EncodingParams{
				TitleIndex:        title.Index,
				InputFilePath:     fmt.Sprintf("%s/%s", config.MKVOutputDirectory, title.FileName),
				OutputFilePath:    fmt.Sprintf("%s/%s", config.HBOutputDirectory, title.FileName),
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
				progressLock.Lock()
				progress[titleIndex].Encoding = "In Progress"
				progressLock.Unlock()

				encErr := hb.Encode(ctx, &params)

				if encErr != nil {
					fmt.Printf("An error occurred while encoding %s\nError: %v\n", params.InputFilePath, encErr)
					cancelProcessing()
					return
				}

				// Update progress for encoding completion
				progressLock.Lock()
				progress[titleIndex].Encoding = "Complete"
				progressLock.Unlock()

			case <-ctx.Done():
				return
			}
		}
	}()

	// Periodically refresh the display to ensure updates
	go func() {
		for ctx.Err() == nil {
			time.Sleep(500 * time.Millisecond)
			refreshDisplay()
		}
	}()

	wg.Wait()

	refreshDisplay()

	fmt.Printf("\nOperation Complete.\n")

	// if err != nil {
	// 	fmt.Printf("An error occurred while attempting to read from the config file.\nError: %v\n", err)
	// }

	// fmt.Println("\nHandy Task Parameter Values")

	// // If not provided arg values to config values
	// if quality == 0 {
	// 	quality = config.Quality
	// }

	// fmt.Printf("Quality: %d\n", quality)

	// if encoder == "" {
	// 	encoder = config.Encoder
	// }

	// fmt.Printf("Encoder: %s\n", encoder)
	// fmt.Println()

	// inputDirInfo, err := os.Stat(fmt.Sprintf("./%s", INPUT_DIR_NAME))

	// if err != nil {
	// 	if errors.Is(err, os.ErrNotExist) {
	// 		fmt.Printf("Input directory does not exist. Run handy -s and load the resulting '%s' directory with .mkv files to be encoded.\n", INPUT_DIR_NAME)
	// 	} else {
	// 		fmt.Printf("An error occurred while scanning for a preexisting %s directory.\nError: %v\n", INPUT_DIR_NAME, err)
	// 	}

	// 	return
	// }

	// // Check to make sure that the input directory contains .mkv files
	// inputDirEntries, err := os.ReadDir(inputDirInfo.Name())

	// if err != nil {
	// 	fmt.Printf("An error occurred while reading files from the input directory.\nError: %v\n", err)
	// 	return
	// }

	// var mkvDirEntries []fs.DirEntry = make([]fs.DirEntry, 0)

	// for _, entry := range inputDirEntries {
	// 	if strings.HasSuffix(entry.Name(), ".mkv") && !entry.IsDir() {
	// 		mkvDirEntries = append(mkvDirEntries, entry)
	// 	}
	// }

	// if len(mkvDirEntries) == 0 {
	// 	fmt.Printf("No .mkv files found in input directory. Exiting.\n")
	// 	return
	// }

	// outputDirInfo, err := os.Stat(fmt.Sprintf("./%s", OUTPUT_DIR_NAME))

	// if err != nil {
	// 	if errors.Is(err, os.ErrNotExist) {
	// 		fmt.Printf("Output directory does not exist in the current working directory. Creating the output directory...\n")

	// 		err := createDirectories(OUTPUT_DIR_NAME)

	// 		if err != nil {
	// 			fmt.Printf("An error occurred while creating output directory.\nError: %v\n", err)
	// 			return
	// 		}
	// 	} else {
	// 		fmt.Printf("An error occurred while scanning for a preexisting %s directory: %v\n", INPUT_DIR_NAME, err)
	// 		return
	// 	}
	// }

	// app := tview.NewApplication()
	// nav := ui.NewNavigator(context.Background())

	// homePage := ui.NewHomePage()
	// homePage.Setup(app, nav)

	// // Start the application.
	// err := app.SetRoot(nav.Pages, true).Run()

	// if err != nil {
	// 	panic(err)
	// }

	// totalBytesSaved := int64(0)
	// overallStartTime := time.Now()

	// for _, mkvFile := range mkvDirEntries {

	// 	inputFileInfo, err := mkvFile.Info()

	// 	if err != nil {
	// 		fmt.Printf("An error occurred when reading input file info from disk. Aborting remaining encoding tasks.\nError:  %v\n", err)
	// 	}

	// 	outputFileRelPath := fmt.Sprintf("./%s/%s", outputDirInfo.Name(), inputFileInfo.Name())

	// 	fmt.Printf("Encoding: %s\n", inputFileInfo.Name())

	// 	// HandBrakeCLI --input 50_First_Dates.mkv --output output.mkv --encoder nvenc_h264 --quality 18 --subtitle-lang-list English --all-subtitles --audio-lang-list English --all-audio
	// 	handbrakeCommand := exec.Command("HandBrakeCLI",
	// 		"--input", fmt.Sprintf("./%s/%s", inputDirInfo.Name(), inputFileInfo.Name()),
	// 		"--output", outputFileRelPath,
	// 		"--encoder", encoder,
	// 		"--quality", fmt.Sprintf("%d", quality),
	// 		"--subtitle-lang-list", "eng,jpn",
	// 		"--all-subtitles",
	// 		"--audio-lang-list", "eng",
	// 		"--all-audio",
	// 	)

	// 	// Get stdout pipe
	// 	stdoutPipe, err := handbrakeCommand.StdoutPipe()

	// 	if err != nil {
	// 		fmt.Printf("An error occurred when creating stdout pipe from HandBrakeCLI. Aborting remaining encoding tasks.\nError:  %v\n", err)
	// 		return
	// 	}

	// 	encodeStartTime := time.Now()

	// 	// Start the command
	// 	if err := handbrakeCommand.Start(); err != nil {
	// 		fmt.Printf("An error when spawning HandBrakeCLI process. Aborting remaining encoding tasks.\nError: %v\n", err)
	// 		return
	// 	}

	// 	// Read the output from the pipe
	// 	buf := make([]byte, 1024)

	// 	for {
	// 		n, err := stdoutPipe.Read(buf)
	// 		if n > 0 {
	// 			fmt.Print(string(buf[:n]))
	// 		}
	// 		if err != nil {
	// 			break
	// 		}
	// 	}

	// 	// Wait for the command to finish
	// 	if err := handbrakeCommand.Wait(); err != nil {
	// 		fmt.Printf("An error occurred during sqawned HandBrakeCLI execution.\nError: %v\n", err)
	// 		return
	// 	}

	// 	encodeDuration := time.Since(encodeStartTime).Round(time.Second)

	// 	outputFileAbsPath, err := filepath.Abs(outputFileRelPath)

	// 	if err != nil {
	// 		fmt.Printf("Error getting full path of output file '%s'. Aborting remaining encoding tasks.\nError: %v\n", outputDirInfo.Name(), err)
	// 		return
	// 	}

	// 	outputSize, err := getFileSize(outputFileAbsPath)

	// 	if err != nil {
	// 		fmt.Printf("An error occurred when reading output file info from disk. Aborting remaining encoding tasks.\nError:  %v\n", err)
	// 	}

	// 	bytesSaved := inputFileInfo.Size() - outputSize
	// 	totalBytesSaved += bytesSaved

	// 	fmt.Printf("Encode Complete: Output file - %s\n", outputFileAbsPath)
	// 	fmt.Printf("Time Elapsed: %s\n", formatTimeElapsedString(encodeDuration))
	// 	fmt.Printf("Saved Space: %s\n", formatSavedSpace(bytesSaved))
	// }

	// totalDuration := time.Since(overallStartTime).Round(time.Second)
	// fmt.Printf("\nEncoding Tasks Complete: See results in the '%s' directory.\n", outputDirInfo.Name())
	// fmt.Printf("Total Time Elapsed: %s\n", formatTimeElapsedString(totalDuration))
	// fmt.Printf("Total Saved Space: %s\n", formatSavedSpace(totalBytesSaved))
}
