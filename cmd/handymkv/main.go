package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"

	"github.com/dmars8047/handymkv/internal/hmkv"
)

var applicationVersion = "dev"

func getVersion() string {
	if applicationVersion != "dev" {
		return applicationVersion
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return applicationVersion
}

func main() {
	// Handle subcommands before flag.Parse()
	if len(os.Args) > 1 && os.Args[1] == "config" {
		hmkv.PrintLogo()

		if len(os.Args) > 2 && os.Args[2] == "edit" {
			configPath, err := hmkv.GetConfigFilePath()
			if err != nil {
				if err == hmkv.ErrConfigNotFound {
					fmt.Printf("No config file found. Run 'handymkv config setup' to create one.\n\n")
					return
				}
				fmt.Printf("Error locating config file: %v\n\n", err)
				return
			}
			if err := openInEditor(configPath); err != nil {
				if errors.Is(err, exec.ErrNotFound) {
					fmt.Printf("No editor found. Set the EDITOR or VISUAL environment variable to specify your preferred editor.\nExample: export EDITOR=nano\n\n")
				} else {
					fmt.Printf("Could not open editor: %v\n\n", err)
				}
			}
			return
		}

		if len(os.Args) > 2 && os.Args[2] == "setup" {
			_, hb, err := checkForPrerequisites()
			if err != nil {
				outputFailedPrerequisiteCheck(err)
				fmt.Print("Please run the configuration wizard again after addressing prerequisite dependency issues.\n\n")
				fmt.Printf("Exiting.\n\n")
				return
			}
			if err := hmkv.Setup(hb); err != nil {
				fmt.Printf("An error occurred during the setup process.\nError: %v\n", err)
			}
			return
		}

		// Default: show current config
		config, err := hmkv.ReadConfig()
		if err != nil {
			if err == hmkv.ErrConfigNotFound {
				fmt.Printf("Config file not found. Please run 'handymkv config setup' to create one.\n\n")
				return
			}
			fmt.Printf("An error occurred while reading the configuration file.\n\nError: %v\n", err)
			return
		}
		fmt.Printf("Configuration file found.\n\n%s\n", config)
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "discs" {
		hmkv.PrintLogo()

		mkv, _, err := checkForPrerequisites()
		if err != nil {
			outputFailedPrerequisiteCheck(err)
			fmt.Print("Exiting.\n\n")
			return
		}

		fmt.Printf("Detecting available discs...\n\n")

		discs, err := mkv.ListDiscs()
		if err != nil {
			fmt.Printf("An error occurred while listing the discs.\n\nError: %v\n", err)
			return
		}

		if len(discs) < 1 {
			fmt.Printf("No discs found.\n\n")
			return
		}

		fmt.Printf("Available discs:\n\n")
		for _, disc := range discs {
			fmt.Printf("Disc - %d - %s\n", disc.Index, disc.Name)
		}
		fmt.Printf("\n")
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "automations" {
		hmkv.PrintLogo()

		if len(os.Args) > 2 {
			switch os.Args[2] {
			case "create":
				if err := hmkv.CreateAutomation(); err != nil {
					fmt.Printf("An error occurred creating automation: %v\n\n", err)
				}
				return
			case "show":
				if len(os.Args) < 4 {
					fmt.Println("Usage: handymkv automations show <name>")
					fmt.Println()
					return
				}
				if err := hmkv.PrintAutomation(os.Args[3]); err != nil {
					fmt.Printf("An error occurred: %v\n\n", err)
				}
				return
			case "delete":
				if len(os.Args) < 4 {
					fmt.Println("Usage: handymkv automations delete <name>")
					fmt.Println()
					return
				}
				if err := hmkv.DeleteAutomation(os.Args[3]); err != nil {
					fmt.Printf("An error occurred: %v\n\n", err)
				}
				return
			}
		}

		if err := hmkv.PrintAutomations(); err != nil {
			fmt.Printf("An error occurred listing automations: %v\n\n", err)
		}
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "history" {
		hmkv.PrintLogo()

		cfg, cfgErr := hmkv.ReadConfig()
		if cfgErr == hmkv.ErrConfigNotFound {
			fmt.Println("No configuration found. Please run the configuration wizard with 'handymkv config setup'.")
			fmt.Println()
			return
		}
		if cfgErr != nil {
			fmt.Printf("An error occurred reading the configuration file: %v\n\n", cfgErr)
			return
		}
		if cfg.DisableManifests {
			fmt.Println("Run history is disabled in your configuration.")
			fmt.Println()
			return
		}

		if len(os.Args) > 2 && os.Args[2] == "clear" {
			if err := hmkv.ClearHistory(); err != nil {
				fmt.Printf("An error occurred clearing run history: %v\n\n", err)
			}
			return
		}

		var index int = -1
		if len(os.Args) > 2 {
			index, _ = strconv.Atoi(os.Args[2])
		}
		if err := hmkv.PrintHistory(index); err != nil {
			fmt.Printf("An error occurred reading run history: %v\n\n", err)
		}
		return
	}

	// Parse command line args
	var discIds string
	var automationNames string
	var version bool

	flag.BoolVar(&version, "v", false, "Version. Prints the version of the application.")
	flag.StringVar(&discIds, "d", "0", "Discs. A comma delimited list of disc indexes to rip. Example: -d 0,1,2")
	flag.StringVar(&automationNames, "a", "", "Automations. A comma delimited list of automation names to run after encoding. Example: -a move-to-plex,notify-discord")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nSubcommands:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  config\n    \tShow the current configuration.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  config setup\n    \tRun the configuration wizard.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  config edit\n    \tOpen the config file in the default editor.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  discs\n    \tList available discs.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  history\n    \tShow a summary list of past runs.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  history <number>\n    \tShow details for a specific past run.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  history clear\n    \tDelete all manifest files from the run history directory.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  automations\n    \tList all saved automations.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  automations create\n    \tCreate a new automation.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  automations show <name>\n    \tShow details of an automation.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  automations delete <name>\n    \tDelete an automation.\n")
	}

	flag.Parse()

	hmkv.PrintLogo()

	if version {
		fmt.Printf("HandyMKV version %s\n\n", getVersion())
		return
	}

	mkv, hb, err := checkForPrerequisites()
	if err != nil {
		outputFailedPrerequisiteCheck(err)
		fmt.Print("Exiting.\n\n")
		return
	}

	discIdSet := make(map[int]struct{})

	for _, rawDiscId := range strings.Split(strings.ReplaceAll(discIds, " ", ""), ",") {

		id, err := strconv.Atoi(rawDiscId)

		if err != nil || id < 0 {
			fmt.Printf("Invalid disc index value detected.\n\nExiting.\n\n")
			return
		}

		discIdSet[id] = struct{}{}
	}

	if len(discIdSet) < 1 {
		fmt.Printf("No valid disc parameters detected.\n\nExiting.\n\n")
		return
	}

	discIdInts := make([]int, 0, len(discIdSet))

	for id := range discIdSet {
		discIdInts = append(discIdInts, id)
	}

	slices.Sort(discIdInts)

	var autoNames []string
	if automationNames != "" {
		seen := make(map[string]struct{})
		for _, name := range strings.Split(automationNames, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				if _, ok := seen[name]; !ok {
					autoNames = append(autoNames, name)
					seen[name] = struct{}{}
				}
			}
		}
	}

	err = hmkv.Exec(mkv, hb, discIdInts, getVersion(), autoNames)
	if err != nil {
		if err == hmkv.ErrConfigNotFound {
			fmt.Printf("Config file not found. Please run the configuration wizard with 'handymkv -c'.\n\n")
			return
		} else if discErr, ok := err.(*hmkv.DiscError); ok {
			fmt.Printf("An error occurred while reading titles from disc %d. Please ensure the disc is inserted and try again.\n\n", discErr.DiscId)
			return
		}

		fmt.Printf("\nAn error occurred during handymkv execution process.\n\nError - %v\n\n", err)

		// If the error is an ExternalProcessError, print the process output
		expErr, isExternalProcessErr := err.(*hmkv.ExternalProcessError)

		if isExternalProcessErr && expErr.ProcessOuput != "" {
			fmt.Print(expErr.ProcessOuput)
		}
	}
}

func outputFailedPrerequisiteCheck(err error) {
	if err != nil {
		switch err {
		case ErrMakeMKVExecNotFound:
			fmt.Print("MakeMKV executable not found.\n\n")
			fmt.Print("Please download, install, and apply a valid license key to MakeMKV. MakeMKV is available for download at https://makemkv.com/download/. ")
			fmt.Print("In most cases MakeMKV will be automatically detected. However, if you have installed it in a non-standard location, make sure the makemkvcon executable is accessible via the $PATH.\n")
			fmt.Print("\nPlease run the configuration wizard again after installing MakeMKV.\n\n")
		case ErrHandBrakeCLIExecNotFound:
			fmt.Print("HandBrakeCLI executable not found.\n\n")
			fmt.Print("The HandBrakeCLI executable is available for download at https://handbrake.fr/downloads2.php. ")
			fmt.Print("The HandBrakeCLI downloaded executable must be accessible via the $PATH. ")

			// Get the users home directory
			usr, err := user.Current()
			if err != nil {
				return
			}

			path := filepath.Join(usr.HomeDir, "handymkv", "bin")

			fmt.Printf("Alternatively, you can place the HandBrakeCLI executable in the following directory: %s. ", path)
			fmt.Print("You may need to create the directory if it does not already exist.\n\n")
		default:
			fmt.Print("An unknown error occurred while checking for prerequisite dependencies.\n\n")
		}
	}
}

// Checks for application prerequisites. Returns an error if a prerequisite is not found.
func checkForPrerequisites() (*hmkv.MakeMKV, *hmkv.HandBrakeCLI, error) {
	mkv, err := hmkv.GetMakeMKVExecutable()
	if err != nil {
		return nil, nil, ErrMakeMKVExecNotFound
	}

	hb, err := hmkv.GetHandBrakeCLIExecutable()
	if err != nil {
		return nil, nil, ErrHandBrakeCLIExecNotFound
	}

	return hmkv.NewMakeMKV(mkv), hmkv.NewHandBrakeCLI(hb), nil
}

var (
	ErrHandBrakeCLIExecNotFound = fmt.Errorf("HandBrakeCLI executable not found")
	ErrMakeMKVExecNotFound      = fmt.Errorf("MakeMKV executable not found")
)

func openInEditor(path string) error {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}

	var cmd *exec.Cmd
	if editor != "" {
		cmd = exec.Command(editor, path)
	} else {
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("cmd", "/c", "start", "", path)
		case "darwin":
			cmd = exec.Command("open", "-t", path)
		default:
			cmd = exec.Command("xdg-open", path)
		}
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
