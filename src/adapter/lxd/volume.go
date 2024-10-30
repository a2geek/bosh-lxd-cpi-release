package lxd

import (
	"fmt"
	"io"

	client "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
)

func (a *lxdApiAdapter) DeleteStoragePoolVolume(pool, name string) error {
	return a.client.DeleteStoragePoolVolume(pool, "custom", name)
}

func (a *lxdApiAdapter) GetStoragePoolVolume(pool, name string) (string, error) {
	_, etag, err := a.client.GetStoragePoolVolume(pool, "custom", name)
	return etag, err
}

func (a *lxdApiAdapter) ResizeStoragePoolVolume(pool, name string, newSize int) error {
	volume, etag, err := a.client.GetStoragePoolVolume(pool, "custom", name)
	if err != nil {
		return err
	}

	writable := volume.Writable()
	writable.Config["size"] = fmt.Sprintf("%dMiB", newSize)

	return a.client.UpdateStoragePoolVolume(pool, "custom", name, writable, etag)
}

func (a *lxdApiAdapter) CreateStoragePoolVolume(pool, name string, size int) error {
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

func (a *lxdApiAdapter) CreateStoragePoolVolumeFromISO(pool, diskName string, backupFile io.Reader) error {
	return wait(a.client.CreateStoragePoolVolumeFromISO(pool, client.StoragePoolVolumeBackupArgs{
		Name:       diskName,
		BackupFile: backupFile,
	}))
}

func (a *lxdApiAdapter) UpdateStoragePoolVolumeDescription(pool, diskName, description string) error {
	volume, etag, err := a.client.GetStoragePoolVolume(pool, "custom", diskName)
	if err != nil {
		return err
	}

	volume.Description = description

	return a.client.UpdateStoragePoolVolume(pool, "custom", diskName, volume.Writable(), etag)
}
func (a *lxdApiAdapter) GetStoragePoolVolumeUsage(pool string) (map[string]int, error) {
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
