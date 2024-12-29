package hnd

import (
	"errors"
	"fmt"
	"os"
)

// Reads the size of the specified file and returns it in bytes.
// If the file does not exist or an error occurs, it returns an error.
func GetFileSize(filePath string) (int64, error) {
	// Get file info using os.Stat
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, fmt.Errorf("file does not exist: %s", filePath)
		}
		return 0, fmt.Errorf("error retrieving file info: %w", err)
	}

	// Return the size of the file
	return fileInfo.Size(), nil
}