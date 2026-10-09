package vtf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/galaco/vtf/format"
	"github.com/galaco/vtf/internal"
)

const (
	vtfSignature = "VTF\x00"
	// headerStructSize is the encoded size of Header, which describes every version.
	// Older versions have smaller headers; fields they lack are left zeroed.
	headerStructSize = 72
	// commonHeaderSize is the size of the signature, version and header size, which every version starts with
	commonHeaderSize = 16
	minHeaderSize    = 64
	maxHeaderSize    = 1024
	// maxPreallocSize is the largest read that is allocated up front. Larger reads grow as
	// data arrives, so a small file claiming to be huge cannot exhaust memory.
	maxPreallocSize = 16 << 20
)

var (
	// ErrSignatureMismatch occurs when a stream does not start with the VTF magic signature
	ErrSignatureMismatch = errors.New("vtf: header signature does not match: VTF\\x00")
	// ErrMipmapSizeMismatch occurs when the stream ends before the image data its header describes
	ErrMipmapSizeMismatch = errors.New("vtf: image data exceeds file size")
	// ErrInvalidDimensions occurs when texture dimensions are invalid
	ErrInvalidDimensions = errors.New("vtf: invalid texture dimensions")
	// ErrInvalidMipmapCount occurs when mipmap count is unreasonable
	ErrInvalidMipmapCount = errors.New("vtf: invalid mipmap count")
	// ErrInvalidHeaderSize occurs when header size is invalid
	ErrInvalidHeaderSize = errors.New("vtf: invalid header size")
	// ErrUnsupportedVersion occurs when VTF version is not supported
	ErrUnsupportedVersion = errors.New("vtf: unsupported VTF version")
	// ErrUnsupportedFormat occurs when image data is stored in a format whose size cannot be computed
	ErrUnsupportedFormat = internal.ErrUnsupportedFormat
	// ErrInvalidResource occurs when the resource directory of a v7.3+ texture is invalid
	ErrInvalidResource = errors.New("vtf: invalid resource")
)

// Deprecated error names, kept for compatibility.
var (
	// Deprecated: use ErrSignatureMismatch.
	ErrorVtfSignatureMismatch = ErrSignatureMismatch
	// Deprecated: volume textures are supported, so this error is no longer returned.
	ErrorTextureDepthNotSupported = errors.New("vtf: only vtf textures with depth 1 are supported")
	// Deprecated: use ErrMipmapSizeMismatch.
	ErrorMipmapSizeMismatch = ErrMipmapSizeMismatch
	// Deprecated: use ErrInvalidDimensions.
	ErrorInvalidDimensions = ErrInvalidDimensions
	// Deprecated: use ErrInvalidMipmapCount.
	ErrorInvalidMipmapCount = ErrInvalidMipmapCount
	// Deprecated: use ErrInvalidHeaderSize.
	ErrorInvalidHeaderSize = ErrInvalidHeaderSize
	// Deprecated: use ErrUnsupportedVersion.
	ErrorUnsupportedVersion = ErrUnsupportedVersion
)

// Reader reads a texture from a stream. It reads only the bytes the texture
// occupies, leaving the stream positioned after it.
type Reader struct {
	stream io.Reader
	// offset is the number of bytes read from stream
	offset  int64
	header  *Header
	entries []resourceEntry
	vtf     *Vtf
	err     error
}

// NewReader returns a Reader that reads a texture from r
func NewReader(r io.Reader) *Reader {
	return &Reader{
		stream: r,
	}
}

// ReadHeader reads and validates the header of a texture, without reading any image data.
// Read may be called afterwards to read the rest of the texture.
func (reader *Reader) ReadHeader() (*Header, error) {
	if reader.header == nil {
		if reader.err == nil {
			reader.header, reader.entries, reader.err = reader.readHeader()
		}
		if reader.err != nil {
			return nil, reader.err
		}
	}

	header := *reader.header
	return &header, nil
}

// Read parses vtf image from stream into a usable structure
// Expect ErrMipmapSizeMismatch if the stream ends before the image data the header describes,
// normally due to a truncated file or tampered Header data.
func (reader *Reader) Read() (*Vtf, error) {
	if reader.vtf == nil && reader.err == nil {
		reader.vtf, reader.err = reader.read()
	}
	return reader.vtf, reader.err
}

