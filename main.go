package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	target  string
	workdir string
	args    string
	icon    string
)

var createCmd = &cobra.Command{
	Use:   "create <output.lnk>",
	Short: "Create a Windows shortcut (.lnk) file",
	Long: `Create a Windows shortcut (.lnk) file from command-line arguments.
No Windows API is used; the .lnk binary format is generated directly.
Works on Windows, Linux, and macOS.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, positional []string) error {
		output := positional[0]
		fmt.Printf("Creating shortcut: %s -> %s\n", output, target)
		return nil
	},
}

var rootCmd = &cobra.Command{
	Use:   "lnkgen",
	Short: "Windows shortcut (.lnk) generator",
}

func init() {
	createCmd.Flags().StringVarP(&target, "target", "t", "", "Target executable or document path (required)")
	createCmd.Flags().StringVarP(&workdir, "workdir", "w", "", "Working directory (Start in)")
	createCmd.Flags().StringVarP(&args, "args", "a", "", "Command-line arguments")
	createCmd.Flags().StringVarP(&icon, "icon", "i", "", "Icon file path (e.g. C:\\path\\to\\icon.ico or exe,0)")
	createCmd.MarkFlagRequired("target")
	rootCmd.AddCommand(createCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
