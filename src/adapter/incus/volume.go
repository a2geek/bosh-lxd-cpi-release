package incus

import (
	"fmt"
	"io"

	client "github.com/lxc/incus/client"
	"github.com/lxc/incus/shared/api"
)

func (a *incusApiAdapter) DeleteStoragePoolVolume(pool, name string) error {
	return a.client.DeleteStoragePoolVolume(pool, "custom", name)
}

func (a *incusApiAdapter) GetStoragePoolVolume(pool, name string) (string, error) {
	_, etag, err := a.client.GetStoragePoolVolume(pool, "custom", name)
	return etag, err
}

func (a *incusApiAdapter) ResizeStoragePoolVolume(pool, name string, newSize int) error {
	volume, etag, err := a.client.GetStoragePoolVolume(pool, "custom", name)
	if err != nil {
		return err
	}

	writable := volume.Writable()
	writable.Config["size"] = fmt.Sprintf("%dMiB", newSize)

	return a.client.UpdateStoragePoolVolume(pool, "custom", name, writable, etag)
}

func (a *incusApiAdapter) CreateStoragePoolVolume(pool, name string, size int) error {
	storageVolumeRequest := api.StorageVolumesPost{
		Name:        name,
		Type:        "custom",
		ContentType: "block",
		StorageVolumePut: api.StorageVolumePut{
			Config: map[string]string{
				"size": fmt.Sprintf("%dMiB", size),
			},
		},
	}

	return a.client.CreateStoragePoolVolume(pool, storageVolumeRequest)
}

func (a *incusApiAdapter) CreateStoragePoolVolumeFromISO(pool, diskName string, backupFile io.Reader) error {
	return wait(a.client.CreateStoragePoolVolumeFromISO(pool, client.StoragePoolVolumeBackupArgs{
		Name:       diskName,
		BackupFile: backupFile,
	}))
}

func (a *incusApiAdapter) UpdateStoragePoolVolumeDescription(pool, diskName, description string) error {
	volume, etag, err := a.client.GetStoragePoolVolume(pool, "custom", diskName)
	if err != nil {
		return err
	}

	volume.Description = description

	return a.client.UpdateStoragePoolVolume(pool, "custom", diskName, volume.Writable(), etag)
}
func (a *incusApiAdapter) GetStoragePoolVolumeUsage(pool string) (map[string]int, error) {
	volumes, err := a.client.GetStoragePoolVolumes(pool)
	if err != nil {
		return nil, err
	}

	data := map[string]int{}
	for _, volume := range volumes {
		data[volume.Name] = len(volume.UsedBy)
	}
	return data, nil
}