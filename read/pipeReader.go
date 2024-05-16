package read

import (
	"bufio"
	"io"
	"os"
)

type pipeReader struct {
}

// NewPipeReader returns new pipe reader
func NewPipeReader() *pipeReader {
	return &pipeReader{}
}

// Read reads from pipe
func (pr *pipeReader) Read() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	b, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
