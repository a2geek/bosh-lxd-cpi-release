package adapter

import "io"

type ApiAdapter interface {
	FindExistingImage(description string) (string, error)
	CreateAndUploadImage(meta ImageMetadata) error
	DeleteImage(alias string) error
	GetStemcellInfo(alias string) (*ImageInfo, error)

	IsManagedNetwork(name string) (bool, error)

	CreateInstance(meta InstanceMetadata) error
	DeleteInstance(name string) error
	GetInstanceInfo(name string) (*InstanceInfo, error)
	UpdateInstanceDescription(name, newDescription string) error
	SetInstanceAction(instanceName string, action Action) error
	IsInstanceStopped(name string) (bool, error)

	ContainerFileRead(containerName, filePath string) (io.ReadCloser, error)
	ContainerFileWrite(containerName, filePath string, agentEnvContents []byte) error

	DeleteStoragePoolVolume(pool, name string) error
	GetStoragePoolVolume(pool, name string) (string, error)
	GetStoragePoolVolumeUsage(pool string) (map[string]int, error)
	ResizeStoragePoolVolume(pool, name string, newSize int) error
	CreateStoragePoolVolumeFromISO(target, pool, diskName string, backupFile io.Reader) error
	CreateStoragePoolVolume(target, pool, name string, contentType ContentType, size int) error
	UpdateStoragePoolVolumeDescription(pool, diskName, description string) error
	ColocateStoragePoolVolumeWithInstance(instanceName, pool, diskName string) error

	DeleteStoragePoolVolumeSnapshot(pool, volumeName, snapshotName string) error
	CreateStoragePoolVolumeSnapshot(pool, volumeName, snapshotName, description string) error

	AttachDevice(instanceName, deviceName string, device map[string]string) error
	DetachDeviceByName(instanceName, deviceName string) error
	DetachDeviceBySource(instanceName, sourceName string) error
	GetDevices(instanceName string) (map[string]map[string]string, error)

	Disconnect()
}

type InstanceType int

const (
	InstanceVM InstanceType = iota
	InstanceContainer
)

type ImageInfo struct {
	Description string
	Type        InstanceType
}

type ImageMetadata struct {
	Alias          string
	Description    string
	Type           InstanceType
	OsDistro       string
	ImagePath      string
	Architecture   string
	CreateDate     int64
	RootDeviceName string
	DiskInMB       int
	TarFile        io.Reader
}

type InstanceInfo struct {
	Location string
	Type     InstanceType
}

type InstanceMetadata struct {
	Name          string
	StemcellAlias string
	Type          InstanceType
	InstanceType  string
	Project       string
	Profiles      []string
	Target        string
	Devices       map[string]map[string]string
	Config        map[string]string
}

type Action string

const (
	StartAction   Action = "start"
	StopAction    Action = "stop"
	RestartAction Action = "restart"
)

type ContentType string

const (
	BlockContent      ContentType = "block"
	FilesystemContent ContentType = "filesystem"
)
