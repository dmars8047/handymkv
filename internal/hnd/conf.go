package hnd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	System ConfigFileLocation = iota
	User
	WorkingDirectory
)

const (
	CONFIG_FILE_NAME = "config.json"
)

var ErrConfigNotFound = errors.New("config file not found")

type ConfigFileLocation int

type HandyConfig struct {
	EncodeConfig       EncodingParams `json:"encoding_params"`
	MKVOutputDirectory string         `json:"mkv_output_directory"`
	HBOutputDirectory  string         `json:"handbrake_output_directory"`
}

func (config *HandyConfig) String() string {
	var sb strings.Builder

	sb.WriteString("Encode Settings\n\n")
	sb.WriteString(fmt.Sprintf("Encoder: %s\n", config.EncodeConfig.Encoder))
	sb.WriteString(fmt.Sprintf("Quality: %d\n", config.EncodeConfig.Quality))
	sb.WriteString(fmt.Sprintf("Audio Languages: %s\n", strings.Join(config.EncodeConfig.AudioLanguages, ", ")))
	sb.WriteString(fmt.Sprintf("Include All Relevant Audio: %t\n", config.EncodeConfig.IncludeAllRelevantAudio))
	sb.WriteString(fmt.Sprintf("Subtitle Languages: %s\n", strings.Join(config.EncodeConfig.SubtitleLanguages, ", ")))
	sb.WriteString(fmt.Sprintf("Include All Relevant Subtitles: %t\n", config.EncodeConfig.IncludeAllRelevantSubtitles))
	sb.WriteString("\n")
	sb.WriteString("Output Directories\n\n")
	sb.WriteString(fmt.Sprintf("MKV Output Directory: %s\n", config.MKVOutputDirectory))
	sb.WriteString(fmt.Sprintf("HandBrake Output Directory: %s\n", config.HBOutputDirectory))

	return sb.String()
}

func getUserConfigPath() (string, error) {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable is not set")
		}
		return filepath.Join(appData, "handy", CONFIG_FILE_NAME), nil
	} else {
		usr, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("error getting current user: %w", err)
		}
		return filepath.Join(usr.HomeDir, ".config", "handy", CONFIG_FILE_NAME), nil
	}
}

// Reads the config file and returns a config struct.
func ReadConfig() (*HandyConfig, error) {
	// Check in the current working directory
	filePath := fmt.Sprintf("./%s", CONFIG_FILE_NAME)
	if _, err := os.Stat(filePath); err == nil {
		return readConfigFile(filePath)
	}

	// Check in the user configuration directory
	userConfigPath, err := getUserConfigPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(userConfigPath); err == nil {
		return readConfigFile(userConfigPath)
	}

	return nil, ErrConfigNotFound
}

// Helper function to read and unmarshal the config file
func readConfigFile(filePath string) (*HandyConfig, error) {
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg HandyConfig
	err = json.Unmarshal(fileData, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &cfg, nil
}

// Creates a config file with all global defaults. The file will be written to the specified location.
func CreateConfigFile(location ConfigFileLocation, config *HandyConfig, overwrite bool) error {
	var configPath string

	switch location {
	case System:
		// System-wide config location (not implemented in this example)
		return fmt.Errorf("system-wide config location not supported")
	case User:
		var err error
		configPath, err = getUserConfigPath()
		if err != nil {
			return err
		}
	case WorkingDirectory:
		configPath = fmt.Sprintf("./%s", CONFIG_FILE_NAME)
	default:
		return fmt.Errorf("unknown config file location")
	}

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0740); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	// Check if the config file already exists
	if _, err := os.Stat(configPath); err == nil && !overwrite {
		fmt.Printf("\nA config file already exists at %s. Overwrite? [y/n]\n\n", configPath)
		var choice string
		fmt.Scanln(&choice)
		if strings.ToLower(choice) != "y" {
			fmt.Printf("\nSkipping creation of config file. The file %s already exists.\n", configPath)
			return nil
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("error checking for existing config file: %w", err)
	}

	// Marshal the config struct to JSON
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling config to JSON: %w", err)
	}

	// Write the JSON data to the config file
	if err := os.WriteFile(configPath, configData, 0640); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}
