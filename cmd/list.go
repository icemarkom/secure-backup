package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/icemarkom/secure-backup/internal/manifest"
	"github.com/icemarkom/secure-backup/internal/retention"
	"github.com/spf13/cobra"
)

var (
	listDir     string
	listPattern string
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
	listCmd.Flags().StringVar(&listPattern, "pattern", "backup_*.tar.gz.gpg", "Filename pattern to match")

	listCmd.MarkFlagRequired("dest")
}

func runList(cmd *cobra.Command, args []string) error {
	backups, err := retention.ListBackups(listDir, listPattern)
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
			formatAge(backup.Age))
		fmt.Printf("  Size:     %s\n", retention.FormatSize(backup.Size))

		// Try to read manifest
		manifestPath := filepath.Join(listDir,
			strings.TrimSuffix(backup.Name, ".tar.gz.gpg")+".json")
		if m, err := manifest.Read(manifestPath); err == nil {
			fmt.Printf("  Source:   %s\n", m.SourcePath)
			fmt.Printf("  Tool:     %s %s\n", m.CreatedBy.Tool, m.CreatedBy.Version)
			fmt.Printf("  Host:     %s\n", m.CreatedBy.Hostname)
			fmt.Printf("  Checksum: %s...\n", m.ChecksumValue[:16])
		} else {
			fmt.Printf("  (No manifest available)\n")
		}
		fmt.Println()
	}

	return nil
}

// formatAge formats a duration as a human-readable age string
func formatAge(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24

	if days > 0 {
		return fmt.Sprintf("%dd%dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	minutes := int(d.Minutes())
	return fmt.Sprintf("%dm", minutes)
}
