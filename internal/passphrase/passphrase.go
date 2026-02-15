package passphrase

import (
	"fmt"
	"os"
	"strings"
)

// Get retrieves the passphrase from one of three sources in priority order:
// 1. Flag value (--passphrase)
// 2. Environment variable (SECURE_BACKUP_PASSPHRASE)
// 3. File path (--passphrase-file)
//
// Returns an error if multiple sources are provided (mutually exclusive).
// Returns empty string if no sources are provided (allowed for keys without passphrase).
// Prints a security warning to stderr if the flag value is used.
func Get(flagValue, envName, filePath string) (string, error) {
	// Count how many sources are provided
	sourcesProvided := 0
	sources := []string{}

	if flagValue != "" {
		sourcesProvided++
		sources = append(sources, "--passphrase flag")
	}

	envValue := ""
	if envName != "" {
		envValue = os.Getenv(envName)
		if envValue != "" {
			sourcesProvided++
			sources = append(sources, fmt.Sprintf("%s environment variable", envName))
		}
	}

	if filePath != "" {
		sourcesProvided++
		sources = append(sources, "--passphrase-file flag")
	}

	// Check mutual exclusivity
	if sourcesProvided > 1 {
		return "", fmt.Errorf("multiple passphrase sources provided (%s). Use only one method", strings.Join(sources, ", "))
	}

	// Priority 1: Flag value
	if flagValue != "" {
		// Print security warning to stderr
		fmt.Fprintln(os.Stderr, "WARNING: Passphrase on command line is insecure and visible in process lists. Use SECURE_BACKUP_PASSPHRASE environment variable or --passphrase-file instead.")
		return flagValue, nil
	}

	// Priority 2: Environment variable
	if envValue != "" {
		return envValue, nil
	}

	// Priority 3: File
	if filePath != "" {
		return readFromFile(filePath)
	}

	// No sources provided - return empty string (allowed for keys without passphrase)
	return "", nil
}

// readFromFile reads the passphrase from a file safely
func readFromFile(path string) (string, error) {
	// Check file exists and get permissions
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("passphrase file not found: %s", path)
		}
		return "", fmt.Errorf("failed to read passphrase file: %w", err)
	}

	// Check file permissions - warn if world-readable
	mode := fileInfo.Mode()
	if mode.Perm()&0004 != 0 {
		fmt.Fprintf(os.Stderr, "WARNING: Passphrase file %s is world-readable. Recommend: chmod 600 %s\n", path, path)
	}

	// Read file content
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read passphrase file: %w", err)
	}

	// Trim whitespace and newlines
	passphrase := strings.TrimSpace(string(data))

	if passphrase == "" {
		return "", fmt.Errorf("passphrase file is empty: %s", path)
	}

	return passphrase, nil
}
