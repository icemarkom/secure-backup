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
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	appVersion = "dev"
	appCommit  = "unknown"
	appDate    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:           "secure-backup",
	Short:         "Secure, encrypted backups for any directory",
	Long:          `Create GPG-encrypted, compressed backups of any directory with automated retention management.`,
	SilenceErrors: true,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
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

// ExecuteContext runs the root command with a context for signal handling
func ExecuteContext(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	rootCmd.AddCommand(versionCmd)
	// Note: other commands (backup, restore, verify, list) register themselves
	// in their respective init() functions
}
