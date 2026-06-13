package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"lnkgen/internal/lnk"
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

		shortcut := &lnk.LnkFile{
			Target:  target,
			WorkDir: workdir,
			Args:    args,
			Icon:    icon,
		}

		f, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()

		n, err := shortcut.WriteTo(f)
		if err != nil {
			return fmt.Errorf("failed to write shortcut: %w", err)
		}

		fmt.Printf("Created shortcut: %s (%d bytes)\n", output, n)
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
