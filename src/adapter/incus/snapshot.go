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
	err := wait(a.client.CreateStoragePoolVolumeSnapshot(pool, "custom", volumeName, post))
	if err != nil {
		return err
	}

	snapshot, etag, err := a.client.GetStoragePoolVolumeSnapshot(pool, "custom", volumeName, snapshotName)
	if err != nil {
		return err
	}

	// Update the snapshot with the description
	snapshot.Description = description
	return a.client.UpdateStoragePoolVolumeSnapshot(pool, "custom", volumeName, snapshotName, snapshot.Writable(), etag)
}
