package adapter

import "io"

type ApiAdapter interface {
	// TODO FIX NAMES - USE LXD/Incus TERMINOLOGY
	FindExistingStemcell(description string) (string, error)
	CreateAndUploadStemcell(meta ImageMetadata) error
	DeleteStemcell(alias string) error

	// TODO FIX NAMES - USE LXD/Incus TERMINOLOGY
	CreateVM(meta InstanceMetadata) error
	DeleteInstance(name string) error
	GetInstance(name string) (string, error)
	UpdateInstanceDescription(name, newDescription string) error
	SetInstanceAction(instanceName, action string) error
	IsInstanceStopped(name string) (bool, error)

	// TODO StoragePoolVolumeMeta?
	// TODO Remote volType - we only use custom
	DeleteStoragePoolVolume(pool, volType, name string) error
	GetStoragePoolVolume(pool, volType, name string) (string, error)
	ResizeStoragePoolVolume(pool, volType, name string, newSize int) error
	CreateStoragePoolVolumeFromISO(pool, diskName string, backupFile io.Reader) error
	CreateStoragePoolVolume(pool, name string, size int) error
	UpdateStoragePoolVolumeDescription(poolName, diskName, description string) error

	// TODO StoragePoolVolumeSnapshotMeta?
	DeleteStoragePoolVolumeSnapshot(pool, volType, volumeName, snapshotName string) error
	CreateStoragePoolVolumeSnapshot(pool, volType, volumeName, snapshotName, description string) error

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
