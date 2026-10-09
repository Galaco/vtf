package vtf

// Vtf: Exported vtf format
// Contains a Header, resources (v7.3+), low res thumbnail & high-res mipmaps
type Vtf struct {
	header                  Header
	resources               []Resource
	lowResolutionImageData  []uint8
	highResolutionImageData [][][][][]uint8 //[]mipmap[]frame[]face[]slice
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

// LowResImageData returns raw data of low-resolution thumbnail
func (vtf *Vtf) LowResImageData() []uint8 {
	return vtf.lowResolutionImageData
}

// HighResImageData returns all data for all mipmaps, indexed [mipmap][frame][face][slice].
// Mipmaps are ordered smallest first. Cubemaps have 6 faces (7 when an older
// texture includes a spheremap), and volume textures have one slice per depth layer.
func (vtf *Vtf) HighResImageData() [][][][][]uint8 {
	return vtf.highResolutionImageData
}

// Image returns raw data of the first frame of the highest resolution mipmap
func (vtf *Vtf) Image() []uint8 {
	return vtf.HighestResolutionImageForFrame(0)
}

// MipmapsForFrame returns all mipmap sizes for a single frame
// For cubemaps and volume textures, only the first face and slice is returned.
func (vtf *Vtf) MipmapsForFrame(frame int) [][]uint8 {
	ret := make([][]uint8, vtf.header.MipmapCount)

	for idx, mipmap := range vtf.highResolutionImageData {
		ret[idx] = mipmap[frame][0][0]
	}

	return ret
}

// HighestResolutionImageForFrame returns the best possible resolution
// for a single frame in the vtf
// For cubemaps and volume textures, only the first face and slice is returned.
func (vtf *Vtf) HighestResolutionImageForFrame(frame int) []byte {
	return vtf.highResolutionImageData[vtf.header.MipmapCount-1][frame][0][0]
}
