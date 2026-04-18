package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "simplecli",
	Short: "A simple CLI tool",
	Long:  `A simple CLI tool built with Cobra.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello from SimpleCLI!")
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
