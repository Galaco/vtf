package vtf

// Header for VTF Header format
// Contents includes information for all versions
// Its up to the implementee to decide what properties they need
// based on the version
type Header struct {
	HeaderCommon
	Header72
	Header73
}

// HeaderCommon: All VTF versions start with these properties
type HeaderCommon struct {
	//Signature is file signature char
	Signature [4]byte
	// Version - Version[0].version[1] e.g. 7.2 uint
	Version [2]uint32
	// HeaderSize is size of Header (16 byte aligned, currently 80bytes) uint
	HeaderSize uint32
	// Width of largest mipmap (^2) ushort
	Width uint16
	// Height of largest mipmap (^2) ushort
	Height uint16
	// Flags are VTF Flags uint
	Flags uint32
	// Frames in number of frames (if animated) default: 1 ushort
	Frames uint16
	// FirstFrame is first frame in animation (0 based) ushort
	FirstFrame uint16
	_          [4]byte
	// Reflectivity vector float
	Reflectivity [3]float32
	_            [4]byte
	// BumpmapScale is bumpmap scale float
	BumpmapScale float32
	// HighResImageFormat is high resolution image format uint (probably 4?)
	HighResImageFormat uint32
	// MipmapCount is number of mipmaps uchar
	MipmapCount uint8
	// LowResImageFormat is low resolution image format (always DXT1 [=14]) uint
	LowResImageFormat uint32
	// LowResImageWidth is low resolution image width uchar
	LowResImageWidth uint8
	// LowResImageHeight is low resolution image height uchar
	LowResImageHeight uint8
}

// Header72 is v7.2+ includes these properties
type Header72 struct {
	// Depth of the largest mipmap in pixels (^2) ushort
	Depth uint16
}

// Header73 is v7.3+ includes these properties
type Header73 struct {
	_ [3]byte
	// NumResource is number of resources this vtf has
	NumResource uint32
}
