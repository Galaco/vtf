package vtf


type Vtf struct {
	header header
	resources []byte
	lowResolutionImageData []uint8
	highResolutionImageData [][][][][]byte //[]mipmap[]frame[]face[]slice
}

func (vtf *Vtf) GetHeader() header {
	return vtf.header
}

func (vtf *Vtf) GetLowResImageData() []uint8 {
	return vtf.lowResolutionImageData
}

func (vtf *Vtf) GetHighResImageData() [][][][][]byte {
	return vtf.highResolutionImageData
}

// Get all mipmap sizes for a single frame
func (vtf *Vtf) GetMipmapsForFrame(frame int) [][]byte {
	ret := make([][]byte, vtf.header.MipmapCount)

	for idx,mipmap := range vtf.highResolutionImageData {
		ret[idx] = mipmap[frame][0][0]
	}

	return ret
}

// Get the best possible resolution for a single frame in the vtf
func (vtf* Vtf) GetHighestResolutionImageForFrame(frame int) []byte {
	// @TODO This currently only supports single face, single Z Slice images
	return vtf.highResolutionImageData[vtf.header.MipmapCount - 1][frame][0][0]
}