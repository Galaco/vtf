package vtf

import (
	"io"
	"os"
)

// ReadFromStream loads a vtf from standard
// io.Reader stream. Only the bytes the texture occupies are read.
func ReadFromStream(stream io.Reader) (*Vtf, error) {
	return NewReader(stream).Read()
}

// ReadFromFile is a wrapper for ReadFromStream wrapper to load directly from
// filesystem. Exists for convenience
func ReadFromFile(filepath string) (*Vtf, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	return ReadFromStream(file)
}
