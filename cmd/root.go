package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "backup-docker",
	Short: "Encrypted backup tool for Docker volumes",
	Long: `backup-docker is a tool for creating encrypted, compressed backups 
of directories and Docker volumes.

Features:
  - Native Go implementation (no external dependencies for core functions)
  - GPG encryption support
  - gzip compression
  - Docker volume backup support
  - Automated retention management`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags can be added here if needed
}
