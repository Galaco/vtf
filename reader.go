package vtf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/galaco/vtf/format"
	"github.com/galaco/vtf/internal"
)

const (
	vtfSignature = "VTF\x00"
	// headerStructSize is the encoded size of Header, which describes every version.
	// Older versions have smaller headers; fields they lack are left zeroed.
	headerStructSize = 72
)

var (
	// ErrorVtfSignatureMismatch occurs when a stream does not start with the VTF magic signature
	ErrorVtfSignatureMismatch = errors.New("header signature does not match: VTF\x00")
	// ErrorTextureDepthNotSupported occurs when attempting to parse a stream with depth > 1
	ErrorTextureDepthNotSupported = errors.New("only vtf textures with depth 1 are supported")
	// ErrorMipmapSizeMismatch occurs when the file is too small to contain the image data its header describes
	ErrorMipmapSizeMismatch = errors.New("image data exceeds file size")
	// ErrorInvalidDimensions occurs when texture dimensions are invalid
	ErrorInvalidDimensions = errors.New("invalid texture dimensions")
	// ErrorInvalidMipmapCount occurs when mipmap count is unreasonable
	ErrorInvalidMipmapCount = errors.New("invalid mipmap count")
	// ErrorInvalidHeaderSize occurs when header size is invalid
	ErrorInvalidHeaderSize = errors.New("invalid header size")
	// ErrorUnsupportedVersion occurs when VTF version is not supported
	ErrorUnsupportedVersion = errors.New("unsupported VTF version")
	// ErrorUnsupportedFormat occurs when image data is stored in a format whose size cannot be computed
	ErrorUnsupportedFormat = internal.ErrUnsupportedFormat
)

// Reader reads from a vtf stream
type Reader struct {
	stream io.Reader
}

// ReadHeader reads the header of a texture only.
func (reader *Reader) ReadHeader() (*Header, error) {
	data, err := io.ReadAll(reader.stream)
	if err != nil {
		return nil, err
	}

	// Header
	return reader.parseHeader(data)
}

