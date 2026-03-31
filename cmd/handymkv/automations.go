package main

import (
	"github.com/dmars8047/handymkv/internal/hmkv"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(automationsCmd())
}

func automationsCmd() *cobra.Command {
	runE := func(cmd *cobra.Command, args []string) error {
		return hmkv.PrintAutomations()
	}
	cmd := &cobra.Command{
		Use:     "automations",
		Aliases: []string{"automate"},
		Short:   "List all saved automations",
		RunE:    runE,
	}

	cmd.AddCommand(automationsCreateCmd())
	cmd.AddCommand(automationsShowCmd())
	cmd.AddCommand(automationsDeleteCmd())

	return cmd
}

func automationsCreateCmd() *cobra.Command {
	runE := func(cmd *cobra.Command, args []string) error {
		return hmkv.CreateAutomation()
	}
	// TODO: Could probably take in an optional parameter
	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"new"},
		Short:   "Create a new automation",
		RunE:    runE,
	}

	return cmd
}

func automationsShowCmd() *cobra.Command {
	runE := func(cmd *cobra.Command, args []string) error {
		return hmkv.PrintAutomation(args[0])
	}
	cmd := &cobra.Command{
		Use:   "show <name>",
		Args:  cobra.ExactArgs(1),
		Short: "Show details of an automation",
		RunE:  runE,
	}

	return cmd
}

func automationsDeleteCmd() *cobra.Command {
	runE := func(cmd *cobra.Command, args []string) error {
		return hmkv.DeleteAutomation(args[0])
	}
	cmd := &cobra.Command{
		Use:     "delete <name>",
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"rm"},
		Short:   "Delete an automation",
		RunE:    runE,
	}

	return cmd
}
