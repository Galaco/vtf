package vtf

import (
	"errors"
	"fmt"

	"github.com/galaco/vtf/v2/internal"
)

// ErrOutOfRange occurs when requesting a mipmap, frame, face or slice the texture does not have
var ErrOutOfRange = errors.New("vtf: image index out of range")

// Vtf is a decoded texture.
// Contains a Header, resources (v7.3+), low res thumbnail & high-res mipmaps.
// Image data is returned in its stored format, and shares memory with the texture.
type Vtf struct {
	header    Header
	resources []Resource
	lowRes    []byte
	// images is indexed [mipmap][frame][face][slice], with mipmap 0 the largest
	images [][][][][]byte
}

// Header returns vtf Header
func (vtf *Vtf) Header() Header {
	return vtf.header
}

// Resources returns the resource directory of a v7.3+ texture, in the order it is stored.
// Older versions have no resources.
func (vtf *Vtf) Resources() []Resource {
	return vtf.resources
}

// Resource returns the first resource with the given tag, if the texture has one
func (vtf *Vtf) Resource(tag ResourceTag) (Resource, bool) {
	for _, resource := range vtf.resources {
		if resource.Tag == tag {
			return resource, true
		}
	}
	return Resource{}, false
}

// LowResImageData returns raw data of the low resolution thumbnail, stored in
// Header().LowResImageFormat. It is empty if the texture has no thumbnail.
func (vtf *Vtf) LowResImageData() []byte {
	return vtf.lowRes
}

// Image returns raw data of the largest mipmap of the first frame, face and slice
func (vtf *Vtf) Image() []byte {
	return vtf.images[0][0][0][0]
}

// Mipmap returns raw data of a single image. Mipmap level 0 is the largest.
// Use MipmapDimensions for its size.
func (vtf *Vtf) Mipmap(level, frame, face, slice int) ([]byte, error) {
	if level < 0 || level >= vtf.MipmapCount() {
		return nil, fmt.Errorf("%w: mipmap %d of %d", ErrOutOfRange, level, vtf.MipmapCount())
	}
	if frame < 0 || frame >= vtf.FrameCount() {
		return nil, fmt.Errorf("%w: frame %d of %d", ErrOutOfRange, frame, vtf.FrameCount())
	}
	if face < 0 || face >= vtf.FaceCount() {
		return nil, fmt.Errorf("%w: face %d of %d", ErrOutOfRange, face, vtf.FaceCount())
	}
	slices := vtf.images[level][frame][face]
	if slice < 0 || slice >= len(slices) {
		return nil, fmt.Errorf("%w: slice %d of %d for mipmap %d", ErrOutOfRange, slice, len(slices), level)
	}

	return slices[slice], nil
}

// MipmapCount returns the number of mipmaps
func (vtf *Vtf) MipmapCount() int {
	return len(vtf.images)
}

// FrameCount returns the number of animation frames
func (vtf *Vtf) FrameCount() int {
	return len(vtf.images[0])
}

// FaceCount returns the number of faces per frame: 6 for cubemaps (7 when an older
// texture also stores a spheremap), otherwise 1
func (vtf *Vtf) FaceCount() int {
	return len(vtf.images[0][0])
}

// MipmapDimensions returns the width, height and depth (number of slices) of a mipmap level
func (vtf *Vtf) MipmapDimensions(level int) (width, height, depth int) {
	return internal.MipmapDimensions(int(vtf.header.Width), int(vtf.header.Height), textureDepth(&vtf.header), level)
}
