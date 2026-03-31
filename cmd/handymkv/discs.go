package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(discsCmd())
}

func discsCmd() *cobra.Command {
	runE := func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Detecting available discs...\n\n")

		discs, err := mkv.ListDiscs()
		if err != nil {
			return err
		}

		if len(discs) < 1 {
			fmt.Println("No discs found.")
			return nil
		}

		fmt.Printf("Available discs:\n\n")
		for _, disc := range discs {
			fmt.Printf("Disc - %d - %s\n", disc.Index, disc.Name)
		}
		fmt.Println()
		return nil
	}
	cmd := &cobra.Command{
		Use:     "discs",
		Aliases: []string{"disc"},
		Short:   "List available discs",
		RunE:    runE,
	}

	return cmd
}
