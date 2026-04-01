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
func Exec(mkv *MakeMKV, hb *HandBrakeCLI, discIds []int, appVersion string, automationNames []string) error {
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

		titles, err := mkv.getTitles(discId)

		if err != nil {
			return err
		}

		fmt.Printf("The following titles were read from the disc - %s\n\n", titles[0].DiscTitle)

		for _, title := range titles {
			fmt.Printf("ID: %d, Title Name: %s, Size: %s, Length: %s\n", title.Index, title.FileName, title.FileSizeDesc, title.Length)
		}

		var titleSelections string

		// Prompt the user for input
		fmt.Print("\nEnter the IDs of the titles to process (0,1,2...) or enter 'all' to process all titles: \n\n")
		titleSelections = readLine()

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
				return !slices.Contains(selectedIds, x.Index)
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

	// If there any titles that have an identical disc title to another disc, set prependDiscToSub to true for those titles
	var discNames = make(map[string]int)

	for _, title := range processTitles {
		discNames[strings.ToLower(title.DiscTitle)]++
	}

	for i := range processTitles {
		if discNames[strings.ToLower(processTitles[i].DiscTitle)] > 1 {
			processTitles[i].SetPrependDiscToSubdirectory(true)
		}
	}

	// Titles progress tracking
	tracker := progressTracker{
		statuses:        make([]titleStatus, len(processTitles)),
		refreshInterval: 200 * time.Millisecond,
	}

	for i, title := range processTitles {
		tracker.statuses[i] = titleStatus{
			TitleIndex:        title.Index,
			Title:             title.FileName,
			DiscId:            title.DiscId,
			Ripping:           Pending,
			Encoding:          Pending,
			ExpectedSizeBytes: int64(title.FileSizeBytes),
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

	// Automation selection and pre-run param collection
	var selectedAutomations []Automation
	var preRunParams map[string]string

	for _, name := range automationNames {
		a, err := LoadAutomation(name)
		if err != nil {
			fmt.Printf("Warning: could not load automation '%s': %v\n", name, err)
			continue
		}
		selectedAutomations = append(selectedAutomations, *a)
	}

	if len(selectedAutomations) > 0 {
		var paramErr error
		preRunParams, paramErr = resolvePreRunParams(selectedAutomations)
		if paramErr != nil {
			fmt.Printf("Warning: %v\nAutomations will be skipped.\n\n", paramErr)
			selectedAutomations = nil
		}
	}

	ctx, cancelProcessing := context.WithCancel(context.Background())
	var encChannel = make(chan EncodingParams, len(processTitles))
	var processWaitGroup sync.WaitGroup
	var manifestMu sync.Mutex
	var manifestEntries []EncodingParams

	processStartTime := time.Now()

	// Start central refresh ticker for display updates
	stopRefreshTicker := tracker.startRefreshTicker(ctx)

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
				ripTitles(mkv, ctx, &tracker, discTitles, config, encChannel, cancelProcessing)
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

				if err := encodeTitle(ctx, hb, &tracker, &params); err != nil {
					tracker.setError(err)
					cancelProcessing()
					return
				}

				manifestMu.Lock()
				manifestEntries = append(manifestEntries, params)
				manifestMu.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()

	processWaitGroup.Wait()

	// Stop ticker IMMEDIATELY before printing final messages
	stopRefreshTicker()

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

	// Run automations before raw file deletion so scripts can access raw MKV files
	var automationEntries []manifestAutomation
	if len(selectedAutomations) > 0 {
		outputData := buildRunOutputData(
			config.HBOutputDirectory,
			config.MKVOutputDirectory,
			processDuration,
			len(processTitles),
			config.DeleteRawMKVFiles,
			totalSizeRaw,
			totalSizeEncoded,
		)
		automationEntries = RunAutomations(selectedAutomations, preRunParams, outputData)
	}

	// Write run manifest (after automations so it can record automation data)
	if !config.DisableManifests {
		manifestDir := config.ManifestDirectory
		if manifestDir == "" {
			var mdErr error
			manifestDir, mdErr = getManifestDir()
			if mdErr != nil {
				fmt.Printf("Warning: could not determine manifest directory: %v\n", mdErr)
				manifestDir = ""
			}
		}
		if manifestDir != "" {
			m := buildManifest(processTitles, manifestEntries, processStartTime, processDuration, config.DeleteRawMKVFiles, appVersion, automationEntries)
			manifestPath, wErr := writeManifest(manifestDir, processStartTime, m)
			if wErr != nil {
				fmt.Printf("Warning: could not write manifest: %v\n", wErr)
			} else {
				fmt.Printf("Manifest written to: %s\n", manifestPath)
			}
		}
	}

	if config.DeleteRawMKVFiles {
		deleteRawFiles(config)
	}

	// Tell the user where the encoded files are located
	fmt.Printf("\nEncoded files are located in: %s\n\n", config.HBOutputDirectory)

	return nil
}

func ripTitles(
	mkv *MakeMKV,
	ctx context.Context,
	tracker *progressTracker,
	processTitles []TitleInfo,
	config *handyMKVConfig,
	encChannel chan EncodingParams,
	cancelProcessing context.CancelFunc) {

	for _, title := range processTitles {
		mkvOutputDirectory := filepath.Join(config.MKVOutputDirectory, title.Subdirectory())
		mkvOutputPath := filepath.Join(mkvOutputDirectory, title.FileName)

		ripStartTime := time.Now()

		// Start progress poller before ripping
		stopPoller := tracker.startProgressPoller(
			ctx,
			title.Index,
			title.DiscId,
			int64(title.FileSizeBytes),
			mkvOutputPath,
		)

		tracker.applyChange(title.Index, title.DiscId, func(status *titleStatus) {
			status.Ripping = InProgress
			status.RippingProgress = 0
			status.OutputFilePath = mkvOutputPath
		})

		ripErr := mkv.ripTitle(ctx, &title, mkvOutputDirectory)

		// Stop the poller regardless of success or failure
		stopPoller()

		if ripErr != nil {
			tracker.setError(ripErr)
			cancelProcessing()
			return
		}

		// Update progress for ripping completion
		tracker.applyChange(title.Index, title.DiscId, func(status *titleStatus) {
			status.Ripping = Complete
			status.RippingProgress = 100
		})
		tracker.forceRefresh() // Force immediate display for completion

		ripDuration := time.Since(ripStartTime).Round(time.Second)
		var rippedSizeBytes int64
		if stat, err := os.Stat(mkvOutputPath); err == nil {
			rippedSizeBytes = stat.Size()
		}

		// Replace spaces with underscores for encoding run.
		encodingOutputFileName := title.GetEncodingFileName(config)

		hbOutputDir := filepath.Join(config.HBOutputDirectory, title.Subdirectory())

		encChannel <- EncodingParams{
			TitleIndex:          title.Index,
			DiscId:              title.DiscId,
			MKVOutputPath:       mkvOutputPath,
			HandBrakeOutputPath: filepath.Join(hbOutputDir, encodingOutputFileName),
			RippedFileSizeBytes: rippedSizeBytes,
			RippingDuration:     ripDuration.String(),
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
func Setup(hb *HandBrakeCLI) error {
	fmt.Printf("What level of configuration would you like to create?\n\n")
	fmt.Println("1 - User-wide configuration (recommended).")
	fmt.Println("2 - Current working directory.")
	fmt.Println()

	configLocationSelectionString := readLine()

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
		config, err = promptForConfig(hb, configLocationSelection)

		if err != nil {
			fmt.Printf("An error occurred while prompting for configuration values: %v\n", err)
			return err
		}

		fmt.Printf("\n%s\n", config.String())
		fmt.Printf("Accept these settings? [y/N]\n\n")

		if strings.ToLower(readLine()) == "y" {
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
