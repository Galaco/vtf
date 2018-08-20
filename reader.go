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

	lowResImageCompressedSize := (int(header.LowResImageWidth) *
		int(header.LowResImageHeight)) *
		int(colourformat.BytesPerPixel(colourformat.ColorFormat(header.LowResImageFormat)))
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
		return nil,nil
	}

	return nil,nil
}

func (reader *Reader) parseLowResImageData(header *header, buffer []byte) ([]uint8,error) {
	bufferSize := (int(header.LowResImageWidth) * int(header.LowResImageHeight)) / 2 //DXT1 texture compression manages 50% compression
	imgBuffer := make([]byte, bufferSize)
	byteReader := bytes.NewReader(buffer[96:96+bufferSize])
	sectionReader := io.NewSectionReader(byteReader, 0, int64(bufferSize))
	_, err := sectionReader.Read(imgBuffer)
	if err != nil {
		return nil, err
	}

	return colourformat.FromDXT1(int(header.LowResImageWidth), int(header.LowResImageHeight), imgBuffer)
}

// Parse all image data here
func (reader *Reader) parseImageData(header *header, buffer []byte) ([][][][][]byte,error) {
	bufferOffset := 0
	format := colourformat.ColorFormat(header.HighResImageFormat)


	dataSize := colourformat.GetImageSizeInBytes(format, int(header.Width), int(header.Height))
	imgOffset := (len(buffer)) - dataSize
	img,err := getRawImageDataFromBuffer(buffer[imgOffset:], format, int(header.Width), int(header.Height))
	if err != nil {
		return nil,err
	}
	ret := make([][][][][]byte, 1)
	ret[0] = make([][][][]byte, 1)
	ret[0][0] = make([][][]byte, 1)
	ret[0][0][0] = make([][]byte, 1)
	ret[0][0][0][0] = img
	header.Frames = 0
	return ret,nil

	// IGNORE MIPMAPS
	width := int(header.LowResImageWidth)
	height := int(header.LowResImageHeight)

	for {
		if width == 1 || height == 1 {
			break
		}
		width /= 2
		height /= 2
	}

	mipMaps := make([][][][][]byte, header.MipmapCount)
	// Iterate mipmap; smallest to largest
	for mipmapIdx := uint8(0); mipmapIdx < header.MipmapCount; mipmapIdx++ {
		width *= 2
		height *= 2
		frames := make([][][][]byte, header.Frames)

		// Frame by frame; first to last
		for frameIdx := uint16(0); frameIdx < header.Frames; frameIdx++ {
			faces := make([][][]byte, 1)
			// Face by face; first to last
			// @TODO is this correct to use depth? How to know how many faces there are
			for faceIdx := uint16(0); faceIdx < header.Depth; faceIdx++ {
				zSlices := make([][]byte, 1)
				// Z Slice by Z Slice; first to last
				// @TODO wtf is a z slice, and how do we know how many there are
				for sliceIdx := uint16(0); sliceIdx < 1; sliceIdx++ {
					dataSize := colourformat.GetImageSizeInBytes(format, width, height)
					img,_ := getRawImageDataFromBuffer(buffer[bufferOffset:bufferOffset+dataSize], format, width, height)

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

func getRawImageDataFromBuffer(buffer []byte, format colourformat.ColorFormat, width int, height int) ([]byte,error) {
	switch format {
	case colourformat.BGRX8888:
		return colourformat.FromBGRX8888(width, height, buffer)
	case colourformat.RGB888:
		return buffer, nil
	case colourformat.Dxt1:
		decompressed, err := colourformat.FromDXT1(width, height, buffer)
		if err != nil {
			return nil, err
		}
		return decompressed, nil
	case colourformat.Dxt5:
		decompressed, err := colourformat.FromDXT5(width, height, buffer)
		if err != nil {
			return nil, err
		}
		return decompressed, nil
	default:
		return nil, errors.New("unsupported image format: " + string(format))
	}
}