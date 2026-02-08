package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test flag defaults
func TestBackupCommand_FlagDefaults(t *testing.T) {
	cmd := backupCmd

	compression, _ := cmd.Flags().GetString("compression")
	assert.Equal(t, "gzip", compression, "default compression should be gzip")

	encryption, _ := cmd.Flags().GetString("encryption")
	assert.Equal(t, "gpg", encryption, "default encryption should be gpg")

	retention, _ := cmd.Flags().GetInt("retention")
	assert.Equal(t, 0, retention, "default retention should be 0 (keep all)")

	verbose, _ := cmd.Flags().GetBool("verbose")
	assert.False(t, verbose, "default verbose should be false")
}

func TestRestoreCommand_FlagDefaults(t *testing.T) {
	cmd := restoreCmd

	verbose, _ := cmd.Flags().GetBool("verbose")
	assert.False(t, verbose, "default verbose should be false")
}

func TestVerifyCommand_FlagDefaults(t *testing.T) {
	cmd := verifyCmd

	quick, _ := cmd.Flags().GetBool("quick")
	assert.False(t, quick, "default quick should be false")

	verbose, _ := cmd.Flags().GetBool("verbose")
	assert.False(t, verbose, "default verbose should be false")
}

func TestListCommand_FlagDefaults(t *testing.T) {
	cmd := listCmd

	pattern, _ := cmd.Flags().GetString("pattern")
	assert.Equal(t, "backup_*.tar.gz.gpg", pattern, "default pattern should match backup files")
}

func TestBackupCommand_VerboseFlag(t *testing.T) {
	cmd := backupCmd

	// Parse flags
	cmd.ParseFlags([]string{"-v"})
	verbose, _ := cmd.Flags().GetBool("verbose")
	assert.True(t, verbose)

	// Reset
	cmd.Flags().Set("verbose", "false")

	// Test long form
	cmd.ParseFlags([]string{"--verbose"})
	verbose, _ = cmd.Flags().GetBool("verbose")
	assert.True(t, verbose)
}

func TestRestoreCommand_VerboseFlag(t *testing.T) {
	cmd := restoreCmd

	cmd.ParseFlags([]string{"-v"})
	verbose, _ := cmd.Flags().GetBool("verbose")
	assert.True(t, verbose)
}

func TestBackupCommand_RetentionFlag(t *testing.T) {
	cmd := backupCmd

	// Test setting retention
	cmd.ParseFlags([]string{"--retention", "30"})
	retention, _ := cmd.Flags().GetInt("retention")
	assert.Equal(t, 30, retention)
}

func TestVerifyCommand_QuickFlag(t *testing.T) {
	cmd := verifyCmd

	cmd.ParseFlags([]string{"--quick"})
	quick, _ := cmd.Flags().GetBool("quick")
	assert.True(t, quick)
}

func TestBackupCommand_CompressionFlag(t *testing.T) {
	cmd := backupCmd

	cmd.ParseFlags([]string{"--compression", "gzip"})
	compression, _ := cmd.Flags().GetString("compression")
	assert.Equal(t, "gzip", compression)
}

func TestRootCommand_HasSubcommands(t *testing.T) {
	// Test that root command has expected subcommands
	commands := rootCmd.Commands()

	commandNames := make([]string, 0)
	for _, cmd := range commands {
		commandNames = append(commandNames, cmd.Name())
	}

	assert.Contains(t, commandNames, "backup")
	assert.Contains(t, commandNames, "restore")
	assert.Contains(t, commandNames, "verify")
	assert.Contains(t, commandNames, "list")
	assert.Contains(t, commandNames, "version")
}

func TestBackupCommand_RequiredFlagsRegistered(t *testing.T) {
	cmd := backupCmd

	// Check required flags exist
	assert.NotNil(t, cmd.Flags().Lookup("source"))
	assert.NotNil(t, cmd.Flags().Lookup("dest"))
	assert.NotNil(t, cmd.Flags().Lookup("public-key"))
}

func TestRestoreCommand_RequiredFlagsRegistered(t *testing.T) {
	cmd := restoreCmd

	assert.NotNil(t, cmd.Flags().Lookup("file"))
	assert.NotNil(t, cmd.Flags().Lookup("dest"))
	assert.NotNil(t, cmd.Flags().Lookup("private-key"))
}

func TestVerifyCommand_RequiredFlagsRegistered(t *testing.T) {
	cmd := verifyCmd

	assert.NotNil(t, cmd.Flags().Lookup("file"))
	assert.NotNil(t, cmd.Flags().Lookup("private-key"))
	assert.NotNil(t, cmd.Flags().Lookup("quick"))
}

func TestListCommand_RequiredFlagsRegistered(t *testing.T) {
	cmd := listCmd

	assert.NotNil(t, cmd.Flags().Lookup("dest"))
	assert.NotNil(t, cmd.Flags().Lookup("pattern"))
}
