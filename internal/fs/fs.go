package fs

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dmars8047/handy/internal/hb"
)

const (
	CONFIG_FILE_NAME = ".handy_config.json"
)

type HandyConfig struct {
	EncodeConfig       hb.EncodingParams `json:"encoding_params"`
	MKVOutputDirectory string            `json:"mkv_output_directory"`
	HBOutputDirectory  string            `json:"handbrake_output_directory"`
}

// Creates a config file with all global defaults. The file will be written to the current working directory.
func CreateConfigFile() error {
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

	defaultConfig := HandyConfig{
		EncodeConfig: hb.EncodingParams{
			Quality:           hb.DEFAULT_QUALITY,
			Encoder:           hb.DEFAULT_ENCODER,
			SubtitleLanguages: []string{"eng"},
			AudioLanguages:    []string{"eng", "jpn"},
		},
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
func ReadConfig() (*HandyConfig, error) {
	filePath := fmt.Sprintf("./%s", CONFIG_FILE_NAME)

	// Check if the file exists
	if _, err := os.Stat(filePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Config file does not exist, return default config
			defaultConfig := HandyConfig{
				EncodeConfig: hb.EncodingParams{
					Quality:           hb.DEFAULT_QUALITY,
					Encoder:           hb.DEFAULT_ENCODER,
					SubtitleLanguages: []string{"eng"},
					AudioLanguages:    []string{"eng", "jpn"},
				},
			}

			return &defaultConfig, nil
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
	var cfg HandyConfig
	err = json.Unmarshal(fileData, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &cfg, nil
}

// Takes in the list of directory names to create. Returns a full path to the input file and a full path to the output file.
func CreateDirectories(dirNames ...string) error {
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
func GetFileSize(filePath string) (int64, error) {
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

// Deletes a file
func DeleteFile(filePath string) error {
	return os.Remove(filePath)
}