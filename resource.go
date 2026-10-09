package vtf

// ResourceTag identifies the type of a resource stored in a v7.3+ texture
type ResourceTag string

const (
	// ResourceTagLowResImage is the low resolution thumbnail image
	ResourceTagLowResImage ResourceTag = "\x01\x00\x00"
	// ResourceTagImage is the high resolution image data
	ResourceTagImage ResourceTag = "\x30\x00\x00"
	// ResourceTagSheet is particle sprite sheet data
	ResourceTagSheet ResourceTag = "\x10\x00\x00"
	// ResourceTagCRC is a CRC of the source image. Stored without a data chunk
	ResourceTagCRC ResourceTag = "CRC"
	// ResourceTagLOD is texture level of detail clamping settings. Stored without a data chunk
	ResourceTagLOD ResourceTag = "LOD"
	// ResourceTagTextureSettingsEx is extended texture flags. Stored without a data chunk
	ResourceTagTextureSettingsEx ResourceTag = "TSO"
	// ResourceTagKeyValues is arbitrary KeyValues text data
	ResourceTagKeyValues ResourceTag = "KVD"
)

const (
	// ResourceFlagNoDataChunk is set when a resource stores its 4 byte value in the
	// resource directory, rather than in a data chunk elsewhere in the file
	ResourceFlagNoDataChunk = 0x02

	// maxResources is the maximum number of entries in a resource directory
	maxResources = 32
	// resourceDirectoryOffset is where the resource directory starts in a v7.3+ header
	resourceDirectoryOffset = 80
	// resourceEntrySize is the encoded size of a resource directory entry
	resourceEntrySize = 8
)

// Resource is an entry in the resource directory of a v7.3+ texture
type Resource struct {
	// Tag identifies the resource type
	Tag ResourceTag
	// Flags are the resource flags, e.g. ResourceFlagNoDataChunk
	Flags uint8
	// Data is the resource contents. Resources with ResourceFlagNoDataChunk hold their
	// 4 byte value here. Image resources are nil; use the image accessors on Vtf instead.
	Data []byte
}

// resourceEntry is a resource directory entry as stored on disk
type resourceEntry struct {
	tag   ResourceTag
	flags uint8
	// offset of the resource data, or its value when flags has ResourceFlagNoDataChunk
	offset uint32
}

func (entry resourceEntry) hasDataChunk() bool {
	return entry.flags&ResourceFlagNoDataChunk == 0
}
