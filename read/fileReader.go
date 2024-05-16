package read

import (
	"io"
	"os"
)

type fileReader struct {
	fileName string
}

// NewFileReader returns new file reader
func NewFileReader(fileName string) *fileReader {
	return &fileReader{fileName: fileName}
}

// Read reads from file
func (fr *fileReader) Read() (string, error) {
	f, err := os.OpenFile(fr.fileName, os.O_RDONLY, 0)
	defer f.Close()
	if err != nil {
		return "", err
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
