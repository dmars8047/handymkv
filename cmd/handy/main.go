package main

import (
	"flag"
	"fmt"
	"os/exec"

	"github.com/dmars8047/handy/internal/handy"
)

const applicationVersion = "0.0.1"

func main() {
	// Parse command line args
	var discId int
	var quality int
	var encoder string
	var version bool
	var readConfig bool

	flag.BoolVar(&version, "v", false, "Version. Prints the version of the application.")

	configFlag := flag.Bool("c", false, "Config. Runs the configuration wizard.")

	flag.IntVar(&quality, "q", -1, "Quality. Sets the quality value to be used for each encoding task. If not provided then the value will be read from cofnig file.")

	flag.StringVar(&encoder, "e", "", "Encoder. If not provided then the value will be read from the config file.")

	flag.IntVar(&discId, "d", 0, "Disc. The disc index to rip. If not provided then disc 0 will be ripped.")

	flag.BoolVar(&readConfig, "r", false, "Read. Reads and outputs the first encountered configuration file. The current working directory is searched first, then the user-level configuration.")

	flag.Parse()

	handy.PrintLogo()

	if version {
		fmt.Printf("Handy version %s\n\n", applicationVersion)
		return
	}

	setup := *configFlag

	if setup {
		err := handy.Setup()

		if err != nil {
			fmt.Printf("An error occurred during the setup process.\nError: %v\n", err)
		}

		return
	}

	if readConfig {
		config, err := handy.ReadConfig()

		if err != nil {
			if err == handy.ErrConfigNotFound {
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

	err = handy.Exec(discId, quality, encoder)

	if err != nil {
		if err == handy.ErrConfigNotFound {
			fmt.Printf("Config file not found. Please run the configuration wizard with 'handy -c'.\n\n")
			return
		} else if err == handy.ErrTitlesDiscRead {
			fmt.Printf("An error occurred while reading titles from disc %d. Please ensure the disc is inserted and try again.\n\n", discId)
			return
		}

		fmt.Printf("An error occurred during handy execution process.\n\nError %v\n\n", err)
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
