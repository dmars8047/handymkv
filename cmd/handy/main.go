package main

import (
	"flag"
	"fmt"
	"os/exec"

	"github.com/dmars8047/handy/internal/hnd"
)

func main() {
	// Parse command line args
	var discId int
	var quality int
	var encoder string

	configFlag := flag.Bool("c", false, "Config. Runs the configuration wizard.")

	flag.IntVar(&quality, "q", -1, fmt.Sprintf(`Quality. 
	Sets the quality value to be used for each encoding task. If not provided then the value will be read from the '%s' file. 
	If the config file cannot be found then then a default value of '%d' will be used.`, hnd.CONFIG_FILE_NAME, hnd.DEFAULT_QUALITY))

	flag.StringVar(&encoder, "e", "", fmt.Sprintf(`Encoder. 
	If not provided then the value will be read from the '%s' file. 
	If the config file cannot be found then then a default value of '%s' will be used.`, hnd.CONFIG_FILE_NAME, hnd.DEFAULT_ENCODER))

	flag.IntVar(&discId, "d", 0, "Disc. The disc index to rip. If not provided then disc 0 will be ripped.")

	flag.Parse()

	hnd.PrintLogo()

	setup := *configFlag

	if setup {
		err := hnd.Setup()

		if err != nil {
			fmt.Printf("An error occurred during the setup process.\nError: %v\n", err)
		}

		return
	}

	err := checkForPrerequisites()

	if err != nil {
		fmt.Printf("Prerequisite not found or inaccessible. Make sure makemkvcon and HandBrakeCLI are accessible via the PATH.\nExiting.\n")
		return
	}

	err = hnd.Exec(discId, quality, encoder)

	if err != nil {
		if err == hnd.ErrConfigNotFound {
			fmt.Printf("Config file not found. Please run the configuration wizard with 'handy -c'.\n\n")
			return
		} else if err == hnd.ErrTitlesDiscRead {
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
