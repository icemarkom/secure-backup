// Copyright 2026 Marko Milivojevic
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sort"

	"github.com/icemarkom/secure-backup/internal/common"
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
	backups, err := retention.ListBackups(listDir)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		fmt.Printf("No backups found in %s\n", listDir)
		return nil
	}

	// Partition into managed (with manifest) and orphan (without)
	type managedBackup struct {
		info     retention.BackupInfo
		manifest *manifest.Manifest
	}
	var managed []managedBackup
	var orphans []retention.BackupInfo

	for _, backup := range backups {
		manifestPath := manifest.ManifestPath(backup.Path)
		if m, err := manifest.Read(manifestPath); err == nil {
			managed = append(managed, managedBackup{info: backup, manifest: m})
		} else {
			orphans = append(orphans, backup)
		}
	}

	// Sort each group by modification time (newest first)
	sort.Slice(managed, func(i, j int) bool {
		return managed[i].info.ModTime.After(managed[j].info.ModTime)
	})
	sort.Slice(orphans, func(i, j int) bool {
		return orphans[i].ModTime.After(orphans[j].ModTime)
	})

	fmt.Printf("Found %d backup(s) in %s:\n", len(backups), listDir)

	// Display managed backups
	if len(managed) > 0 {
		fmt.Printf("\n=== Managed Backups (%d) ===\n", len(managed))
		for _, mb := range managed {
			fmt.Printf("\n%s\n", mb.info.Name)
			fmt.Printf("  Source:   %s\n", mb.manifest.SourcePath)
			fmt.Printf("  Host:     %s\n", mb.manifest.CreatedBy.Hostname)
			fmt.Printf("  Modified: %s (%s ago)\n",
				mb.info.ModTime.Format("2006-01-02 15:04"),
				common.Age(mb.info.Age))
			fmt.Printf("  Size:     %s\n", common.Size(mb.info.Size))
			fmt.Printf("  Tool:     %s %s\n", mb.manifest.CreatedBy.Tool, mb.manifest.CreatedBy.Version)
			fmt.Printf("  Checksum: %s\n", mb.manifest.ChecksumValue)
		}
	}

	// Display orphan backups
	if len(orphans) > 0 {
		fmt.Printf("\n=== Orphan Backups (%d) — no manifest, limited info ===\n", len(orphans))
		for _, backup := range orphans {
			fmt.Printf("\n%s\n", backup.Name)
			fmt.Printf("  Modified: %s (%s ago)\n",
				backup.ModTime.Format("2006-01-02 15:04"),
				common.Age(backup.Age))
			fmt.Printf("  Size:     %s\n", common.Size(backup.Size))
			fmt.Printf("  (no manifest)\n")
		}
	}

	fmt.Println()
	return nil
}
