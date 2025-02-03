package main

import (
	"flag"
	"fmt"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/dmars8047/handymkv/internal/hmkv"
)

const applicationVersion = "0.1.8"

func main() {
	// Parse command line args
	var discIds string
	var version bool
	var readConfig bool
	var configure bool
	var listDiscs bool

	flag.BoolVar(&version, "v", false, "Version. Prints the version of the application.")
	flag.BoolVar(&configure, "c", false, "Configure. Runs the configuration wizard.")
	flag.BoolVar(&readConfig, "r", false, "Read. Reads and outputs the first encountered configuration file. The current working directory is searched first, then the user-level configuration.")
	flag.BoolVar(&listDiscs, "l", false, "List. Lists the available discs. The disc index is required to rip a disc. Drives without a valid disc inserted will not be listed.")
	flag.StringVar(&discIds, "d", "0", "Discs. A comma delimited list of disc indexes to rip. Example: -d 0,1,2")

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
		fmt.Printf("Prerequisite not found or inaccessible. Make sure makemkvcon and HandBrakeCLI are accessible via the PATH.\n\nExiting.\n\n")
		return
	}

	if listDiscs {
		fmt.Printf("Detecting available discs...\n\n")

		discs, err := hmkv.ListDiscs()

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

	err = hmkv.Exec(discIdInts)

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
