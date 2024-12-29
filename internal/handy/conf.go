package handy

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
	System configFileLocation = iota
	User
	WorkingDirectory
)

const (
	configFileName = "config.json"
)

var ErrConfigNotFound = errors.New("config file not found")

type configFileLocation int

type handyConfig struct {
	EncodeConfig       EncodingParams `json:"encoding_params"`
	MKVOutputDirectory string         `json:"mkv_output_directory"`
	HBOutputDirectory  string         `json:"handbrake_output_directory"`
	DeleteRawMKVFiles  bool           `json:"delete_raw_mkv_files"`
}

func (config *handyConfig) String() string {
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
	sb.WriteString(fmt.Sprintf("Automatically Delete Raw MKV Files: %t\n", config.DeleteRawMKVFiles))

	return sb.String()
}

func getUserConfigPath() (string, error) {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable is not set")
		}
		return filepath.Join(appData, "handy", configFileName), nil
	} else {
		usr, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("error getting current user: %w", err)
		}
		return filepath.Join(usr.HomeDir, ".config", "handy", configFileName), nil
	}
}

// Reads the config file and returns a config struct.
func ReadConfig() (*handyConfig, error) {
	// Check in the current working directory
	filePath := fmt.Sprintf("./%s", configFileName)
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
func readConfigFile(filePath string) (*handyConfig, error) {
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg handyConfig
	err = json.Unmarshal(fileData, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &cfg, nil
}

// Creates a config file with all global defaults. The file will be written to the specified location.
func createConfigFile(location configFileLocation, config *handyConfig, overwrite bool) error {
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
		configPath = fmt.Sprintf("./%s", configFileName)
	default:
		return fmt.Errorf("unknown config file location")
	}

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0740); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	// Check if the config file already exists
	if _, err := os.Stat(configPath); err == nil && !overwrite {
		fmt.Printf("\nA config file already exists at %s. Overwrite? [y/N]\n\n", configPath)
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

// Prompts the user for configuration values and returns a new HandyConfig object.
func promptForConfig(locationSelection int) *handyConfig {
	var config handyConfig

	clear()
	fmt.Printf("You will now answer a series of questions to provide default values for your configuration. Press enter to select the indicated default. For more information, please consult the project documentation.\n\n(Press enter to continue)\n\n")

	// Press enter to continue
	fmt.Scanln()
	clear()

	config.EncodeConfig.Encoder = promptForString("What encoder should be used by default?", "", defaultEncoder, possibleEncoderValues)
	clear()
	config.EncodeConfig.Quality = promptForInt("What should the default quality be set to?", "", defaultQuality)
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

	config.DeleteRawMKVFiles = promptForBool("Automatically delete raw unencoded files after ripping/encoding operations?",
		"If enabled, raw unencoded mkv files will be deleted after the ripping/encoding operation completes. If disabled, raw unencoded files will be retained. Leaving this option enabled is recommended as it will save space on the disk.",
		true)

	clear()

	return &config
}