func (reader *Reader) read() (*Vtf, error) {
	header, err := reader.ReadHeader()
	if err != nil {
		return nil, err
	}

	v := &Vtf{
		header: *header,
	}

	// Before 7.3 the thumbnail follows the header, and image data follows the thumbnail
	if len(reader.entries) == 0 {
		if v.lowResolutionImageData, err = reader.readLowResolutionMipmap(header); err != nil {
			return nil, err
		}
		if v.highResolutionImageData, err = reader.readMipmaps(header, -1); err != nil {
			return nil, err
		}
		return v, nil
	}

	if v.resources, v.lowResolutionImageData, v.highResolutionImageData, err = reader.readResources(header); err != nil {
		return nil, err
	}

	return v, nil
}

// readHeader reads and validates the header, including the resource directory of 7.3+ textures
func (reader *Reader) readHeader() (*Header, []resourceEntry, error) {
	// Signature, version and header size are common to every version
	buffer := make([]byte, commonHeaderSize, maxHeaderSize)
	if err := reader.readFull(buffer); err != nil {
		return nil, nil, fmt.Errorf("vtf: reading header: %w", err)
	}
	if string(buffer[:4]) != vtfSignature {
		return nil, nil, ErrSignatureMismatch
	}
	headerSize := binary.LittleEndian.Uint32(buffer[12:16])
	if headerSize < minHeaderSize || headerSize > maxHeaderSize {
		return nil, nil, fmt.Errorf("%w: %d bytes (expected %d-%d)", ErrInvalidHeaderSize, headerSize, minHeaderSize, maxHeaderSize)
	}
	buffer = buffer[:headerSize]
	if err := reader.readFull(buffer[commonHeaderSize:]); err != nil {
		return nil, nil, fmt.Errorf("vtf: reading header: %w", err)
	}

	header, err := parseHeader(buffer)
	if err != nil {
		return nil, nil, err
	}
	if err := reader.validateHeader(header); err != nil {
		return nil, nil, err
	}

	return header, parseResourceEntries(header, buffer), nil
}

