package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/dmars8047/handy/internal/ui"
	"github.com/rivo/tview"
)

const (
	INPUT_DIR_NAME   = "Input"
	OUTPUT_DIR_NAME  = "Output"
	DEFAULT_QUALITY  = 20     // 18
	DEFAULT_ENCODER  = "h264" //nvenc_h264
	CONFIG_FILE_NAME = ".handy_config.json"
)

var possibleEncoderValues map[string]struct{} = map[string]struct{}{
	"svt_av1":          {},
	"svt_av1_10bit":    {},
	"x264":             {},
	"x264_10bit":       {},
	"nvenc_h264":       {},
	"x265":             {},
	"x265_10bit":       {},
	"x265_12bit":       {},
	"nvenc_h265":       {},
	"nvenc_h265_10bit": {},
	"mpeg4":            {},
	"mpeg2":            {},
	"VP8":              {},
	"VP9":              {},
	"VP9_10bit":        {},
	"theora":           {},
}

func main() {
	var quality int
	var encoder string

	setupFlag := flag.Bool("s", false, fmt.Sprintf(`Setup. 
	Creates the %s and %s directory as well as a %s file with sensible defaults in the current working directory. 
	Will not overwrite if they exist.`, INPUT_DIR_NAME, OUTPUT_DIR_NAME, CONFIG_FILE_NAME))

	listFlag := flag.Bool("l", false, `List Encoders. 
	Displays a list of all valid encoder options.`)

	flag.IntVar(&quality, "q", 0, fmt.Sprintf(`Quality. 
	Sets the quality value to be used for each encoding task. If not provided then the value will be read from the '%s' file. 
	If the config file cannot be found then then a default value of '%d' will be used.`, CONFIG_FILE_NAME, DEFAULT_QUALITY))

	flag.StringVar(&encoder, "e", "", fmt.Sprintf(`Encoder. 
	If not provided then the value will be read from the '%s' file. 
	If the config file cannot be found then then a default value of '%s' will be used.`, CONFIG_FILE_NAME, DEFAULT_ENCODER))

	flag.Parse()

	setup := *setupFlag

	if setup {
		fmt.Println("Creating setup directories...")
		setupErr := createDirectories(INPUT_DIR_NAME, OUTPUT_DIR_NAME)

		if setupErr != nil {
			fmt.Printf("An error occurred while setting up directories.\nError: %v\n", setupErr)
			return
		}

		fmt.Println("Creating config file...")

		setupErr = createConfigFile()

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
		for k := range possibleEncoderValues {
			fmt.Printf("%s\n", k)
		}

		fmt.Printf("\nNote. This is a list of all possible values. Your installation may or may not be configured with all of these.\n")
		return
	}

	if encoder != "" {
		_, encoderAllowed := possibleEncoderValues[encoder]

		if !encoderAllowed {
			fmt.Printf("The provided encoder '%s' does not match any known encoder. Use handy -l to get a list of all valid encoder options.\n", encoder)
			return
		}
	}

	if quality < 0 {
		fmt.Printf("Invalid quality value detected. Setting to default value - %d\n", DEFAULT_QUALITY)
		quality = DEFAULT_QUALITY
	}

	// _, err := exec.LookPath("HandBrakeCLI")

	// if err != nil {
	// 	fmt.Println("HandBrakeCLI not found in $PATH. Please install the HandBrakeCLI. Exiting.")
	// 	return
	// }

	// config, err := readConfig()

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

	app := tview.NewApplication()
	nav := ui.NewNavigator(context.Background())

	homePage := &ui.HomePage{}
	homePage.Setup(app, nav)

	// Start the application.
	err := app.SetRoot(nav.Pages, true).Run()

	if err != nil {
		panic(err)
	}

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

func formatTimeElapsedString(m time.Duration) string {
	minutes := int(math.Floor(m.Minutes()))
	seconds := int(math.Floor(m.Seconds())) - (minutes * 60)
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}

type config struct {
	Encoder string `json:"encoder"`
	Quality int    `json:"quality"`
}

// Creates a config file with all global defaults. The file will be written to the current working directory.
func createConfigFile() error {
	var exists bool = true

	fileInfo, err := os.Stat(fmt.Sprintf("./%s", CONFIG_FILE_NAME))

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			exists = false
		} else {
			fmt.Printf("An error occurred while scanning for a preexisting %s file.\n", CONFIG_FILE_NAME)
			return err
		}
	}

	if exists {
		fmt.Printf("Skipping creation of config file. The file %s already exists.\n", fileInfo.Name())
		return nil
	}

	defaultConfig := config{
		Encoder: DEFAULT_ENCODER,
		Quality: DEFAULT_QUALITY,
	}

	defaultConfigBytes, err := json.Marshal(&defaultConfig)

	if err != nil {
		return err
	}

	err = os.WriteFile(CONFIG_FILE_NAME, defaultConfigBytes, 0750)

	if err != nil {
		return err
	}

	return nil
}

// Reads the config file and returns a config struct.
// If the config file does not exist, it returns a config with global defaults.
func readConfig() (*config, error) {
	filePath := fmt.Sprintf("./%s", CONFIG_FILE_NAME)

	// Check if the file exists
	if _, err := os.Stat(filePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Config file does not exist, return default config
			return &config{
				Encoder: DEFAULT_ENCODER,
				Quality: DEFAULT_QUALITY,
			}, nil
		}
		// Some other error occurred
		return nil, fmt.Errorf("error checking config file: %w", err)
	}

	// Read the file
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Unmarshal the file data into the config struct
	var cfg config
	err = json.Unmarshal(fileData, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &cfg, nil
}

// Takes in the list of directory names to create. Returns a full path to the input file and a full path to the output file.
func createDirectories(dirNames ...string) error {
	for _, dirName := range dirNames {
		relativeDirString := fmt.Sprintf("./%s", dirName)

		fileInfo, err := os.Stat(relativeDirString)

		var exists bool = true

		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				exists = false
			} else {
				fmt.Printf("An error occurred while scanning for a preexisting %s directory.\n", dirName)
				return err
			}
		}

		if exists {
			fmt.Printf("Skipping creation of %s directory. Entry '%s' already exists.\n", dirName, fileInfo.Name())
			continue
		}

		fmt.Printf("Creating %s directory...\n", dirName)

		err = os.Mkdir(relativeDirString, 0755)

		if err != nil {
			fmt.Printf("An error occurred while creating %s directory.\n", dirName)
			return err
		}

		fileInfo, err = os.Stat(relativeDirString)

		if err != nil {
			fmt.Printf("An error occurred while scanning for resulting %s directory.\n", dirName)
			return err
		}

		fullPath, err := filepath.Abs(fileInfo.Name())

		if err != nil {
			fmt.Printf("An error occurred while reading the full path for resulting %s directory.\n", dirName)
			return err
		}

		fmt.Printf("Created directory: %s\n", fullPath)
	}

	return nil
}

// Reads the size of the specified file and returns it in bytes.
// If the file does not exist or an error occurs, it returns an error.
func getFileSize(filePath string) (int64, error) {
	// Get file info using os.Stat
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, fmt.Errorf("file does not exist: %s", filePath)
		}
		return 0, fmt.Errorf("error retrieving file info: %w", err)
	}

	// Return the size of the file
	return fileInfo.Size(), nil
}
