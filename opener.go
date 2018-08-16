package vtf

import (
	"os"
	"io"
)

func ReadFromFile(filepath string) (*Vtf, error) {
	file,err := os.Open(filepath)
	if err != nil {
		return nil,err
	}

	v,err := ReadFromStream(file)
	file.Close()
	return v,err
}

func ReadFromStream(stream io.Reader) (*Vtf, error){
	reader := &Reader{
		stream: stream,
	}

	return reader.Read()
}
