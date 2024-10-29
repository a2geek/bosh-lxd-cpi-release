package adapter

import "io"

type ApiAdapter interface {
	FindExistingImage(description string) (string, error)
	CreateAndUploadImage(meta ImageMetadata) error
	DeleteImage(alias string) error

	CreateInstance(meta InstanceMetadata) error
	DeleteInstance(name string) error
	GetInstance(name string) (string, error)
	UpdateInstanceDescription(name, newDescription string) error
	SetInstanceAction(instanceName string, action Action) error
	IsInstanceStopped(name string) (bool, error)

	// TODO StoragePoolVolumeMeta?
	DeleteStoragePoolVolume(pool, name string) error
	GetStoragePoolVolume(pool, name string) (string, error)
	ResizeStoragePoolVolume(pool, name string, newSize int) error
	CreateStoragePoolVolumeFromISO(pool, diskName string, backupFile io.Reader) error
	CreateStoragePoolVolume(pool, name string, size int) error
	UpdateStoragePoolVolumeDescription(poolName, diskName, description string) error

	// TODO StoragePoolVolumeSnapshotMeta?
	DeleteStoragePoolVolumeSnapshot(pool, volumeName, snapshotName string) error
	CreateStoragePoolVolumeSnapshot(pool, volumeName, snapshotName, description string) error

	AttachDevice(instanceName, deviceName string, device map[string]string) error
	DetachDevice(instanceName, deviceName string) error
	GetDevices(instanceName string) (map[string]map[string]string, error)
}

type ImageMetadata struct {
	Alias          string
	Description    string
	OsDistro       string
	ImagePath      string
	Architecture   string
	CreateDate     int64
	RootDeviceName string
	DiskInMB       int
	TarFile        io.Reader
}

type InstanceMetadata struct {
	Name          string
	StemcellAlias string
	InstanceType  string
	Project       string
	Devices       map[string]map[string]string
	Profiles      []string
	Config        map[string]string
}

type Action string

const (
	StartAction   Action = "start"
	StopAction    Action = "stop"
	RestartAction Action = "restart"
)
