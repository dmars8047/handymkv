package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	INPUT_DIR_NAME  = "Input"
	OUTPUT_DIR_NAME = "Output"
)

func main() {
	setupFlag := flag.Bool("s", false, "Setup. Creates the input and output directory. Will not overwrite if they exist.")

	flag.Parse()

	setup := *setupFlag

	if setup {
		fmt.Println("Creating setup directories...")
		setupErr := createDirectories(INPUT_DIR_NAME, OUTPUT_DIR_NAME)

		if setupErr != nil {
			fmt.Printf("An error occurred while setting up directories.\nError: %v\n", setupErr)
			return
		}

		fmt.Printf("Directory setup complete.\n")
		return
	}

	_, err := exec.LookPath("HandBrakeCLI")

	if err != nil {
		fmt.Println("HandBrakeCLI not found in $PATH. Please install the HandBrakeCLI. Exiting.")
		return
	}

	inputDirInfo, err := os.Stat(fmt.Sprintf("./%s", INPUT_DIR_NAME))

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Printf("Input directory does not exist. Run handy -s and load the resulting '%s' directory with .mkv files to be encoded.", INPUT_DIR_NAME)
		} else {
			fmt.Printf("An error occurred while scanning for a preexisting %s directory.\nError: %v\n", INPUT_DIR_NAME, err)
		}

		return
	}

	// Check to make sure that the input directory contains .mkv files
	inputDirEntries, err := os.ReadDir(inputDirInfo.Name())

	if err != nil {
		fmt.Printf("An error occurred while reading files from the input directory.\nError: %v\n", err)
		return
	}

	var mkvDirEntries []fs.DirEntry = make([]fs.DirEntry, 0)

	for _, entry := range inputDirEntries {
		if strings.HasSuffix(entry.Name(), ".mkv") && !entry.IsDir() {
			mkvDirEntries = append(mkvDirEntries, entry)
		}
	}

	if len(mkvDirEntries) == 0 {
		fmt.Printf("No .mkv files found in input directory. Exiting.\n")
		return
	}

	outputDirInfo, err := os.Stat(fmt.Sprintf("./%s", OUTPUT_DIR_NAME))

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Printf("Output directory does not exist in the current working directory. Creating the output directory...\n")

			err := createDirectories(OUTPUT_DIR_NAME)

			if err != nil {
				fmt.Printf("An error occurred while creating output directory.\nError: %v\n", err)
				return
			}
		} else {
			fmt.Printf("An error occurred while scanning for a preexisting %s directory: %v\n", INPUT_DIR_NAME, err)
			return
		}
	}

	cumulativeStartTime := time.Now()

	for _, mkvFile := range mkvDirEntries {

		outputFileRelPath := fmt.Sprintf("./%s/%s", outputDirInfo.Name(), mkvFile.Name())

		fmt.Printf("Encoding: %s\n", mkvFile.Name())

		// HandBrakeCLI --input 50_First_Dates.mkv --output output.mkv --encoder nvenc_h264 --quality 18 --subtitle-lang-list English --all-subtitles --audio-lang-list English --all-audio
		handbrakeCommand := exec.Command("HandBrakeCLI",
			"--input", fmt.Sprintf("./%s/%s", inputDirInfo.Name(), mkvFile.Name()),
			"--output", outputFileRelPath,
			"--encoder", "nvenc_h264",
			"--quality", "18",
			"--subtitle-lang-list", "eng,jpn",
			"--all-subtitles",
			"--audio-lang-list", "eng",
			"--all-audio",
		)

		// Get stdout pipe
		stdoutPipe, err := handbrakeCommand.StdoutPipe()

		if err != nil {
			fmt.Printf("An error occurred when creating stdout pipe from HandBrakeCLI. Aborting remaining encoding tasks.\nError:  %v\n", err)
			return
		}

		encodeStartTime := time.Now()

		// Start the command
		if err := handbrakeCommand.Start(); err != nil {
			fmt.Printf("An error when spawning HandBrakeCLI process. Aborting remaining encoding tasks.\nError: %v\n", err)
			return
		}

		// Read the output from the pipe
		buf := make([]byte, 1024)

		for {
			n, err := stdoutPipe.Read(buf)
			if n > 0 {
				fmt.Print(string(buf[:n]))
			}
			if err != nil {
				break
			}
		}

		// Wait for the command to finish
		if err := handbrakeCommand.Wait(); err != nil {
			fmt.Printf("An error occurred during sqawned HandBrakeCLI execution.\nError: %v\n", err)
			return
		}

		encodeDuration := time.Since(encodeStartTime).Round(time.Second)

		outputFileAbsPath, err := filepath.Abs(outputFileRelPath)

		if err != nil {
			fmt.Printf("Error getting full path of output file '%s'. Aborting remaining encoding tasks.\nError: %v\n", outputDirInfo.Name(), err)
			return
		}

		fmt.Printf("Encode Complete: Output file - %s\n", outputFileAbsPath)
		fmt.Printf("Time Elapsed: %s\n", timeElapsedString(encodeDuration))
	}

	cumulativeDuration := time.Since(cumulativeStartTime).Round(time.Second)
	fmt.Printf("\nEncoding Tasks Complete: See results in the '%s' directory.\n", outputDirInfo.Name())
	fmt.Printf("Total Time Elapsed: %s\n", timeElapsedString(cumulativeDuration))
}

func timeElapsedString(m time.Duration) string {
	minutes := int(math.Floor(m.Minutes()))
	seconds := int(math.Floor(m.Seconds())) - (minutes * 60)
	return fmt.Sprintf("%dm%ds", minutes, seconds)
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
