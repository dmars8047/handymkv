package handy

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
func Exec(discId int) error {
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

	fmt.Printf("Reading titles from disc...\n\n")

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
		statuses: make([]titleStatus, len(titles)),
	}

	for i, title := range titles {
		tracker.statuses[i] = titleStatus{
			TitleIndex: title.Index,
			Title:      title.FileName,
			Ripping:    Pending,
			Encoding:   Pending,
		}
	}

	// Create output directory dirSlug with timestamp
	dirSlug := fmt.Sprintf("handy_%s", time.Now().Format("2006-01-02_15-04-05"))

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
	var encChannel = make(chan EncodingParams)
	var wg sync.WaitGroup

	processStartTime := time.Now()

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

			ripErr := ripTitle(ctx, &title, config.MKVOutputDirectory)

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

			if config.EncodeConfig.OutputFileFormat != "" && config.EncodeConfig.OutputFileFormat != "mkv" {
				encodingOutputFileName = fmt.Sprintf("%s.%s", strings.TrimSuffix(encodingOutputFileName, ".mkv"), config.EncodeConfig.OutputFileFormat)
			}

			encChannel <- EncodingParams{
				TitleIndex:          title.Index,
				MKVOutputPath:       filepath.Join(config.MKVOutputDirectory, title.FileName),
				HandBrakeOutputPath: filepath.Join(config.HBOutputDirectory, encodingOutputFileName),
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

				encErr := encode(ctx, &params)

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

	totalSizeRaw, totalSizeEncoded, err := calculateTotalFileSizes(titles, config)

	if err != nil {
		fmt.Printf("An error occurred while calculating total sizes: %v\n", err)
	}

	fmt.Printf("\nTotal size of raw unencoded files: %s\n", formatSavedSpace(totalSizeRaw))
	fmt.Printf("Total size of encoded files: %s\n", formatSavedSpace(totalSizeEncoded))
	fmt.Printf("Total disk space saved via encoding: %s\n", formatSavedSpace(totalSizeRaw-totalSizeEncoded))

	if config.DeleteRawMKVFiles {
		deleteRawFiles(config)
	}

	// Tell the user where the encoded files are located
	fmt.Printf("\nEncoded files are located in: %s\n\n", config.HBOutputDirectory)

	return nil
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

	var config *handyConfig

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
