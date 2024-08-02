package agentmgr

type Config struct {
	// ConfigDrive or CDROM
	SourceType string
	// ConfigDrive
	Label        string
	MetadataPath string
	UserdataPath string
	// CDROM
	Filename string
}
