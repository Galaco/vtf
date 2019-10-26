package vtf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/galaco/vtf/format"
	"github.com/galaco/vtf/internal"
	"io"
)

const (
	vtfSignature = "VTF\x00"
)

var (
	// ErrorVtfSignatureMismatch occurs when a stream does not start with the VTF magic signature
	ErrorVtfSignatureMismatch = errors.New("header signature does not match: VTF\x00")
	// ErrorTextureDepthNotSupported occurs when attempting to parse a stream with depth > 1
	ErrorTextureDepthNotSupported = errors.New("only vtf textures with depth 1 are supported")
	// ErrorMipmapSizeMismatch occurs when filesize does not match calculated mipmap size
	ErrorMipmapSizeMismatch = errors.New("expected data size is smaller than actual")
)

// Reader reads from a vtf stream
type Reader struct {
	stream io.Reader
}

// ReadHeader reads the header of a texture only.
func (reader *Reader) ReadHeader() (*Header, error) {
	buf := bytes.Buffer{}
	_, err := buf.ReadFrom(reader.stream)
	if err != nil {
		return nil, err
	}

	// Header
	return reader.parseHeader(buf.Bytes())
}

// Read parses vtf image from stream into a usable structure
// The only error to expect would be if mipmap data size overflows the total file size; normally
// due to tampered Header data.
func (reader *Reader) Read() (*Vtf, error) {
	buf := bytes.Buffer{}
	_, err := buf.ReadFrom(reader.stream)
	if err != nil {
		return nil, err
	}

	// Header
	header, err := reader.parseHeader(buf.Bytes())
	if err != nil {
		return nil, err
	}

	// Resources - in vtf 7.3+ only
	resourceData, err := reader.parseOtherResourceData(header, buf.Bytes())
	if err != nil {
		return nil, err
	}

	// Low resolution preview texture
	lowResImage, err := reader.readLowResolutionMipmap(header, buf.Bytes())
	if err != nil {
		return nil, err
	}

	// Mipmaps
	highResImage, err := reader.readMipmaps(header, buf.Bytes())
	if err != nil {
		return nil, err
	}

	return &Vtf{
		header:                  *header,
		resources:               resourceData,
		lowResolutionImageData:  lowResImage,
		highResolutionImageData: highResImage,
	}, nil
}

// parseHeader reads vtf Header.
func (reader *Reader) parseHeader(buffer []byte) (*Header, error) {
	// We know Header is 96 bytes maximum
	// Note it *may* not be someday...
	headerBytes := make([]byte, 96)

	// Read bytes equal to Header size
	byteReader := bytes.NewReader(buffer)
	sectionReader := io.NewSectionReader(byteReader, 0, int64(len(headerBytes)))
	_, err := sectionReader.Read(headerBytes)
	if err != nil {
		return nil, err
	}

	// Set Header data to read bytes
	header := Header{}
	err = binary.Read(bytes.NewBuffer(headerBytes[:]), binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}
	if string(header.Signature[:4]) != vtfSignature {
		return nil, ErrorVtfSignatureMismatch
	}

	return &header, nil
}

// parseOtherResourceData reads resource data for 7.3+ images
func (reader *Reader) parseOtherResourceData(header *Header, buffer []byte) ([]byte, error) {
	// validate Header version
	if (header.Version[0]*10+header.Version[1] < 73) || header.NumResource == 0 {
		header.Depth = 0
		header.NumResource = 0
		return []byte{}, nil
	}

	return []byte{}, nil
}

// readLowResolutionMipmap reads the low resolution texture information
// This is normally what you see previewed in Hammer texture browser.
// The largest axis should always be 16 wide/tall. The smallest can be any value,
// but is padded out to divisible by 4 for Dxt1 compressionn reasons
func (reader *Reader) readLowResolutionMipmap(header *Header, buffer []byte) ([]uint8, error) {
	bufferSize := internal.ComputeSizeOfMipmapData(
		int(header.LowResImageWidth),
		int(header.LowResImageHeight),
		format.Dxt1)

	imgBuffer := make([]byte, bufferSize)
	byteReader := bytes.NewReader(buffer[int(header.HeaderSize) : int(header.HeaderSize)+bufferSize])
	sectionReader := io.NewSectionReader(byteReader, 0, int64(bufferSize))
	_, err := sectionReader.Read(imgBuffer)
	if err != nil {
		return nil, err
	}

	return imgBuffer, nil
}

// readMipmaps reads all mipmaps
// Returned format is a bit odd, but is just a set of flat arrays containing arrays:
// mipmap[frame[face[slice[RGBA]]]
func (reader *Reader) readMipmaps(header *Header, buffer []byte) ([][][][][]uint8, error) {
	if header.Depth > 1 {
		return [][][][][]uint8{}, ErrorTextureDepthNotSupported
	}

	depth := header.Depth

	// Occurs in texture versions before 7.2
	if depth == 0 {
		depth = 1
	}

	// Only support 1 ZSlice. No known Source game can use > 1 zslices
	numZSlice := uint16(1)
	bufferEnd := len(buffer)

	storedFormat := format.Format(header.HighResImageFormat)
	mipmapSizes := internal.ComputeMipmapSizes(int(header.MipmapCount), int(header.Width), int(header.Height))

	// Iterate mipmap; smallest to largest
	mipMaps := make([][][][][]uint8, header.MipmapCount)
	for mipmapIdx := int8(header.MipmapCount - 1); mipmapIdx >= int8(0); mipmapIdx-- {
		// Frame by frame; first to last
		frames := make([][][][]uint8, header.Frames)
		for frameIdx := uint16(0); frameIdx < header.Frames; frameIdx++ {
			faces := make([][][]uint8, 1)
			// Face by face; first to last
			// @TODO is this correct to use depth? How to know how many faces there are
			for faceIdx := uint16(0); faceIdx < depth; faceIdx++ {
				zSlices := make([][]uint8, 1)
				// Z Slice by Z Slice; first to last
				// @TODO wtf is a z slice, and how do we know how many there are
				for sliceIdx := uint16(0); sliceIdx < numZSlice; sliceIdx++ {
					bufferSize := internal.ComputeSizeOfMipmapData(
						mipmapSizes[mipmapIdx][0],
						mipmapSizes[mipmapIdx][1],
						storedFormat)
					if len(buffer) < bufferEnd-bufferSize {
						return mipMaps, ErrorMipmapSizeMismatch
					}
					img := buffer[bufferEnd-bufferSize : bufferEnd]

					bufferEnd -= bufferSize
					zSlices[sliceIdx] = img
				}
				faces[faceIdx] = zSlices
			}
			frames[frameIdx] = faces
		}
		mipMaps[mipmapIdx] = frames

		// Ensure that we maintain aspect ratio when scaling up mipmaps
	}

	return mipMaps, nil
}
