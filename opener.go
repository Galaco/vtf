package vtf

import (
	"io"
	"os"
)

// ReadFromStream: Load vtf from standard
// io.Reader stream,
func ReadFromStream(stream io.Reader) (*Vtf, error) {
	reader := &Reader{
		stream: stream,
	}

	return reader.Read()
}

// ReadFromFile: ReadFromStream wrapper to load directly from
// filesystem. Exists for convenience
func ReadFromFile(filepath string) (*Vtf, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	v, err := ReadFromStream(file)
	return v, err
}
