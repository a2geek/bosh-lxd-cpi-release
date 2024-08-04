package agentmgr

type Config struct {
	// ConfigDrive or CDROM
	SourceType string
	// ConfigDrive and CDROM
	Label        string
	MetadataPath string
	UserdataPath string
}