// Read parses vtf image from stream into a usable structure
// The only error to expect would be if mipmap data size overflows the total file size; normally
// due to tampered Header data.
func (reader *Reader) Read() (*Vtf, error) {
	data, err := io.ReadAll(reader.stream)
	if err != nil {
		return nil, err
	}

	// Header
	header, err := reader.parseHeader(data)
	if err != nil {
		return nil, err
	}

	// Validate header to prevent DoS attacks via malicious files
	if err := reader.validateHeader(header, len(data)); err != nil {
		return nil, err
	}

	// Resources - in vtf 7.3+ only
	resourceData, err := reader.parseOtherResourceData(header, data)
	if err != nil {
		return nil, err
	}

	// Low resolution preview texture
	lowResImage, err := reader.readLowResolutionMipmap(header, data)
	if err != nil {
		return nil, err
	}

	// Mipmaps immediately follow the low resolution image
	highResImage, err := reader.readMipmaps(header, data[int(header.HeaderSize)+len(lowResImage):])
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
	// Signature, version and header size are common to every version
	if len(buffer) < 16 {
		return nil, fmt.Errorf("reading header: %w", io.ErrUnexpectedEOF)
	}
	if string(buffer[:4]) != vtfSignature {
		return nil, ErrorVtfSignatureMismatch
	}

	// Only decode bytes that belong to the header; anything beyond it is image data
	headerSize := int(binary.LittleEndian.Uint32(buffer[12:16]))
	if headerSize > headerStructSize {
		headerSize = headerStructSize
	}
	if headerSize > len(buffer) {
		return nil, fmt.Errorf("reading header: %w", io.ErrUnexpectedEOF)
	}
	headerBytes := make([]byte, headerStructSize)
	copy(headerBytes, buffer[:headerSize])

	header := Header{}
	if err := binary.Read(bytes.NewReader(headerBytes), binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	// Clear fields that are not part of this version's header
	minorVersion := header.Version[1]
	if minorVersion < 2 {
		header.Header72 = Header72{}
	}
	if minorVersion < 3 {
		header.Header73 = Header73{}
	}

	return &header, nil
}

// validateHeader performs security validation on header fields to prevent DoS attacks
func (reader *Reader) validateHeader(header *Header, fileSize int) error {
	// Validate version - only support 7.0 to 7.5
	majorVersion := header.Version[0]
	minorVersion := header.Version[1]
	if majorVersion != 7 || minorVersion > 5 {
		return fmt.Errorf("%w: %d.%d (only 7.0-7.5 supported)", ErrorUnsupportedVersion, majorVersion, minorVersion)
	}

	// Validate dimensions
	if header.Width == 0 || header.Height == 0 {
		return fmt.Errorf("%w: width=%d, height=%d (cannot be zero)", ErrorInvalidDimensions, header.Width, header.Height)
	}

	// Prevent excessive memory allocation - Source Engine typically uses max 4096x4096
	const maxDimension = 16384 // Allow some headroom beyond typical max
	if header.Width > maxDimension || header.Height > maxDimension {
		return fmt.Errorf("%w: width=%d, height=%d (max %d)", ErrorInvalidDimensions, header.Width, header.Height, maxDimension)
	}

	// Validate mipmap count - should not exceed log2(max(width, height)) + 1
	maxDim := header.Width
	if header.Height > maxDim {
		maxDim = header.Height
	}
	var maxMipmaps uint8 = 0
	for d := maxDim; d > 0; d >>= 1 {
		maxMipmaps++
	}
	if header.MipmapCount == 0 || header.MipmapCount > maxMipmaps {
		return fmt.Errorf("%w: count=%d, expected 1-%d for %dx%d texture", ErrorInvalidMipmapCount, header.MipmapCount, maxMipmaps, header.Width, header.Height)
	}

	// Validate header size
	if header.HeaderSize < 64 || header.HeaderSize > 1024 {
		return fmt.Errorf("%w: %d bytes (expected 64-1024)", ErrorInvalidHeaderSize, header.HeaderSize)
	}

	// Validate header size doesn't exceed file size
	if int(header.HeaderSize) > fileSize {
		return fmt.Errorf("%w: header size %d exceeds file size %d", ErrorInvalidHeaderSize, header.HeaderSize, fileSize)
	}

	// Validate frame count is reasonable
	if header.Frames == 0 || header.Frames > 1024 {
		return fmt.Errorf("%w: frame count %d is invalid (expected 1-1024)", ErrorInvalidDimensions, header.Frames)
	}

	// Validate depth (if present in v7.2+)
	if header.Depth > 1 {
		return ErrorTextureDepthNotSupported
	}

	// Validate low-res dimensions
	if header.LowResImageWidth > 16 || header.LowResImageHeight > 16 {
		return fmt.Errorf("%w: low-res dimensions %dx%d exceed expected maximum 16x16", ErrorInvalidDimensions, header.LowResImageWidth, header.LowResImageHeight)
	}

	return nil
}

// parseOtherResourceData reads resource data for 7.3+ images
func (reader *Reader) parseOtherResourceData(header *Header, buffer []byte) ([]byte, error) {
	return []byte{}, nil
}

// readLowResolutionMipmap reads the low resolution texture information
// This is normally what you see previewed in Hammer texture browser.
// The largest axis should always be 16 wide/tall. The smallest can be any value,
// but is padded out to divisible by 4 for Dxt1 compressionn reasons
func (reader *Reader) readLowResolutionMipmap(header *Header, buffer []byte) ([]uint8, error) {
	// Textures may be stored without a thumbnail
	lowResFormat := format.Format(header.LowResImageFormat)
	if lowResFormat == format.None || header.LowResImageWidth == 0 || header.LowResImageHeight == 0 {
		return []uint8{}, nil
	}

	bufferSize, err := internal.ComputeSizeOfMipmapData(
		int(header.LowResImageWidth),
		int(header.LowResImageHeight),
		lowResFormat)
	if err != nil {
		return nil, fmt.Errorf("low resolution image: %w: %s", err, lowResFormat)
	}

	start := int(header.HeaderSize)
	if bufferSize > len(buffer)-start {
		return nil, fmt.Errorf("low resolution image: %w", ErrorMipmapSizeMismatch)
	}

	return buffer[start : start+bufferSize : start+bufferSize], nil
}

// readMipmaps reads all mipmaps from the start of buffer
// Returned format is a bit odd, but is just a set of flat arrays containing arrays:
// mipmap[frame[face[slice[RGBA]]]
// Mipmaps are indexed smallest first, as they are stored on disk.
func (reader *Reader) readMipmaps(header *Header, buffer []byte) ([][][][][]uint8, error) {
	storedFormat := format.Format(header.HighResImageFormat)
	numMipmaps := int(header.MipmapCount)
	numFrames := int(header.Frames)
	// Faces and Z slices are not yet supported
	numFaces := 1
	numZSlices := 1

	// Compute the size of every mipmap up front, so a truncated file is rejected before allocating
	mipmapSizes := make([]int, numMipmaps)
	totalSize := int64(0)
	for level := 0; level < numMipmaps; level++ {
		width, height, _ := internal.MipmapDimensions(int(header.Width), int(header.Height), 1, level)
		size, err := internal.ComputeSizeOfMipmapData(width, height, storedFormat)
		if err != nil {
			return nil, fmt.Errorf("high resolution image: %w: %s", err, storedFormat)
		}
		mipmapSizes[level] = size
		totalSize += int64(size) * int64(numFrames*numFaces*numZSlices)
	}
	if totalSize > int64(len(buffer)) {
		return nil, fmt.Errorf("high resolution image: %w: need %d bytes, have %d", ErrorMipmapSizeMismatch, totalSize, len(buffer))
	}

	// Data is stored smallest mipmap first, then frame by frame, face by face, slice by slice
	mipMaps := make([][][][][]uint8, numMipmaps)
	offset := 0
	for level := numMipmaps - 1; level >= 0; level-- {
		size := mipmapSizes[level]
		frames := make([][][][]uint8, numFrames)
		for frameIdx := range frames {
			faces := make([][][]uint8, numFaces)
			for faceIdx := range faces {
				zSlices := make([][]uint8, numZSlices)
				for sliceIdx := range zSlices {
					zSlices[sliceIdx] = buffer[offset : offset+size : offset+size]
					offset += size
				}
				faces[faceIdx] = zSlices
			}
			frames[frameIdx] = faces
		}
		// Level 0 is the largest mipmap, but mipmaps are indexed smallest first
		mipMaps[numMipmaps-1-level] = frames
	}

	return mipMaps, nil
}
