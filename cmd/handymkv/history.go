package main

import (
	"fmt"
	"strconv"

	"github.com/dmars8047/handymkv/internal/hmkv"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(historyCmd())
}

func historyCmd() *cobra.Command {
	runE := func(cmd *cobra.Command, args []string) error {
		index := -1
		if len(args) > 0 {
			index, _ = strconv.Atoi(args[0])
		}
		return hmkv.PrintHistory(index)
	}
	cmd := &cobra.Command{
		Use:   "history [number]",
		Args:  cobra.RangeArgs(0, 1),
		Short: "Show a summary list of past runs",
		RunE:  runE,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, cfgErr := hmkv.ReadConfig()
			if cfgErr == hmkv.ErrConfigNotFound {
				return fmt.Errorf("no configuration found, please run the configuration wizard with `handymkv config setup`")
			}
			if cfgErr != nil {
				return fmt.Errorf("an error occurred reading the configuration file: %w", cfgErr)
			}
			if cfg.DisableManifests {
				return fmt.Errorf("run history is disabled in your configuration")
			}
			return nil
		},
	}

	cmd.AddCommand(historyClearCmd())

	return cmd
}

func historyClearCmd() *cobra.Command {
	runE := func(cmd *cobra.Command, args []string) error {
		return hmkv.ClearHistory()
	}
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Delete all manifest files from the run history directory",
		RunE:  runE,
	}

	return cmd
}
