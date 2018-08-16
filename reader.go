package vtf

import (
	"io"
	"bytes"
	"encoding/binary"
	"errors"
)

const headerSize = 96

type Reader struct {
	stream io.Reader
}

// Reads the vtf image from stream into a usable structure
func (reader *Reader) Read() (*Vtf, error) {
	header,err := reader.parseHeader()
	if err != nil {
		return nil,err
	}

	return &Vtf{
		header: *header,
	},nil
}

// Parse vtf header.
func (reader *Reader) parseHeader() (*header,error) {
	buf := bytes.Buffer{}
	_,err := buf.ReadFrom(reader.stream)
	if err != nil {
		return nil,err
	}

	// We know header is 96 bytes
	// Note it *may* not be someday...
	headerBytes := make([]byte, headerSize)

	// Read bytes equal to header size
	byteReader := bytes.NewReader(buf.Bytes())
	sectionReader := io.NewSectionReader(byteReader, 0, int64(len(headerBytes)))
	_, err = sectionReader.Read(headerBytes)
	if err != nil {
		return nil, err
	}

	// Set header data to read bytes
	header := header{}
	err = binary.Read(bytes.NewBuffer(headerBytes[:]), binary.LittleEndian, &header)
	if err != nil {
		return nil,err
	}
	if string(header.Signature[:4]) != "VTF\x00" {
		return nil,errors.New("header signature does not match: VTF\x00")
	}

	return &header,nil
}