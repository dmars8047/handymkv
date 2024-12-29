package hnd

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Executes the main functionality of the program.
// Reads the configuration file, reads titles from the disc, prompts the user for which titles they want to rip,
// and processes the selected titles.
func Exec(discId int, quality int, encoder string) error {
	config, err := ReadConfig()

	if err != nil {
		if err == ErrConfigNotFound {
			return err
		}

		return fmt.Errorf("an unexpected error occurred while reading the configuration file: %w", err)
	}

	// The user provided a quality value via the command line
	if quality != -1 {
		config.EncodeConfig.Quality = quality
	}

	// The user provided an encoder value via the command line
	if encoder != "" {
		config.EncodeConfig.Encoder = encoder
	}

	// Make sure the output directories exist
	err = os.MkdirAll(config.MKVOutputDirectory, 0740)

	if err != nil {
		return fmt.Errorf("an error occurred while creating the mkv output directory: %w", err)
	}

	err = os.MkdirAll(config.HBOutputDirectory, 0740)

	if err != nil {
		return fmt.Errorf("an error occurred while creating the handbrake output directory: %w", err)
	}

	titles, err := getTitles(discId)

	if err != nil {
		return err
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
				fmt.Printf("Invalid title selection input detected.\n")
				return nil
			}

			selectedIds = append(selectedIds, id)
		}

		if len(selectedIds) < 1 {
			fmt.Printf("No selected titles detected.\n")
			return nil
		}

		titles = slices.DeleteFunc(titles, func(x TitleInfo) bool {
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
		return nil
	}

	// Titles progress tracking
	tracker := progressTracker{
		statuses: make(map[int]titleStatus, len(titles)),
	}

	for _, title := range titles {
		tracker.statuses[title.Index] = titleStatus{
			TitleIndex: title.Index,
			Title:      title.FileName,
			Ripping:    Pending,
			Encoding:   Pending,
		}
	}

	processStartTime := time.Now()

	fmt.Println()

	ctx, cancelProcessing := context.WithCancel(context.Background())
	var encChannel = make(chan EncodingParams, 1)
	var wg sync.WaitGroup

	wg.Add(1)

	// MKV
	go func() {
		defer wg.Done()
		defer close(encChannel)

		for _, title := range titles {
			applyInProgress := func(status *titleStatus) {
				status.Ripping = InProgress
			}

			tracker.applyChangeAndDisplay(title.Index, applyInProgress)

			ripErr := RipTitle(ctx, &title, config.MKVOutputDirectory)

			if ripErr != nil {
				tracker.setError(fmt.Errorf("an error occurred while ripping title: %w", ripErr))
				cancelProcessing()
				return
			}

			applyComplete := func(status *titleStatus) {
				status.Ripping = Complete
			}

			// Update progress for ripping completion
			tracker.applyChangeAndDisplay(title.Index, applyComplete)

			// Replace spaces with underscores for encoding run.
			encodingOutputFileName := strings.ReplaceAll(title.FileName, " ", "_")

			encChannel <- EncodingParams{
				TitleIndex:          title.Index,
				MKVOutputPath:       filepath.Join(config.MKVOutputDirectory, title.FileName),
				HandBrakeOutputPath: filepath.Join(config.HBOutputDirectory, encodingOutputFileName),
				Quality:             config.EncodeConfig.Quality,
				Encoder:             config.EncodeConfig.Encoder,
				SubtitleLanguages:   config.EncodeConfig.SubtitleLanguages,
				AudioLanguages:      config.EncodeConfig.AudioLanguages,
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

				applyInProgress := func(status *titleStatus) {
					status.Encoding = InProgress
				}

				tracker.applyChangeAndDisplay(params.TitleIndex, applyInProgress)

				encErr := Encode(ctx, &params)

				if encErr != nil {
					// fmt.Printf("An error occurred while encoding %s\nError: %v\n", params.InputFilePath, encErr)
					tracker.setError(fmt.Errorf("an error occurred while encoding %s: %w", params.MKVOutputPath, encErr))
					cancelProcessing()
					return
				}

				applyComplete := func(status *titleStatus) {
					status.Encoding = Complete
				}

				// Update progress for encoding completion
				tracker.applyChangeAndDisplay(params.TitleIndex, applyComplete)
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Wait()

	if tracker.err != nil {
		return tracker.err
	}

	processDuration := time.Since(processStartTime).Round(time.Second)

	fmt.Printf("\nOperation Complete. Time Elapsed: %s\n", formatTimeElapsedString(processDuration))

	totalSizeRaw, totalSizeEncoded, err := calculateTotalSizes(titles, config)

	if err != nil {
		fmt.Printf("An error occurred while calculating total sizes: %v\n", err)
	}

	fmt.Printf("\nTotal size of raw unencoded files: %s\n", formatSavedSpace(totalSizeRaw))
	fmt.Printf("Total size of encoded files: %s\n", formatSavedSpace(totalSizeEncoded))
	fmt.Printf("Total disk space saved via encoding: %s\n", formatSavedSpace(totalSizeRaw-totalSizeEncoded))

	if !promptUserForDeletion() {
		deleteRawFiles(titles, config)
	}

	return nil
}

// Keeps track of the progress of the ripping and encoding processes.
// Outputs the progress to the terminal.
type progressTracker struct {
	statuses map[int]titleStatus
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
	// The status of the ripping process.
	Ripping statusValue
	// The status of the encoding process.
	Encoding statusValue
}

func (pt *progressTracker) applyChangeAndDisplay(titleIndex int, applyChangeFunc func(*titleStatus)) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	var titleStatus titleStatus = pt.statuses[titleIndex]

	applyChangeFunc(&titleStatus)

	pt.refreshDisplay()
}

func (pt *progressTracker) refreshDisplay() {
	clear()
	PrintLogo()
	fmt.Printf("%-30s%-20s%-20s\n", "Title", "Ripping", "Encoding")
	fmt.Println(strings.Repeat("-", 70))

	for _, status := range pt.statuses {
		rippingColor := getColor(status.Ripping)
		encodingColor := getColor(status.Encoding)

		// Format and pad each column
		displayTitle := strings.TrimSuffix(status.Title, ".mkv")
		titleCol, titleTooLong := padString(displayTitle, 30)
		rippingCol, _ := padString(colorize(status.Ripping, rippingColor), 20)
		encodingCol, _ := padString(colorize(status.Encoding, encodingColor), 20)

		if titleTooLong {
			titleSpillOver := titleCol[29:]
			titleCol = titleCol[0:28]
			fmt.Printf("%s%s%s\n", titleCol, rippingCol, encodingCol)
			fmt.Printf("%s\n", titleSpillOver)
			return
		}

		// Print the row
		fmt.Printf("%s%s%s\n", titleCol, rippingCol, encodingCol)
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

func getTitles(discId int) ([]TitleInfo, error) {
	titles, err := getTitlesFromDisc(discId)

	if err != nil {
		return titles, fmt.Errorf("an error occurred while reading titles from disc: %w", err)
	}

	if len(titles) < 1 {
		return titles, fmt.Errorf("no titles could be read from disc")
	}

	return titles, nil
}

func calculateTotalSizes(titles []TitleInfo, config *HandyConfig) (int64, int64, error) {
	var totalSizeRaw, totalSizeEncoded int64

	for _, title := range titles {
		rawFilePath := filepath.Join(config.MKVOutputDirectory, title.FileName)
		rawFileSize, err := GetFileSize(rawFilePath)
		if err != nil {
			return 0, 0, err
		}
		totalSizeRaw += rawFileSize

		encodedFilePath := filepath.Join(config.HBOutputDirectory, strings.ReplaceAll(title.FileName, " ", "_"))
		encodedFileSize, err := GetFileSize(encodedFilePath)
		if err != nil {
			return 0, 0, err
		}
		totalSizeEncoded += encodedFileSize
	}

	return totalSizeRaw, totalSizeEncoded, nil
}

func promptUserForDeletion() bool {
	var confirmDeleteString string

	fmt.Printf("\nDelete raw unencoded files? (y/N)\n\n")

	fmt.Scanln(&confirmDeleteString)
	fmt.Println()

	return strings.ToLower(confirmDeleteString) == "y"
}

func deleteRawFiles(titles []TitleInfo, config *HandyConfig) {
	for _, title := range titles {
		filePath := filepath.Join(config.MKVOutputDirectory, title.FileName)

		fmt.Printf("Deleting %s\n", filePath)

		if err := os.Remove(filePath); err != nil {
			fmt.Printf("An error occurred while deleting %s - Error: %v\n", filePath, err)
		}
	}
}

// Prompts the user to create a configuration file.
func Setup() error {
	fmt.Printf("What level of configuration would you like to create?\n\n")
	fmt.Println("1 - User-wide configuration.")
	fmt.Println("2 - Current working directory.")
	fmt.Println()

	var configLocationSelectionString string

	fmt.Scanln(&configLocationSelectionString)

	fmt.Println()

	configLocationSelection, err := strconv.Atoi(configLocationSelectionString)

	if err != nil {
		fmt.Println("Configuration file location selection could not be parsed.")
		return err
	}

	if configLocationSelection < 1 || configLocationSelection > 2 {
		fmt.Println("Invalid configuration file location selection.")
		return nil
	}

	var config *HandyConfig

	for {
		config = promptForConfig(configLocationSelection)
		fmt.Printf("\n%s\n", config.String())
		fmt.Printf("Accept these settings? [Y/N]\n\n")

		var choice string
		fmt.Scanln(&choice)

		if strings.ToLower(choice) == "y" {
			break
		}
	}

	clear()
	fmt.Println("Creating config file...")

	err = CreateConfigFile(ConfigFileLocation(configLocationSelection), config, false)

	if err != nil {
		return err
	}

	fmt.Printf("\nConfig file creation complete.\n\n")

	return nil
}

// Prompts the user for configuration values and returns a new HandyConfig object.
func promptForConfig(locationSelection int) *HandyConfig {
	var config HandyConfig

	clear()
	fmt.Printf("You will now answer a series of questions to provide default values for your configuration. Press enter to select the indicated default. For more information, please consult the project documentation.\n\n(Press enter to continue)\n\n")

	// Press enter to continue
	fmt.Scanln()
	clear()

	config.EncodeConfig.Encoder = promptForString("What encoder should be used by default?", "", DEFAULT_ENCODER, PossibleEncoderValues)
	clear()
	config.EncodeConfig.Quality = promptForInt("What should the default quality be set to?", "", DEFAULT_QUALITY)
	clear()

	config.EncodeConfig.AudioLanguages = promptForStringSlice("What audio languages should be included in encoded output files?",
		"Provide a comma delimited list of ISO 639-2 strings. Example: eng,jpn",
		"any")

	clear()

	config.EncodeConfig.IncludeAllRelevantAudio = promptForBool("Include all relevant audio tracks in encoded output files?",
		"Some discs contain multiple audio tracks in the same language. If this option is enabled, all audio tracks in the same language will be included in the encoded output files. If this option is disabled, only the first audio track in the specified language will be included.",
		false)

	clear()

	config.EncodeConfig.SubtitleLanguages = promptForStringSlice("What subtitle languages should be included in encoded output files?",
		"Provide a comma delimited list of ISO 639-2 strings. Example: eng,jpn",
		"eng")
	clear()

	config.EncodeConfig.IncludeAllRelevantSubtitles = promptForBool("Include all relevant subtitle tracks in encoded output files?",
		"Some discs contain multiple subtitle tracks in the same language. If this option is enabled, all subtitle tracks in the same language will be included in the encoded output files.",
		false)
	clear()

	var handyDir string

	if locationSelection == 1 {
		// Get logged in user's home directory
		usr, err := user.Current()

		if err != nil {
			fmt.Printf("Error getting current user: %v\n", err)
		}

		handyDir = filepath.Join(usr.HomeDir, "handy")
	} else {
		handyDir = "."
	}

	defaultMKVOutputDirectory := filepath.Join(handyDir, "mkvoutput")

	config.MKVOutputDirectory = promptForString("Provide a path to a directory that raw unencoded MKV files can be staged.",
		fmt.Sprintf("Absolute path to a directory. Example: %s", defaultMKVOutputDirectory),
		defaultMKVOutputDirectory,
		nil)

	clear()

	defaultHBOutputDirectory := filepath.Join(handyDir, "hboutput")

	config.HBOutputDirectory = promptForString("Provide a path to a directory that HandBrake encoded output files can be placed. Using the same directory as the MKV output directory is not recommended.",
		fmt.Sprintf("Absolute path to a directory. Example: %s", defaultHBOutputDirectory),
		defaultHBOutputDirectory, nil)

	clear()

	return &config
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
