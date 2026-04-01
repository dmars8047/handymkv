package hmkv

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Redo re-encodes titles from a previous run using the current config's encoding settings.
// If runIndex < 0 and/or titleIndices is empty, the user is prompted interactively.
func Redo(hb *HandBrakeCLI, runIndex int, titleIndices []int, allTitles bool, appVersion string) error {
	config, err := ReadConfig()
	if err != nil {
		if err == ErrConfigNotFound {
			return err
		}
		return fmt.Errorf("an unexpected error occurred while reading the configuration file: %w", err)
	}

	files, err := listManifestFiles()
	if err != nil {
		return err
	}

	if len(files) == 0 {
		fmt.Println("No run history found.")
		return nil
	}

	// Interactive run selection
	if runIndex < 0 {
		printRunList(files)
		fmt.Print("Enter the run number to redo: ")
		input := readLine()
		fmt.Println()

		runIndex, err = strconv.Atoi(strings.TrimSpace(input))
		if err != nil {
			return fmt.Errorf("invalid run number: %s", input)
		}
	}

	if runIndex < 1 || runIndex > len(files) {
		return fmt.Errorf("invalid run number %d, valid range: 1-%d", runIndex, len(files))
	}

	m, err := readManifestFile(files[runIndex-1])
	if err != nil {
		return fmt.Errorf("could not read manifest: %w", err)
	}

	// Flatten all titles from the manifest
	var allTitlesFlat []redoTitle
	for _, d := range m.Discs {
		for _, t := range d.Titles {
			subdir := filepath.Base(filepath.Dir(t.RippedFile))
			allTitlesFlat = append(allTitlesFlat, redoTitle{
				discId:   d.DiscId,
				discName: d.DiscName,
				title:    t,
				subdir:   subdir,
			})
		}
	}

	if len(allTitlesFlat) == 0 {
		fmt.Println("No titles found in the selected run.")
		return nil
	}

	// Display titles
	fmt.Printf("Run #%d — %s\n\n", runIndex, m.Date.Local().Format("2006-01-02 15:04:05"))
	for _, rt := range allTitlesFlat {
		fmt.Printf("  [%d] %s (Disc %d: %s, %s)\n",
			rt.title.TitleIndex,
			filepath.Base(rt.title.RippedFile),
			rt.discId,
			rt.discName,
			formatSavedSpace(rt.title.RippedFileSizeBytes),
		)
	}
	fmt.Println()

	// Title selection
	var selected []redoTitle

	if allTitles {
		selected = allTitlesFlat
	} else if len(titleIndices) > 0 {
		indexSet := make(map[int]bool)
		for _, idx := range titleIndices {
			indexSet[idx] = true
		}
		for _, rt := range allTitlesFlat {
			if indexSet[rt.title.TitleIndex] {
				selected = append(selected, rt)
			}
		}
		if len(selected) == 0 {
			return fmt.Errorf("none of the specified title indices were found in the run")
		}
	} else {
		// Interactive title selection
		fmt.Print("Enter title indices to re-encode (0,1,2...) or 'all': ")
		input := strings.TrimSpace(readLine())
		fmt.Println()

		if input == "" {
			fmt.Println("No titles selected. Exiting.")
			return nil
		}

		if input == "all" {
			selected = allTitlesFlat
		} else {
			indexSet := make(map[int]bool)
			for _, raw := range strings.Split(input, ",") {
				idx, err := strconv.Atoi(strings.TrimSpace(raw))
				if err != nil {
					return fmt.Errorf("invalid title index: %s", raw)
				}
				indexSet[idx] = true
			}
			for _, rt := range allTitlesFlat {
				if indexSet[rt.title.TitleIndex] {
					selected = append(selected, rt)
				}
			}
			if len(selected) == 0 {
				return fmt.Errorf("none of the specified title indices were found in the run")
			}
		}
	}

	// Check that ripped files exist
	var missing []string
	for _, rt := range selected {
		if _, err := os.Stat(rt.title.RippedFile); os.IsNotExist(err) {
			missing = append(missing, rt.title.RippedFile)
		}
	}

	if len(missing) > 0 {
		fmt.Println("The following raw MKV files are missing:")
		for _, path := range missing {
			fmt.Printf("  %s\n", path)
		}
		return fmt.Errorf("raw MKV files were deleted — re-rip the disc to re-encode these titles")
	}

	// Create new timestamped output directory
	dirSlug := fmt.Sprintf("handymkv_%s", time.Now().Format("2006-01-02_15-04-05"))
	hbOutputBase := filepath.Join(config.HBOutputDirectory, dirSlug)

	if err := os.MkdirAll(hbOutputBase, 0740); err != nil {
		return fmt.Errorf("could not create output directory: %w", err)
	}

	// Set up progress tracker (ripping already complete)
	tracker := progressTracker{
		statuses:        make([]titleStatus, len(selected)),
		refreshInterval: 200 * time.Millisecond,
	}

	for i, rt := range selected {
		tracker.statuses[i] = titleStatus{
			TitleIndex:        rt.title.TitleIndex,
			Title:             filepath.Base(rt.title.RippedFile),
			DiscId:            rt.discId,
			Ripping:           Complete,
			RippingProgress:   100,
			Encoding:          Pending,
			ExpectedSizeBytes: rt.title.RippedFileSizeBytes,
		}
	}

	ctx, cancelProcessing := context.WithCancel(context.Background())
	defer cancelProcessing()

	processStartTime := time.Now()
	stopRefreshTicker := tracker.startRefreshTicker(ctx)

	// Encode titles sequentially
	var manifestEntries []EncodingParams

	for _, rt := range selected {
		// Derive output filename from ripped file
		outName := filepath.Base(rt.title.RippedFile)
		outName = strings.ReplaceAll(outName, " ", "_")
		if config.EncodeConfig.OutputFileFormat != "" && config.EncodeConfig.OutputFileFormat != "mkv" {
			outName = strings.TrimSuffix(outName, ".mkv") + "." + config.EncodeConfig.OutputFileFormat
		}

		// Preserve subdirectory structure
		outDir := filepath.Join(hbOutputBase, rt.subdir)
		if err := os.MkdirAll(outDir, 0740); err != nil {
			stopRefreshTicker()
			return fmt.Errorf("could not create output subdirectory: %w", err)
		}

		params := EncodingParams{
			TitleIndex:          rt.title.TitleIndex,
			DiscId:              rt.discId,
			MKVOutputPath:       rt.title.RippedFile,
			HandBrakeOutputPath: filepath.Join(outDir, outName),
			RippedFileSizeBytes: rt.title.RippedFileSizeBytes,
			Quality:             config.EncodeConfig.Quality,
			Encoder:             config.EncodeConfig.Encoder,
			EncoderPreset:       config.EncodeConfig.EncoderPreset,
			OutputFileFormat:    config.EncodeConfig.OutputFileFormat,
			Preset:              config.EncodeConfig.Preset,
			PresetFile:          config.EncodeConfig.PresetFile,
			SubtitleLanguages:           config.EncodeConfig.SubtitleLanguages,
			IncludeAllRelevantSubtitles: config.EncodeConfig.IncludeAllRelevantSubtitles,
			AudioLanguages:              config.EncodeConfig.AudioLanguages,
			IncludeAllRelevantAudio:     config.EncodeConfig.IncludeAllRelevantAudio,
		}

		if err := encodeTitle(ctx, hb, &tracker, &params); err != nil {
			stopRefreshTicker()
			return err
		}

		manifestEntries = append(manifestEntries, params)
	}

	stopRefreshTicker()

	if tracker.err != nil {
		return tracker.err
	}

	processDuration := time.Since(processStartTime).Round(time.Second)

	fmt.Printf("\nRedo Complete. Time Elapsed - %s\n", formatTimeElapsedString(processDuration))

	// Calculate and print sizes
	var totalSizeRaw, totalSizeEncoded int64
	for _, entry := range manifestEntries {
		totalSizeRaw += entry.RippedFileSizeBytes
		totalSizeEncoded += entry.EncodedFileSizeBytes
	}

	fmt.Printf("\nTotal size of raw unencoded files - %s\n", formatSavedSpace(totalSizeRaw))
	fmt.Printf("Total size of encoded files - %s\n", formatSavedSpace(totalSizeEncoded))

	if savedSpace := totalSizeRaw - totalSizeEncoded; savedSpace > 0 {
		fmt.Printf("Total disk space saved via encoding - %s\n", formatSavedSpace(savedSpace))
	}

	// Write manifest
	if !config.DisableManifests {
		manifestDir := config.ManifestDirectory
		if manifestDir == "" {
			manifestDir, err = getManifestDir()
			if err != nil {
				fmt.Printf("Warning: could not determine manifest directory: %v\n", err)
				manifestDir = ""
			}
		}
		if manifestDir != "" {
			redoManifest := buildRedoManifest(selected, manifestEntries, processStartTime, processDuration, appVersion)
			manifestPath, wErr := writeManifest(manifestDir, processStartTime, redoManifest)
			if wErr != nil {
				fmt.Printf("Warning: could not write manifest: %v\n", wErr)
			} else {
				fmt.Printf("Manifest written to: %s\n", manifestPath)
			}
		}
	}

	fmt.Printf("\nEncoded files are located in: %s\n\n", hbOutputBase)

	return nil
}

