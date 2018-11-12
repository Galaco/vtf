package vtf

// Header: VTF Header format
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
	Signature [4]byte						//File signature char
	Version [2]uint32						//Version[0].version[1] e.g. 7.2 uint
	HeaderSize uint32						//Size of Header (16 byte aligned, currently 80bytes) uint
	Width uint16							//Width of largest mipmap (^2) ushort
	Height uint16							//Height of largest mipmap (^2) ushort
	Flags uint32							//VTF Flags uint
	Frames uint16							//Number of frames (if animated) default: 1 ushort
	FirstFrame uint16						//First frame in animation (0 based) ushort
	_ [4]byte
	Reflectivity [3]float32					//reflectivity vector float
	_ [4]byte
	BumpmapScale float32					//Bumpmap scale float
	HighResImageFormat uint32				//High resolution image format uint (probably 4?)
	MipmapCount uint8						//Number of mipmaps uchar
	LowResImageFormat uint32				//Low resolution image format (always DXT1 [=14]) uint
	LowResImageWidth uint8					//Low resolution image width uchar
	LowResImageHeight uint8					//Low resolution image height uchar
}

// Header72: v7.2+ includes these properties
type Header72 struct {
	Depth uint16							//Depth of the largest mipmap in pixels (^2) ushort
}

// Header73: v7.3+ includes these properties
type Header73 struct {
	_ [3]byte
	NumResource uint32						// Number of resources this vtf has
}