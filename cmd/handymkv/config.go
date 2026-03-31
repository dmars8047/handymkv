package main

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/dmars8047/handymkv/internal/hmkv"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configCmd())
}

func configCmd() *cobra.Command {
	runE := func(cmd *cobra.Command, args []string) error {
		config, err := hmkv.ReadConfig()
		if err != nil {
			return err
		}

		fmt.Printf("Configuration file found.\n\n%s\n", config)
		return nil
	}
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show current configuration",
		RunE:  runE,
	}
	cmd.AddCommand(configSetupCmd())
	cmd.AddCommand(configEditCmd())

	return cmd
}

func configSetupCmd() *cobra.Command {
	runE := func(cmd *cobra.Command, args []string) error {
		return hmkv.Setup(hb)
	}
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Run the configuration wizard",
		RunE:  runE,
	}

	return cmd
}

func configEditCmd() *cobra.Command {
	runE := func(cmd *cobra.Command, args []string) error {
		configPath, err := hmkv.GetConfigFilePath()
		if err != nil {
			if err == hmkv.ErrConfigNotFound {
				return fmt.Errorf("no config file found, run 'handymkv config setup' to create one")
			}
			return fmt.Errorf("error locating config file: %w", err)
		}
		if err := openInEditor(configPath); err != nil {
			if errors.Is(err, exec.ErrNotFound) {
				return fmt.Errorf("no editor found, set the EDITOR or VISUAL environment variable to specify your preferred editor\nExample: export EDITOR=nano")
			}
			return fmt.Errorf("could not open editor: %w", err)
		}
		return nil
	}
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Open the config file in the default editor",
		RunE:  runE,
	}

	return cmd
}
