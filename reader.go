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
	//
	// Deprecated: volume textures are supported, so this error is no longer returned.
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
	// ErrorInvalidResource occurs when the resource directory of a v7.3+ texture is invalid
	ErrorInvalidResource = errors.New("invalid resource")
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
	entries, err := reader.parseResourceEntries(header, data)
	if err != nil {
		return nil, err
	}
	resources, err := reader.readResources(entries, data)
	if err != nil {
		return nil, err
	}

	layout, err := reader.locateImages(header, entries, data)
	if err != nil {
		return nil, err
	}

	// Low resolution preview texture
	lowResImage, err := reader.readLowResolutionMipmap(header, data, layout.lowResOffset)
	if err != nil {
		return nil, err
	}

	// Mipmaps
	highResImage, err := reader.readMipmaps(header, data[layout.imageOffset:layout.imageEnd])
	if err != nil {
		return nil, err
	}

	return &Vtf{
		header:                  *header,
		resources:               resources,
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
	depth := textureDepth(header)
	if header.Width > maxDimension || header.Height > maxDimension || depth > maxDimension {
		return fmt.Errorf("%w: width=%d, height=%d, depth=%d (max %d)", ErrorInvalidDimensions, header.Width, header.Height, depth, maxDimension)
	}

	// Validate mipmap count - should not exceed log2(max(width, height, depth)) + 1
	maxDim := int(header.Width)
	if int(header.Height) > maxDim {
		maxDim = int(header.Height)
	}
	if depth > maxDim {
		maxDim = depth
	}
	var maxMipmaps uint8 = 0
	for d := maxDim; d > 0; d >>= 1 {
		maxMipmaps++
	}
	if header.MipmapCount == 0 || header.MipmapCount > maxMipmaps {
		return fmt.Errorf("%w: count=%d, expected 1-%d for %dx%dx%d texture", ErrorInvalidMipmapCount, header.MipmapCount, maxMipmaps, header.Width, header.Height, depth)
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

	// Validate resource directory (v7.3+) fits in the header
	if header.NumResource > maxResources {
		return fmt.Errorf("%w: %d resources (max %d)", ErrorInvalidResource, header.NumResource, maxResources)
	}
	if header.NumResource > 0 && resourceDirectoryOffset+header.NumResource*resourceEntrySize > header.HeaderSize {
		return fmt.Errorf("%w: %d bytes is too small for %d resources", ErrorInvalidHeaderSize, header.HeaderSize, header.NumResource)
	}

	// Validate low-res dimensions
	if header.LowResImageWidth > 16 || header.LowResImageHeight > 16 {
		return fmt.Errorf("%w: low-res dimensions %dx%d exceed expected maximum 16x16", ErrorInvalidDimensions, header.LowResImageWidth, header.LowResImageHeight)
	}

	return nil
}

// parseResourceEntries reads the resource directory of 7.3+ images
func (reader *Reader) parseResourceEntries(header *Header, buffer []byte) ([]resourceEntry, error) {
	entries := make([]resourceEntry, header.NumResource)
	for i := range entries {
		entry := buffer[resourceDirectoryOffset+i*resourceEntrySize:]
		entries[i] = resourceEntry{
			tag:    ResourceTag(entry[0:3]),
			flags:  entry[3],
			offset: binary.LittleEndian.Uint32(entry[4:8]),
		}
	}

	return entries, nil
}

// readResources reads the contents of every resource, except image data
func (reader *Reader) readResources(entries []resourceEntry, buffer []byte) ([]Resource, error) {
	resources := make([]Resource, 0, len(entries))
	for _, entry := range entries {
		resource := Resource{Tag: entry.tag, Flags: entry.flags}
		switch {
		case !entry.hasDataChunk():
			resource.Data = binary.LittleEndian.AppendUint32(nil, entry.offset)
		case entry.tag == ResourceTagLowResImage || entry.tag == ResourceTagImage:
			// Image data has no size prefix; it is read by readLowResolutionMipmap and readMipmaps
		default:
			// Data chunks are prefixed with their size
			start := int64(entry.offset) + 4
			if start > int64(len(buffer)) {
				return nil, fmt.Errorf("resource %q: %w", entry.tag, ErrorMipmapSizeMismatch)
			}
			end := start + int64(binary.LittleEndian.Uint32(buffer[start-4:start]))
			if end > int64(len(buffer)) {
				return nil, fmt.Errorf("resource %q: %w", entry.tag, ErrorMipmapSizeMismatch)
			}
			resource.Data = buffer[start:end:end]
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// imageLayout locates image data within a texture
type imageLayout struct {
	// lowResOffset is the offset of the low resolution image, or -1 if there is none
	lowResOffset int
	// imageOffset and imageEnd bound the region that holds the high resolution image
	imageOffset, imageEnd int
}

// locateImages finds the low and high resolution images
func (reader *Reader) locateImages(header *Header, entries []resourceEntry, buffer []byte) (imageLayout, error) {
	// Before 7.3 the thumbnail follows the header, and image data follows the thumbnail
	if len(entries) == 0 {
		lowResSize, err := lowResImageSize(header)
		if err != nil {
			return imageLayout{}, err
		}
		imageOffset := int(header.HeaderSize) + lowResSize
		if imageOffset > len(buffer) {
			return imageLayout{}, fmt.Errorf("low resolution image: %w", ErrorMipmapSizeMismatch)
		}
		return imageLayout{lowResOffset: int(header.HeaderSize), imageOffset: imageOffset, imageEnd: len(buffer)}, nil
	}

	layout := imageLayout{lowResOffset: -1, imageOffset: -1, imageEnd: len(buffer)}
	for _, entry := range entries {
		if int64(entry.offset) > int64(len(buffer)) && entry.hasDataChunk() {
			return imageLayout{}, fmt.Errorf("resource %q: %w", entry.tag, ErrorMipmapSizeMismatch)
		}
		switch {
		case entry.tag == ResourceTagLowResImage && layout.lowResOffset < 0:
			layout.lowResOffset = int(entry.offset)
		case entry.tag == ResourceTagImage && layout.imageOffset < 0:
			layout.imageOffset = int(entry.offset)
		}
	}
	if layout.imageOffset < 0 {
		return imageLayout{}, fmt.Errorf("%w: no high resolution image resource", ErrorInvalidResource)
	}

	// Image data has no stored size, but ends no later than the next resource
	for _, entry := range entries {
		if offset := int(entry.offset); entry.hasDataChunk() && offset > layout.imageOffset && offset < layout.imageEnd {
			layout.imageEnd = offset
		}
	}

	return layout, nil
}

// readLowResolutionMipmap reads the low resolution texture information
// This is normally what you see previewed in Hammer texture browser.
// The largest axis should always be 16 wide/tall. The smallest can be any value,
// but is padded out to divisible by 4 for Dxt1 compressionn reasons
func (reader *Reader) readLowResolutionMipmap(header *Header, buffer []byte, offset int) ([]uint8, error) {
	bufferSize, err := lowResImageSize(header)
	if err != nil {
		return nil, err
	}
	if offset < 0 || bufferSize == 0 {
		return []uint8{}, nil
	}

	if bufferSize > len(buffer)-offset {
		return nil, fmt.Errorf("low resolution image: %w", ErrorMipmapSizeMismatch)
	}

	return buffer[offset : offset+bufferSize : offset+bufferSize], nil
}

// lowResImageSize returns the size of the low resolution image, which is 0 if there is none
func lowResImageSize(header *Header) (int, error) {
	// Textures may be stored without a thumbnail
	lowResFormat := format.Format(header.LowResImageFormat)
	if lowResFormat == format.None || header.LowResImageWidth == 0 || header.LowResImageHeight == 0 {
		return 0, nil
	}

	size, err := internal.ComputeSizeOfMipmapData(
		int(header.LowResImageWidth),
		int(header.LowResImageHeight),
		lowResFormat)
	if err != nil {
		return 0, fmt.Errorf("low resolution image: %w: %s", err, lowResFormat)
	}

	return size, nil
}

// readMipmaps reads all mipmaps from the start of buffer
// Returned format is a bit odd, but is just a set of flat arrays containing arrays:
// mipmap[frame[face[slice[RGBA]]]
// Mipmaps are indexed smallest first, as they are stored on disk.
func (reader *Reader) readMipmaps(header *Header, buffer []byte) ([][][][][]uint8, error) {
	numFaces, err := faceCount(header, int64(len(buffer)))
	if err != nil {
		return nil, err
	}

	// Check the whole image is present before allocating
	totalSize, err := imageDataSize(header, numFaces)
	if err != nil {
		return nil, err
	}
	if totalSize > int64(len(buffer)) {
		return nil, fmt.Errorf("high resolution image: %w: need %d bytes, have %d", ErrorMipmapSizeMismatch, totalSize, len(buffer))
	}

	storedFormat := format.Format(header.HighResImageFormat)
	numMipmaps := int(header.MipmapCount)
	numFrames := int(header.Frames)

	// Data is stored smallest mipmap first, then frame by frame, face by face, slice by slice
	mipMaps := make([][][][][]uint8, numMipmaps)
	offset := 0
	for level := numMipmaps - 1; level >= 0; level-- {
		width, height, numZSlices := internal.MipmapDimensions(int(header.Width), int(header.Height), textureDepth(header), level)
		size, _ := internal.ComputeSizeOfMipmapData(width, height, storedFormat)
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

// textureDepth returns the number of Z slices in the largest mipmap
func textureDepth(header *Header) int {
	// Depth is only stored from 7.2
	if header.Depth == 0 {
		return 1
	}
	return int(header.Depth)
}

// faceCount returns the number of faces stored per frame. available is the
// size of the region holding the high resolution image.
func faceCount(header *Header, available int64) (int, error) {
	if header.Flags&FlagEnvironmentMap == 0 {
		return 1, nil
	}

	// Cubemaps in 7.1-7.4 may also store a spheremap as a 7th face, which can only
	// be detected by the size of the image data.
	if minorVersion := header.Version[1]; minorVersion >= 1 && minorVersion < 5 {
		size, err := imageDataSize(header, 7)
		if err != nil {
			return 0, err
		}
		if available >= size {
			return 7, nil
		}
	}

	return 6, nil
}

// imageDataSize returns the size of the high resolution image data for every mipmap, frame, face and slice
func imageDataSize(header *Header, numFaces int) (int64, error) {
	storedFormat := format.Format(header.HighResImageFormat)

	total := int64(0)
	for level := 0; level < int(header.MipmapCount); level++ {
		width, height, depth := internal.MipmapDimensions(int(header.Width), int(header.Height), textureDepth(header), level)
		size, err := internal.ComputeSizeOfMipmapData(width, height, storedFormat)
		if err != nil {
			return 0, fmt.Errorf("high resolution image: %w: %s", err, storedFormat)
		}
		total += int64(size) * int64(depth) * int64(numFaces) * int64(header.Frames)
	}

	return total, nil
}
