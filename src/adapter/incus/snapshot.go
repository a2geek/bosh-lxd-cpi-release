package incus

import (
	"github.com/lxc/incus/v6/shared/api"
)

func (a *incusApiAdapter) DeleteStoragePoolVolumeSnapshot(pool, volumeName, snapshotName string) error {
	return wait(a.client.DeleteStoragePoolVolumeSnapshot(pool, "custom", volumeName, snapshotName))
}

func (a *incusApiAdapter) CreateStoragePoolVolumeSnapshot(pool, volumeName, snapshotName, description string) error {
	post := api.StorageVolumeSnapshotsPost{
		Name: snapshotName,
	}
	return wait(a.client.CreateStoragePoolVolumeSnapshot(pool, "custom", volumeName, post))
}
