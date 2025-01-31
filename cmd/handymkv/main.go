package main

import (
	"flag"
	"fmt"
	"os/exec"

	"github.com/dmars8047/handymkv/internal/hmkv"
)

const applicationVersion = "0.1.6"

func main() {
	// Parse command line args
	var discId int
	var version bool
	var readConfig bool
	var configure bool

	flag.BoolVar(&version, "v", false, "Version. Prints the version of the application.")

	flag.BoolVar(&configure, "c", false, "Configure. Runs the configuration wizard.")

	flag.IntVar(&discId, "d", 0, "Disc. The disc index to rip. If not provided then disc 0 will be ripped.")

	flag.BoolVar(&readConfig, "r", false, "Read. Reads and outputs the first encountered configuration file. The current working directory is searched first, then the user-level configuration.")

	flag.Parse()

	hmkv.PrintLogo()

	if version {
		fmt.Printf("HandyMKV version %s\n\n", applicationVersion)
		return
	}

	if configure {
		err := checkForPrerequisites()

		if err != nil {
			fmt.Printf("Prerequisite not found or inaccessible. Make sure makemkvcon and HandBrakeCLI are accessible via the PATH.\nExiting.\n")
			return
		}

		err = hmkv.Setup()

		if err != nil {
			fmt.Printf("An error occurred during the setup process.\nError: %v\n", err)
		}

		return
	}

	if readConfig {
		config, err := hmkv.ReadConfig()

		if err != nil {
			if err == hmkv.ErrConfigNotFound {
				fmt.Printf("Config file not found. Please run the configuration wizard with 'handy -c'.\n\n")
				return
			}

			fmt.Printf("An error occurred while reading the configuration file.\n\nError: %v\n", err)
			return
		}

		fmt.Printf("Configuration file found.\n\n%+v\n", config)

		return
	}

	err := checkForPrerequisites()

	if err != nil {
		fmt.Printf("Prerequisite not found or inaccessible. Make sure makemkvcon and HandBrakeCLI are accessible via the PATH.\nExiting.\n")
		return
	}

	err = hmkv.Exec(discId)

	if err != nil {
		if err == hmkv.ErrConfigNotFound {
			fmt.Printf("Config file not found. Please run the configuration wizard with 'handymkv -c'.\n\n")
			return
		} else if err == hmkv.ErrTitlesDiscRead {
			fmt.Printf("An error occurred while reading titles from disc %d. Please ensure the disc is inserted and try again.\n\n", discId)
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

// Checks for application prerequisites. Returns an error if a prerequisite is not found.
func checkForPrerequisites() error {
	_, err := exec.LookPath("makemkvcon")

	if err != nil {
		fmt.Printf("makemkvcon not found in $PATH. Please install the makemkvcon.\n")
		return err
	}

	_, err = exec.LookPath("HandBrakeCLI")

	if err != nil {
		fmt.Printf("HandBrakeCLI not found in $PATH. Please install the HandBrakeCLI.\n")
		return err
	}

	return nil
}
