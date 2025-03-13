package lxd

import (
	"fmt"
	"io"

	client "github.com/canonical/lxd/client"
	lxd "github.com/canonical/lxd/client"
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

func (a *lxdApiAdapter) CreateStoragePoolVolume(target, pool, name string, size int) error {
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

	c := a.client
	if target != "" {
		c = c.UseTarget(target)
	}
	return c.CreateStoragePoolVolume(pool, storageVolumeRequest)
}

func (a *lxdApiAdapter) CreateStoragePoolVolumeFromISO(target, pool, diskName string, backupFile io.Reader) error {
	c := a.client
	if target != "" {
		c = c.UseTarget(target)
	}
	return wait(c.CreateStoragePoolVolumeFromISO(pool, client.StoragePoolVolumeBackupArgs{
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

func (a *lxdApiAdapter) ColocateStoragePoolVolumeWithInstance(instanceName, pool, diskName string) error {
	instanceLoc, err := a.GetInstanceLocation(instanceName)
	if err != nil {
		return err
	}

	volume, _, err := a.client.GetStoragePoolVolume(pool, "custom", diskName)
	if err != nil {
		return err
	}

	if instanceLoc == volume.Location {
		return nil
	}

	srcServer := a.client.UseTarget(volume.Location)
	dstServer := a.client.UseTarget(instanceLoc)

	args := &lxd.StoragePoolVolumeCopyArgs{
		Name:       volume.Name,
		Mode:       "move",
		VolumeOnly: false,
	}

	// Manually move since MoveStoragePoolVolume() gives error:
	// "Moving storage volumes between remotes is not implemented"
	err = wait(dstServer.CopyStoragePoolVolume(pool, srcServer, pool, *volume, args))
	if err != nil {
		return err
	}

	return srcServer.DeleteStoragePoolVolume(pool, "custom", diskName)
}
