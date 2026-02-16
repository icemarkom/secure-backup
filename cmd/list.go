package cmd

import (
	"fmt"
	"sort"

	"github.com/icemarkom/secure-backup/internal/format"
	"github.com/icemarkom/secure-backup/internal/manifest"
	"github.com/icemarkom/secure-backup/internal/retention"
	"github.com/spf13/cobra"
)

var (
	listDir string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available backups",
	Long:  `List all backup files in the specified directory with age and size information.`,
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVar(&listDir, "dest", "", "Backup directory to list (required)")

	listCmd.MarkFlagRequired("dest")
}

func runList(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	backups, err := retention.ListBackups(listDir, "")
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		fmt.Printf("No backups found in %s\n", listDir)
		return nil
	}

	// Sort by modification time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModTime.After(backups[j].ModTime)
	})

	fmt.Printf("Found %d backup(s) in %s:\n\n", len(backups), listDir)

	for _, backup := range backups {
		fmt.Printf("%s\n", backup.Name)
		fmt.Printf("  Modified: %s (%s ago)\n",
			backup.ModTime.Format("2006-01-02 15:04"),
			format.Age(backup.Age))
		fmt.Printf("  Size:     %s\n", format.Size(backup.Size))

		// Try to read manifest
		manifestPath := manifest.ManifestPath(backup.Path)
		if m, err := manifest.Read(manifestPath); err == nil {
			fmt.Printf("  Source:   %s\n", m.SourcePath)
			fmt.Printf("  Tool:     %s %s\n", m.CreatedBy.Tool, m.CreatedBy.Version)
			fmt.Printf("  Host:     %s\n", m.CreatedBy.Hostname)
			fmt.Printf("  Checksum: %s\n", m.ChecksumValue)
		} else {
			fmt.Printf("  (No manifest available)\n")
		}
		fmt.Println()
	}

	return nil
}
