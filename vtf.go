package vtf

// Vtf: Exported vtf format
// Contains a Header, resources (v7.3+), low res thumbnail & high-res mipmaps
type Vtf struct {
	header                  Header
	resources               []byte
	lowResolutionImageData  []uint8
	highResolutionImageData [][][][][]byte //[]mipmap[]frame[]face[]slice
}

// GetHeader: Get vtf Header
func (vtf *Vtf) GetHeader() Header {
	return vtf.header
}

// GetLowResImageData: Get raw data of low-resolution thumbnail
func (vtf *Vtf) GetLowResImageData() []uint8 {
	return vtf.lowResolutionImageData
}

// GetHighResImageData: Get all data for all mipmaps
func (vtf *Vtf) GetHighResImageData() [][][][][]byte {
	return vtf.highResolutionImageData
}

// GetMipmapsForFrame: Get all mipmap sizes for a single frame
func (vtf *Vtf) GetMipmapsForFrame(frame int) [][]byte {
	ret := make([][]byte, vtf.header.MipmapCount)

	for idx, mipmap := range vtf.highResolutionImageData {
		ret[idx] = mipmap[frame][0][0]
	}

	return ret
}

// GetHighestResolutionImageForFrame: Get the best possible resolution
// for a single frame in the vtf
func (vtf *Vtf) GetHighestResolutionImageForFrame(frame int) []byte {
	// @TODO This currently only supports single face, single Z Slice images
	return vtf.highResolutionImageData[vtf.header.MipmapCount-1][frame][0][0]
}