// parseHeader decodes a complete header
func parseHeader(buffer []byte) (*Header, error) {
	// Only decode bytes that belong to the header; anything beyond it is image data
	headerBytes := make([]byte, headerStructSize)
	copy(headerBytes, buffer)

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
func (reader *Reader) validateHeader(header *Header) error {
	// Validate version - only support 7.0 to 7.5
	majorVersion := header.Version[0]
	minorVersion := header.Version[1]
	if majorVersion != 7 || minorVersion > 5 {
		return fmt.Errorf("%w: %d.%d (only 7.0-7.5 supported)", ErrUnsupportedVersion, majorVersion, minorVersion)
	}

	// Validate dimensions
	if header.Width == 0 || header.Height == 0 {
		return fmt.Errorf("%w: width=%d, height=%d (cannot be zero)", ErrInvalidDimensions, header.Width, header.Height)
	}

	// Prevent excessive memory allocation - Source Engine typically uses max 4096x4096
	const maxDimension = 16384 // Allow some headroom beyond typical max
	depth := textureDepth(header)
	if header.Width > maxDimension || header.Height > maxDimension || depth > maxDimension {
		return fmt.Errorf("%w: width=%d, height=%d, depth=%d (max %d)", ErrInvalidDimensions, header.Width, header.Height, depth, maxDimension)
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
		return fmt.Errorf("%w: count=%d, expected 1-%d for %dx%dx%d texture", ErrInvalidMipmapCount, header.MipmapCount, maxMipmaps, header.Width, header.Height, depth)
	}

	// Validate header size
	if header.HeaderSize < minHeaderSize || header.HeaderSize > maxHeaderSize {
		return fmt.Errorf("%w: %d bytes (expected %d-%d)", ErrInvalidHeaderSize, header.HeaderSize, minHeaderSize, maxHeaderSize)
	}

	// Validate frame count is reasonable
	if header.Frames == 0 || header.Frames > 1024 {
		return fmt.Errorf("%w: frame count %d is invalid (expected 1-1024)", ErrInvalidDimensions, header.Frames)
	}

	// Validate resource directory (v7.3+) fits in the header
	if header.NumResource > maxResources {
		return fmt.Errorf("%w: %d resources (max %d)", ErrInvalidResource, header.NumResource, maxResources)
	}
	if header.NumResource > 0 && resourceDirectoryOffset+header.NumResource*resourceEntrySize > header.HeaderSize {
		return fmt.Errorf("%w: %d bytes is too small for %d resources", ErrInvalidHeaderSize, header.HeaderSize, header.NumResource)
	}

	// Validate low-res dimensions
	if header.LowResImageWidth > 16 || header.LowResImageHeight > 16 {
		return fmt.Errorf("%w: low-res dimensions %dx%d exceed expected maximum 16x16", ErrInvalidDimensions, header.LowResImageWidth, header.LowResImageHeight)
	}

	return nil
}

// parseResourceEntries decodes the resource directory of 7.3+ images from a complete header
func parseResourceEntries(header *Header, buffer []byte) []resourceEntry {
	entries := make([]resourceEntry, header.NumResource)
	for i := range entries {
		entry := buffer[resourceDirectoryOffset+i*resourceEntrySize:]
		entries[i] = resourceEntry{
			tag:    ResourceTag(entry[0:3]),
			flags:  entry[3],
			offset: binary.LittleEndian.Uint32(entry[4:8]),
		}
	}

	return entries
}

// readResources reads every resource of a 7.3+ texture, including the images
func (reader *Reader) readResources(header *Header) ([]Resource, []uint8, [][][][][]uint8, error) {
	resources := make([]Resource, len(reader.entries))
	var lowResImage []uint8
	var highResImage [][][][][]uint8

	// The stream is read forwards, so visit data chunks in the order they are stored
	var chunks []int
	foundLowRes, foundImage := false, false
	for i, entry := range reader.entries {
		resources[i] = Resource{Tag: entry.tag, Flags: entry.flags}
		if !entry.hasDataChunk() {
			resources[i].Data = binary.LittleEndian.AppendUint32(nil, entry.offset)
			continue
		}
		switch {
		case entry.tag == ResourceTagLowResImage && foundLowRes, entry.tag == ResourceTagImage && foundImage:
			return nil, nil, nil, fmt.Errorf("%w: duplicate resource %q", ErrInvalidResource, entry.tag)
		case entry.tag == ResourceTagLowResImage:
			foundLowRes = true
		case entry.tag == ResourceTagImage:
			foundImage = true
		}
		chunks = append(chunks, i)
	}
	if !foundImage {
		return nil, nil, nil, fmt.Errorf("%w: no high resolution image resource", ErrInvalidResource)
	}
	sort.SliceStable(chunks, func(a, b int) bool {
		return reader.entries[chunks[a]].offset < reader.entries[chunks[b]].offset
	})

	for i, idx := range chunks {
		entry := reader.entries[idx]
		if err := reader.skipTo(int64(entry.offset)); err != nil {
			return nil, nil, nil, fmt.Errorf("resource %q: %w", entry.tag, err)
		}

		var err error
		switch entry.tag {
		case ResourceTagLowResImage:
			lowResImage, err = reader.readLowResolutionMipmap(header)
		case ResourceTagImage:
			// Image data has no stored size, but ends no later than the next resource
			limit := int64(-1)
			if i+1 < len(chunks) {
				limit = int64(reader.entries[chunks[i+1]].offset) - int64(entry.offset)
			}
			highResImage, err = reader.readMipmaps(header, limit)
		default:
			resources[idx].Data, err = reader.readDataChunk()
			if err != nil {
				err = fmt.Errorf("resource %q: %w", entry.tag, err)
			}
		}
		if err != nil {
			return nil, nil, nil, err
		}
	}

	if lowResImage == nil {
		lowResImage = []uint8{}
	}

	return resources, lowResImage, highResImage, nil
}

// readDataChunk reads a size prefixed resource data chunk
func (reader *Reader) readDataChunk() ([]byte, error) {
	size := make([]byte, 4)
	if err := reader.readFull(size); err != nil {
		return nil, imageReadError(err)
	}

	return reader.readN(int64(binary.LittleEndian.Uint32(size)))
}

// readLowResolutionMipmap reads the low resolution texture information
// This is normally what you see previewed in Hammer texture browser.
// The largest axis should always be 16 wide/tall. The smallest can be any value,
// but is padded out to divisible by 4 for Dxt1 compressionn reasons
func (reader *Reader) readLowResolutionMipmap(header *Header) ([]uint8, error) {
	bufferSize, err := lowResImageSize(header)
	if err != nil {
		return nil, err
	}

	data, err := reader.readN(int64(bufferSize))
	if err != nil {
		return nil, fmt.Errorf("low resolution image: %w", err)
	}

	return data, nil
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
		return 0, fmt.Errorf("vtf: low resolution image: %w: %s", err, lowResFormat)
	}

	return size, nil
}

