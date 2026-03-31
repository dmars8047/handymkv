package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/dmars8047/handymkv/internal/hmkv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "handymkv",
	Short: "A MakeMKV + HandBrake productivity tool",
	Long: `
╦ ╦┌─┐┌┐┌┌┬┐┬ ┬╔╦╗╦╔═╦  ╦
╠═╣├─┤│││ ││└┬┘║║║╠╩╗╚╗╔╝
╩ ╩┴ ┴┘└┘─┴┘ ┴ ╩ ╩╩ ╩ ╚╝ 

A MakeMKV + HandBrake productivity tool`,
	Version: getVersion(),
	RunE: func(cmd *cobra.Command, args []string) error {
		err := hmkv.Exec(mkv, hb, discIdInts, getVersion(), autoNames)
		if err != nil {
			if err == hmkv.ErrConfigNotFound {
				return fmt.Errorf("config file not found, please run the configuration wizard with `handymkv config setup`")
			} else if discErr, ok := err.(*hmkv.DiscError); ok {
				return fmt.Errorf("an error occurred while reading titles from disc %d, please ensure the disc is inserted and try again", discErr.DiscId)
			}

			fmt.Printf("\nAn error occurred during handymkv execution process.\n\nError - %v\n\n", err)

			// If the error is an ExternalProcessError, print the process output
			expErr, isExternalProcessErr := err.(*hmkv.ExternalProcessError)

			if isExternalProcessErr && expErr.ProcessOuput != "" {
				fmt.Print(expErr.ProcessOuput)
			}
		}
		return nil
	},
}

var (
	mkv        *hmkv.MakeMKV
	hb         *hmkv.HandBrakeCLI
	discIdInts []int
	autoNames  []string
)

func init() {
	var err error
	mkv, hb, err = checkForPrerequisites()
	if err != nil {
		fmt.Print(outputFailedPrerequisiteCheck(err))
		fmt.Print("Exiting.\n\n")
		os.Exit(1)
	}

	rootCmd.Flags().IntSliceVarP(&discIdInts, "discs", "d", []int{0}, "A comma delimited list of disc indexes to rip. Example: -d 0,1,2")
	rootCmd.Flags().StringSliceVarP(&autoNames, "automations", "a", []string{}, "A comma delimited list of automation names to run after encoding. Example: -a move-to-plex,notify-discord")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// For ldflags
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

func outputFailedPrerequisiteCheck(err error) error {
	if err != nil {
		var sb strings.Builder
		switch err {
		case ErrMakeMKVExecNotFound:
			sb.WriteString("MakeMKV executable not found.\n\n")
			sb.WriteString("Please download, install, and apply a valid license key to MakeMKV. MakeMKV is available for download at https://makemkv.com/download/. ")
			sb.WriteString("In most cases MakeMKV will be automatically detected. However, if you have installed it in a non-standard location, make sure the makemkvcon executable is accessible via the $PATH.\n")
			sb.WriteString("\nPlease run the configuration wizard again after installing MakeMKV.\n\n")
		case ErrHandBrakeCLIExecNotFound:
			sb.WriteString("HandBrakeCLI executable not found.\n\n")
			sb.WriteString("The HandBrakeCLI executable is available for download at https://handbrake.fr/downloads2.php. ")
			sb.WriteString("The HandBrakeCLI downloaded executable must be accessible via the $PATH. ")

			// Get the users home directory
			usr, err := user.Current()
			if err != nil {
				return err
			}

			path := filepath.Join(usr.HomeDir, "handymkv", "bin")

			fmt.Fprintf(&sb, "Alternatively, you can place the HandBrakeCLI executable in the following directory: %s. ", path)
			sb.WriteString("You may need to create the directory if it does not already exist.\n\n")
		default:
			sb.WriteString("An unknown error occurred while checking for prerequisite dependencies.\n\n")
		}
		return fmt.Errorf("%s", sb.String())
	}
	return nil
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
