package vtf

import (
	"io"
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/galaco/vtf/colourformat"
)

const headerSize = 96

type Reader struct {
	stream io.Reader
}

// Reads the vtf image from stream into a usable structure
func (reader *Reader) Read() (*Vtf, error) {
	buf := bytes.Buffer{}
	_,err := buf.ReadFrom(reader.stream)
	if err != nil {
		return nil,err
	}

	header,err := reader.parseHeader(buf.Bytes())
	if err != nil {
		return nil,err
	}

	resourceData,err := reader.parseOtherResourceData(header, buf.Bytes())
	if err != nil {
		return nil,err
	}
	lowResImage,err := reader.parseLowResImageData(header, buf.Bytes())
	if err != nil {
		return nil,err
	}

	lowResImageCompressedSize := colourformat.GetImageSizeInBytes(
		colourformat.ColorFormat(header.LowResImageFormat),
		int(header.LowResImageWidth),
		int(header.LowResImageHeight))
	highResImage,err := reader.parseImageData(header, buf.Bytes()[header.HeaderSize + uint32(lowResImageCompressedSize):])
	if err != nil {
		return nil,err
	}

	return &Vtf{
		header: *header,
		resources: resourceData,
		lowResolutionImageData: lowResImage,
		highResolutionImageData: highResImage,
	},nil
}

// Parse vtf header.
func (reader *Reader) parseHeader(buffer []byte) (*header,error) {

	// We know header is 96 bytes
	// Note it *may* not be someday...
	headerBytes := make([]byte, headerSize)

	// Read bytes equal to header size
	byteReader := bytes.NewReader(buffer)
	sectionReader := io.NewSectionReader(byteReader, 0, int64(len(headerBytes)))
	_, err := sectionReader.Read(headerBytes)
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

// Returns resource data for 7.3+ images
func (reader *Reader) parseOtherResourceData(header *header, buffer []byte) ([]byte, error) 	{
	// validate header version
	if (header.Version[0]*10 + header.Version[1] < 73) || header.NumResource == 0 {
		header.Depth = 0
		header.NumResource = 0
		return []byte{},nil
	}

	return []byte{},nil
}

func (reader *Reader) parseLowResImageData(header *header, buffer []byte) ([]uint8,error) {
	padWidth := int(header.LowResImageWidth)
	padHeight := int(header.LowResImageHeight)
	if header.LowResImageWidth % 4 != 0 {
		padWidth += 4 - (int(header.LowResImageWidth) % 4)
	} else if header.LowResImageHeight % 4 != 0 {
		padHeight += 4 - (int(header.LowResImageHeight) % 4)
	}
	bufferSize := (padWidth * padHeight) / 2 //DXT1 texture compression manages 50% compression
	imgBuffer := make([]byte, bufferSize)
	byteReader := bytes.NewReader(buffer[96:96+bufferSize])
	sectionReader := io.NewSectionReader(byteReader, 0, int64(bufferSize))
	_, err := sectionReader.Read(imgBuffer)
	if err != nil {
		return nil, err
	}

	return imgBuffer, nil
}

// Parse all image data here
func (reader *Reader) parseImageData(header *header, buffer []byte) ([][][][][]byte,error) {
	if header.Depth > 1 {
		return [][][][][]byte{}, errors.New("only vtf textures with depth 1 are supported")
	}
	depth := header.Depth

	if depth == 0 {
		depth = 1
	}

	numZSlice := uint16(1)
	bufferOffset := 0

	width := int(header.Width)
	height := int(header.Height)
	format := colourformat.ColorFormat(header.HighResImageFormat)

	// Force mipmap count

	header.MipmapCount = 1
	bufferOffset = (len(buffer) - 1) - colourformat.GetImageSizeInBytes(format, width, height)

	// Iterate mipmap; smallest to largest
	mipMaps := make([][][][][]byte, header.MipmapCount)
	for mipmapIdx := uint8(0); mipmapIdx < header.MipmapCount; mipmapIdx++ {
		// Frame by frame; first to last
		frames := make([][][][]byte, header.Frames)
		for frameIdx := uint16(0); frameIdx < header.Frames; frameIdx++ {
			faces := make([][][]byte, 1)
			// Face by face; first to last
			// @TODO is this correct to use depth? How to know how many faces there are
			for faceIdx := uint16(0); faceIdx < depth; faceIdx++ {
				zSlices := make([][]byte, 1)
				// Z Slice by Z Slice; first to last
				// @TODO wtf is a z slice, and how do we know how many there are
				for sliceIdx := uint16(0); sliceIdx < numZSlice; sliceIdx++ {
					dataSize := colourformat.GetImageSizeInBytes(format, width, height)
					img := buffer[bufferOffset:bufferOffset+dataSize]

					bufferOffset += dataSize
					zSlices[sliceIdx] = img
				}
				faces[faceIdx] = zSlices
			}
			frames[frameIdx] = faces
		}
		mipMaps[mipmapIdx] = frames
	}

	return mipMaps,nil
}
