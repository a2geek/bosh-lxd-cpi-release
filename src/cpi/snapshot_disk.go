package cpi

import (
	"github.com/canonical/lxd/shared/api"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

func (c CPI) SnapshotDisk(diskCID apiv1.DiskCID, meta apiv1.DiskMeta) (apiv1.SnapshotCID, error) {
	id, err := c.uuidGen.Generate()
	if err != nil {
		return apiv1.SnapshotCID{}, err
	}

	snapshotID := SNAPSHOT_PREFIX + id

	post := api.StorageVolumeSnapshotsPost{
		Name:        snapshotID,
		Description: diskCID.AsString(),
	}
	err = wait(c.client.CreateStoragePoolVolumeSnapshot(c.config.Server.StoragePool, "custom", diskCID.AsString(), post))

	// We need both the volumeName as well as the snapshotName for deletion
	// and the only way to encode that seems to be in the snapshotCID.
	return apiv1.NewSnapshotCID(diskCID.AsString() + "_" + snapshotID), err
}
