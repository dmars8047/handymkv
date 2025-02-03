package hmkv

import (
	"context"
	"fmt"
	"os"
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
func Exec(discIds []int) error {
	config, err := ReadConfig()

	if err != nil {
		if err == ErrConfigNotFound {
			return err
		}

		return fmt.Errorf("an unexpected error occurred while reading the configuration file: %w", err)
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

	processTitles := make([]TitleInfo, 0)

	for i, discId := range discIds {

		fmt.Printf("Reading titles from disc %d...\n\n", discId)

		titles, err := getTitles(discId)

		if err != nil {
			return err
		}

		fmt.Printf("The following titles were read from the disc - %s\n\n", titles[0].DiscTitle)

		for _, title := range titles {
			fmt.Printf("ID: %d, Title Name: %s, Size: %s, Length: %s\n", title.Index, title.FileName, title.FileSize, title.Length)
		}

		var titleSelections string

		// Prompt the user for input
		fmt.Print("\nEnter the IDs of the titles to process (0,1,2...) or enter 'all' to process all titles: \n\n")
		fmt.Scanln(&titleSelections)

		// Remove invalid characters
		titleSelections = strings.ReplaceAll(titleSelections, " ", "")
		titleSelections = strings.Trim(titleSelections, ",")
		titleSelections = strings.ReplaceAll(titleSelections, "(", "")
		titleSelections = strings.ReplaceAll(titleSelections, ")", "")

		if titleSelections == "" {
			fmt.Printf("No title selections detected. Exiting.\n\n")
			return nil
		}

		// If the user entered 'all', don't filter the titles
		if titleSelections != "all" {
			rawIds := strings.Split(titleSelections, ",")
			selectedIds := make([]int, 0)

			for _, rawIds := range rawIds {
				id, err := strconv.Atoi(rawIds)

				if err != nil {
					fmt.Printf("\nInvalid title selection input detected.\n\n")
					return nil
				}

				selectedIds = append(selectedIds, id)
			}

			if len(selectedIds) < 1 {
				fmt.Printf("\nNo selected titles detected.\n\n")
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

		processTitles = append(processTitles, titles...)

		if i < len(discIds)-1 {
			fmt.Println()
		}
	}

	if len(processTitles) < 1 {
		fmt.Printf("\nNo titles to process. Exiting.\n\n")
		return nil
	}

	// If there any discs with identical titles, set prependDiscToSub to true for those titles
	var titleNames = make(map[string]int)

	for _, title := range processTitles {
		titleNames[strings.ToLower(title.FileName)]++
	}

	for _, title := range processTitles {
		if titleNames[strings.ToLower(title.FileName)] > 1 {
			title.SetPrependDiscToSubdirectory(true)
		}
	}

	// Titles progress tracking
	tracker := progressTracker{
		statuses: make([]titleStatus, len(processTitles)),
	}

	for i, title := range processTitles {
		tracker.statuses[i] = titleStatus{
			TitleIndex: title.Index,
			Title:      title.FileName,
			DiscId:     title.DiscId,
			Ripping:    Pending,
			Encoding:   Pending,
		}
	}

	// Create output directory dirSlug with timestamp
	dirSlug := fmt.Sprintf("handymkv_%s", time.Now().Format("2006-01-02_15-04-05"))

	config.MKVOutputDirectory = filepath.Join(config.MKVOutputDirectory, dirSlug)

	err = os.MkdirAll(config.MKVOutputDirectory, 0740)

	if err != nil {
		return fmt.Errorf("an error occurred while creating the mkv output directory: %w", err)
	}

	config.HBOutputDirectory = filepath.Join(config.HBOutputDirectory, dirSlug)

	err = os.MkdirAll(config.HBOutputDirectory, 0740)

	if err != nil {
		return fmt.Errorf("an error occurred while creating the handbrake output directory: %w", err)
	}

	fmt.Println()

	ctx, cancelProcessing := context.WithCancel(context.Background())
	var encChannel = make(chan EncodingParams, len(processTitles))
	var processWaitGroup sync.WaitGroup

	processStartTime := time.Now()

	// MKV
	processWaitGroup.Add(1)

	// For each disc rip the titles
	go func() {
		defer close(encChannel)
		defer processWaitGroup.Done()

		var rippingWaitGroup sync.WaitGroup

		for _, discId := range discIds {
			var discTitles []TitleInfo

			for _, title := range processTitles {
				if title.DiscId == discId {
					discTitles = append(discTitles, title)
				}
			}

			if len(discTitles) < 1 {
				continue
			}

			// Make sure the subdirectories exists
			os.MkdirAll(filepath.Join(config.MKVOutputDirectory, discTitles[0].Subdirectory()), 0740)
			os.MkdirAll(filepath.Join(config.HBOutputDirectory, discTitles[0].Subdirectory()), 0740)

			rippingWaitGroup.Add(1)

			go func() {
				defer rippingWaitGroup.Done()
				ripTitles(ctx, &tracker, discTitles, config, encChannel, cancelProcessing)
			}()
		}

		rippingWaitGroup.Wait()
	}()

	// HB
	processWaitGroup.Add(1)

	go func() {
		defer processWaitGroup.Done()
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

				// Make sure the input file exists
				if _, err := os.Stat(params.MKVOutputPath); os.IsNotExist(err) {
					tracker.setError(fmt.Errorf("encoding input file %s does not exist", params.MKVOutputPath))
					cancelProcessing()
					return
				}

				encErr := encode(ctx, &params)

				if encErr != nil {
					tracker.setError(encErr)
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

	processWaitGroup.Wait()

	if tracker.err != nil {
		return tracker.err
	}

	processDuration := time.Since(processStartTime).Round(time.Second)

	fmt.Printf("\nOperation Complete. Time Elapsed - %s\n", formatTimeElapsedString(processDuration))

	totalSizeRaw, totalSizeEncoded, err := calculateTotalFileSizes(processTitles, config)

	if err != nil {
		fmt.Printf("An error occurred while calculating total sizes - %v\n", err)
	}

	fmt.Printf("\nTotal size of raw unencoded files - %s\n", formatSavedSpace(totalSizeRaw))
	fmt.Printf("Total size of encoded files - %s\n", formatSavedSpace(totalSizeEncoded))

	savedSpace := totalSizeRaw - totalSizeEncoded
	if savedSpace > 0 {
		fmt.Printf("Total disk space saved via encoding - %s\n", formatSavedSpace(totalSizeRaw-totalSizeEncoded))
	}

	if config.DeleteRawMKVFiles {
		deleteRawFiles(config)
	}

	// Tell the user where the encoded files are located
	fmt.Printf("\nEncoded files are located in: %s\n\n", config.HBOutputDirectory)

	return nil
}

func ripTitles(
	ctx context.Context,
	tracker *progressTracker,
	processTitles []TitleInfo,
	config *handyMKVConfig,
	encChannel chan EncodingParams,
	cancelProcessing context.CancelFunc) {

	for _, title := range processTitles {
		applyInProgress := func(status *titleStatus) {
			status.Ripping = InProgress
		}

		tracker.applyChangeAndDisplay(title.Index, applyInProgress)

		var mkvOutputDirectory string = filepath.Join(config.MKVOutputDirectory, title.Subdirectory())

		ripErr := ripTitle(ctx, &title, mkvOutputDirectory)

		if ripErr != nil {
			tracker.setError(ripErr)
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

		if config.EncodeConfig.OutputFileFormat != "" && config.EncodeConfig.OutputFileFormat != "mkv" {
			encodingOutputFileName = fmt.Sprintf("%s.%s", strings.TrimSuffix(encodingOutputFileName, ".mkv"), config.EncodeConfig.OutputFileFormat)
		}

		var hbOutputDir string = filepath.Join(config.HBOutputDirectory, title.Subdirectory())

		encChannel <- EncodingParams{
			TitleIndex:          title.Index,
			MKVOutputPath:       filepath.Join(mkvOutputDirectory, title.FileName),
			HandBrakeOutputPath: filepath.Join(hbOutputDir, encodingOutputFileName),
			Quality:             config.EncodeConfig.Quality,
			Encoder:             config.EncodeConfig.Encoder,
			EncoderPreset:       config.EncodeConfig.EncoderPreset,
			OutputFileFormat:    config.EncodeConfig.OutputFileFormat,
			Preset:              config.EncodeConfig.Preset,
			PresetFile:          config.EncodeConfig.PresetFile,
			SubtitleLanguages:   config.EncodeConfig.SubtitleLanguages,
			AudioLanguages:      config.EncodeConfig.AudioLanguages,
		}
	}
}

// Prompts the user to create a configuration file.
func Setup() error {
	fmt.Printf("What level of configuration would you like to create?\n\n")
	fmt.Println("1 - User-wide configuration (recommended).")
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

	var config *handyMKVConfig

	for {
		config, err = promptForConfig(configLocationSelection)

		if err != nil {
			fmt.Printf("An error occurred while prompting for configuration values: %v\n", err)
			return err
		}

		fmt.Printf("\n%s\n", config.String())
		fmt.Printf("Accept these settings? [y/N]\n\n")

		var choice string
		fmt.Scanln(&choice)

		if strings.ToLower(choice) == "y" {
			break
		}
	}

	clear()
	fmt.Println("Creating config file...")

	err = createConfigFile(configFileLocation(configLocationSelection), config, false)

	if err != nil {
		return err
	}

	fmt.Printf("\nConfig file creation complete.\n\n")

	return nil
}
