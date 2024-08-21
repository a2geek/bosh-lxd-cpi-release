package agentmgr

type Config struct {
	// FAT32 or CDROM
	SourceType string
	// FAT32 and CDROM
	Label        string
	MetadataPath string
	UserdataPath string
	// Agent file store location
	FileStorePath string
}
