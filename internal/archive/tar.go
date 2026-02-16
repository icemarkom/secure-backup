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

package archive

import (
	"archive/tar"
	"fmt"
	"io"
	"io/fs"
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

	// Verify source exists (Lstat to avoid following if source itself is a symlink)
	sourceInfo, err := os.Lstat(absPath)
	if err != nil {
		return fmt.Errorf("failed to stat source path: %w", err)
	}

	tw := tar.NewWriter(w)
	defer tw.Close()

	// Base directory for relative paths
	baseDir := filepath.Dir(absPath)
	baseName := filepath.Base(absPath)

	// Walk the directory tree (WalkDir uses Lstat — does not follow symlinks)
	return filepath.WalkDir(absPath, func(file string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk error at %s: %w", file, err)
		}

		// Compute relative path for archive
		var relPath string
		if sourceInfo.IsDir() {
			relPath, err = filepath.Rel(baseDir, file)
			if err != nil {
				return fmt.Errorf("failed to get relative path: %w", err)
			}
		} else {
			if file == absPath {
				relPath = baseName
			} else {
				relPath = filepath.Base(file)
			}
		}

		// Handle symlinks before calling d.Info() (which would follow them)
		if d.Type()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(file)
			if err != nil {
				return fmt.Errorf("failed to read symlink %s: %w", file, err)
			}
			header := &tar.Header{
				Typeflag: tar.TypeSymlink,
				Name:     relPath,
				Linkname: linkTarget,
			}
			if err := tw.WriteHeader(header); err != nil {
				return fmt.Errorf("failed to write tar header for symlink %s: %w", file, err)
			}
			return nil
		}

		// Get file info for non-symlink entries
		fi, err := d.Info()
		if err != nil {
			return fmt.Errorf("failed to get file info for %s: %w", file, err)
		}

		// Create tar header from file info
		header, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header for %s: %w", file, err)
		}
		header.Name = relPath

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
