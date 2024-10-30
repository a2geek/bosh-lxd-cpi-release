package lxd

import (
	"github.com/canonical/lxd/shared/api"
)

func (a *lxdApiAdapter) DeleteStoragePoolVolumeSnapshot(pool, volumeName, snapshotName string) error {
	return wait(a.client.DeleteStoragePoolVolumeSnapshot(pool, "custom", volumeName, snapshotName))
}

func (a *lxdApiAdapter) CreateStoragePoolVolumeSnapshot(pool, volumeName, snapshotName, description string) error {
	post := api.StorageVolumeSnapshotsPost{
		Name:        snapshotName,
		Description: description,
	}
	return wait(a.client.CreateStoragePoolVolumeSnapshot(pool, "custom", volumeName, post))
}
