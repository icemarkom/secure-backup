package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	appVersion = "dev"
	appCommit  = "unknown"
	appDate    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "secure-backup",
	Short: "Secure, encrypted backups for any directory",
	Long: `secure-backup is a tool for creating encrypted, compressed backups 
of any directory with optional Docker volume support.

Features:
  - Native Go implementation (no external dependencies for core functions)
  - GPG encryption support
  - gzip compression
  - Automated retention management
  - Streaming architecture (efficient memory usage)`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("secure-backup %s\n", appVersion)
		fmt.Printf("  commit: %s\n", appCommit)
		fmt.Printf("  built:  %s\n", appDate)
	},
}

// SetVersion sets version information from build-time ldflags
func SetVersion(version, commit, date string) {
	appVersion = version
	appCommit = commit
	appDate = date
}

// GetVersion returns the current application version
func GetVersion() string {
	return appVersion
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(versionCmd)
	// Note: other commands (backup, restore, verify, list) register themselves
	// in their respective init() functions
}
