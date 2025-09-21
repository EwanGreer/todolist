package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "todolist",
	Short: "A beautiful terminal-based todo list application",
	Long: `A beautiful and interactive terminal-based todo list application built with bubbletea.
Features include project filtering, text search, and an intuitive table interface.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}