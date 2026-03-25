package file

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ZipFiles writes a zip archive containing the specified files to w.
// Only the base name of each file is used in the archive (not the full path).
// Files are compressed using the Deflate method.
//
// Example - Write to file:
//
//	f, err := os.Create("archive.zip")
//	if err != nil {
//	    return err
//	}
//	defer f.Close()
//
//	err = file.ZipFiles(f, []string{"/path/to/file1.txt", "/path/to/file2.txt"})
//
// Example - Write to memory buffer:
//
//	var buf bytes.Buffer
//	err := file.ZipFiles(&buf, []string{"/path/to/file.txt"})
//	if err != nil {
//	    return err
//	}
//	data := buf.Bytes()
//
// Example - Write to HTTP response:
//
//	func handleDownload(w http.ResponseWriter, r *http.Request) {
//	    w.Header().Set("Content-Type", "application/zip")
//	    w.Header().Set("Content-Disposition", "attachment; filename=archive.zip")
//	    err := file.ZipFiles(w, filePaths)
//	    // ...
//	}
func ZipFiles(w io.Writer, filePaths []string) error {
	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	for _, filePath := range filePaths {
		if err := addFileToZip(zipWriter, filePath); err != nil {
			return err
		}
	}

	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close zip writer: %w", err)
	}

	return nil
}

// addFileToZip adds a single file to the zip archive
func addFileToZip(zipWriter *zip.Writer, filePath string) error {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	// Get file info for the header
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info for %s: %w", filePath, err)
	}

	// Create zip header
	header, err := zip.FileInfoHeader(fileInfo)
	if err != nil {
		return fmt.Errorf("failed to create zip header for %s: %w", filePath, err)
	}

	// Use only the base name in the zip (not the full path)
	header.Name = filepath.Base(filePath)
	header.Method = zip.Deflate

	// Create writer for this file
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create zip entry for %s: %w", filePath, err)
	}

	// Copy file content to zip
	_, err = io.Copy(writer, file)
	if err != nil {
		return fmt.Errorf("failed to write file %s to zip: %w", filePath, err)
	}

	return nil
}
