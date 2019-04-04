package vtf

// Vtf: Exported vtf format
// Contains a Header, resources (v7.3+), low res thumbnail & high-res mipmaps
type Vtf struct {
	header                  Header
	resources               []byte
	lowResolutionImageData  []uint8
	highResolutionImageData [][][][][]byte //[]mipmap[]frame[]face[]slice
}

// Header returns vtf Header
func (vtf *Vtf) Header() Header {
	return vtf.header
}

// LowResImageData returns raw data of low-resolution thumbnail
func (vtf *Vtf) LowResImageData() []uint8 {
	return vtf.lowResolutionImageData
}

// HighResImageData returns all data for all mipmaps
func (vtf *Vtf) HighResImageData() [][][][][]byte {
	return vtf.highResolutionImageData
}

// MipmapsForFrame returns all mipmap sizes for a single frame
func (vtf *Vtf) MipmapsForFrame(frame int) [][]byte {
	ret := make([][]byte, vtf.header.MipmapCount)

	for idx, mipmap := range vtf.highResolutionImageData {
		ret[idx] = mipmap[frame][0][0]
	}

	return ret
}

// HighestResolutionImageForFrame returns the best possible resolution
// for a single frame in the vtf
func (vtf *Vtf) HighestResolutionImageForFrame(frame int) []byte {
	// @TODO This currently only supports single face, single Z Slice images
	return vtf.highResolutionImageData[vtf.header.MipmapCount-1][frame][0][0]
}
