package archive

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CreateTar creates a tar archive from the source directory and writes to the provided writer
func CreateTar(sourcePath string, w io.Writer) error {
	// Resolve to absolute path
	absPath, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Verify source exists
	sourceInfo, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("failed to stat source path: %w", err)
	}

	tw := tar.NewWriter(w)
	defer tw.Close()

	// Base directory for relative paths
	baseDir := filepath.Dir(absPath)
	baseName := filepath.Base(absPath)

	// Walk the directory tree
	return filepath.Walk(absPath, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk error at %s: %w", file, err)
		}

		// Create tar header from file info
		header, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header for %s: %w", file, err)
		}

		// Set relative path in archive
		// If source is a file, use just the filename
		// If source is a directory, preserve structure relative to parent
		if sourceInfo.IsDir() {
			relPath, err := filepath.Rel(baseDir, file)
			if err != nil {
				return fmt.Errorf("failed to get relative path: %w", err)
			}
			header.Name = relPath
		} else {
			// Single file backup - use just the filename
			if file == absPath {
				header.Name = baseName
			} else {
				// This shouldn't happen for single file, but handle gracefully
				header.Name = filepath.Base(file)
			}
		}

		// Handle symlinks
		if fi.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(file)
			if err != nil {
				return fmt.Errorf("failed to read symlink %s: %w", file, err)
			}
			header.Linkname = linkTarget
		}

		// Write header
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header for %s: %w", file, err)
		}

		// Write file data (if it's a regular file)
		if fi.Mode().IsRegular() {
			f, err := os.Open(file)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", file, err)
			}
			defer f.Close()

			if _, err := io.Copy(tw, f); err != nil {
				return fmt.Errorf("failed to write file data for %s: %w", file, err)
			}
		}

		return nil
	})
}

// ExtractTar extracts a tar archive from the reader to the destination directory
func ExtractTar(r io.Reader, destPath string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Resolve to absolute path for security
	absDestPath, err := filepath.Abs(destPath)
	if err != nil {
		return fmt.Errorf("failed to resolve destination path: %w", err)
	}

	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Sanitize the file path to prevent path traversal attacks
		if err := validateTarPath(header.Name); err != nil {
			return fmt.Errorf("invalid tar path %s: %w", header.Name, err)
		}

		// Construct full destination path
		targetPath := filepath.Join(absDestPath, header.Name)

		// Security check: ensure the target path is within destination
		if !strings.HasPrefix(targetPath, absDestPath+string(os.PathSeparator)) &&
			targetPath != absDestPath {
			return fmt.Errorf("invalid tar path %s: path traversal detected", header.Name)
		}

		// Extract based on type
		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}

		case tar.TypeReg:
			// Create parent directory if needed
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", targetPath, err)
			}

			// Create file
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}
			outFile.Close()

		case tar.TypeSymlink:
			// Create parent directory if needed
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", targetPath, err)
			}

			// Create symlink
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				return fmt.Errorf("failed to create symlink %s: %w", targetPath, err)
			}

		default:
			// Skip unsupported types (block devices, character devices, etc.)
			fmt.Fprintf(os.Stderr, "Warning: skipping unsupported file type %c for %s\n", header.Typeflag, header.Name)
		}
	}

	return nil
}

// validateTarPath checks for path traversal attempts in tar archive paths
func validateTarPath(path string) error {
	// Check for absolute paths
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths not allowed in tar archive")
	}

	// Check for path traversal sequences
	cleanPath := filepath.Clean(path)
	if strings.HasPrefix(cleanPath, "..") || strings.Contains(cleanPath, "/../") {
		return fmt.Errorf("path traversal detected")
	}

	return nil
}