// readMipmaps reads all mipmaps. limit is the most bytes the image may occupy, or -1 if unknown.
// Returned format is a bit odd, but is just a set of flat arrays containing arrays:
// mipmap[frame[face[slice[RGBA]]]
// Mipmaps are indexed smallest first, as they are stored on disk.
func (reader *Reader) readMipmaps(header *Header, limit int64) ([][][][][]uint8, error) {
	buffer, numFaces, err := reader.readImageData(header, limit)
	if err != nil {
		return nil, fmt.Errorf("high resolution image: %w", err)
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

// readImageData reads the high resolution image data, returning it and the number of faces it holds
func (reader *Reader) readImageData(header *Header, limit int64) ([]byte, int, error) {
	if header.Flags&FlagEnvironmentMap == 0 {
		size, err := imageDataSize(header, 1)
		if err != nil {
			return nil, 0, err
		}
		data, err := reader.readLimited(size, limit)
		return data, 1, err
	}

	size6, err := imageDataSize(header, 6)
	if err != nil {
		return nil, 0, err
	}

	// Cubemaps in 7.1-7.4 may also store a spheremap as a 7th face, which can only
	// be detected by the size of the image data.
	if minorVersion := header.Version[1]; minorVersion < 1 || minorVersion >= 5 {
		data, err := reader.readLimited(size6, limit)
		return data, 6, err
	}
	size7, err := imageDataSize(header, 7)
	if err != nil {
		return nil, 0, err
	}
	if limit >= 0 {
		if limit >= size7 {
			data, err := reader.readN(size7)
			return data, 7, err
		}
		data, err := reader.readLimited(size6, limit)
		return data, 6, err
	}

	// The image is the end of the texture, so read the spheremap if the stream holds one
	data, err := reader.readN(size7)
	if errors.Is(err, ErrMipmapSizeMismatch) && int64(len(data)) >= size6 {
		return data[:size6:size6], 6, nil
	}
	return data, 7, err
}

// readLimited reads size bytes, if they fit within limit. A limit of -1 is unlimited.
func (reader *Reader) readLimited(size, limit int64) ([]byte, error) {
	if limit >= 0 && size > limit {
		return nil, fmt.Errorf("%w: need %d bytes, have %d", ErrMipmapSizeMismatch, size, limit)
	}
	return reader.readN(size)
}

// textureDepth returns the number of Z slices in the largest mipmap
func textureDepth(header *Header) int {
	// Depth is only stored from 7.2
	if header.Depth == 0 {
		return 1
	}
	return int(header.Depth)
}

// imageDataSize returns the size of the high resolution image data for every mipmap, frame, face and slice
func imageDataSize(header *Header, numFaces int) (int64, error) {
	storedFormat := format.Format(header.HighResImageFormat)

	total := int64(0)
	for level := 0; level < int(header.MipmapCount); level++ {
		width, height, depth := internal.MipmapDimensions(int(header.Width), int(header.Height), textureDepth(header), level)
		size, err := internal.ComputeSizeOfMipmapData(width, height, storedFormat)
		if err != nil {
			return 0, fmt.Errorf("vtf: %w: %s", err, storedFormat)
		}
		total += int64(size) * int64(depth) * int64(numFaces) * int64(header.Frames)
	}

	return total, nil
}

// readFull fills p from the stream
func (reader *Reader) readFull(p []byte) error {
	_, err := reader.readAtMost(p)
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return err
}

// readAtMost fills as much of p as the stream holds
func (reader *Reader) readAtMost(p []byte) (int, error) {
	n, err := io.ReadFull(reader.stream, p)
	reader.offset += int64(n)
	return n, err
}

// readN reads exactly n bytes. If the stream ends first, the bytes read are returned
// along with ErrMipmapSizeMismatch.
func (reader *Reader) readN(n int64) ([]byte, error) {
	if n <= maxPreallocSize {
		data := make([]byte, n)
		read, err := reader.readAtMost(data)
		if err != nil {
			return data[:read:read], imageReadError(err)
		}
		return data, nil
	}

	buffer := bytes.NewBuffer(make([]byte, 0, maxPreallocSize))
	copied, err := io.CopyN(buffer, reader.stream, n)
	reader.offset += copied
	data := buffer.Bytes()
	if err != nil {
		return data, imageReadError(err)
	}
	return data[:n:n], nil
}

// skipTo discards data up to offset
func (reader *Reader) skipTo(offset int64) error {
	if offset < reader.offset {
		return fmt.Errorf("%w: data at offset %d overlaps previous data ending at %d", ErrInvalidResource, offset, reader.offset)
	}
	skipped, err := io.CopyN(io.Discard, reader.stream, offset-reader.offset)
	reader.offset += skipped
	if err != nil {
		return imageReadError(err)
	}
	return nil
}

// imageReadError reports the stream ending early as ErrMipmapSizeMismatch
func imageReadError(err error) error {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return ErrMipmapSizeMismatch
	}
	return err
}