// printRunList displays a numbered list of manifest runs for interactive selection.
func printRunList(files []string) {
	fmt.Printf("Run History\n\n")
	fmt.Printf("  %-4s %-22s %-12s %s\n", "#", "Date", "Duration", "Discs")

	for i, path := range files {
		m, err := readManifestFile(path)
		if err != nil {
			fmt.Printf("  %-4d (error reading manifest: %v)\n", i+1, err)
			continue
		}

		dateStr := m.Date.Local().Format("2006-01-02 15:04:05")
		var discSummaries []string
		for _, d := range m.Discs {
			discSummaries = append(discSummaries, fmt.Sprintf("%s (%d titles)", d.DiscName, len(d.Titles)))
		}

		fmt.Printf("  %-4d %-22s %-12s %s\n", i+1, dateStr, m.Duration, strings.Join(discSummaries, ", "))
	}

	fmt.Println()
}

type redoTitle struct {
	discId   int
	discName string
	title    manifestTitle
	subdir   string // parent directory name of the ripped file
}

// buildRedoManifest builds a manifest struct for a redo run.
func buildRedoManifest(selected []redoTitle, entries []EncodingParams, startTime time.Time, duration time.Duration, appVersion string) *manifest {
	// Group entries by disc
	type discKey struct {
		id   int
		name string
	}

	var discOrder []discKey
	seen := make(map[int]bool)
	discTitles := make(map[int][]manifestTitle)

	for i, rt := range selected {
		if !seen[rt.discId] {
			discOrder = append(discOrder, discKey{id: rt.discId, name: rt.discName})
			seen[rt.discId] = true
		}

		entry := entries[i]
		discTitles[rt.discId] = append(discTitles[rt.discId], manifestTitle{
			TitleIndex:           rt.title.TitleIndex,
			MediaDuration:        rt.title.MediaDuration,
			RippedFile:           rt.title.RippedFile,
			RippedFileSizeBytes:  rt.title.RippedFileSizeBytes,
			EncodedFile:          entry.HandBrakeOutputPath,
			EncodedFileSizeBytes: entry.EncodedFileSizeBytes,
		})
	}

	var discs []manifestDisc
	for _, dk := range discOrder {
		discs = append(discs, manifestDisc{
			DiscId:   dk.id,
			DiscName: dk.name,
			Titles:   discTitles[dk.id],
		})
	}

	return &manifest{
		AppVersion:      appVersion,
		Date:            startTime.UTC(),
		Duration:        duration.String(),
		RawFilesDeleted: false,
		Discs:           discs,
	}
}
