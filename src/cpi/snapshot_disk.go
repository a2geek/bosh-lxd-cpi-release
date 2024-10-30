package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

func (c CPI) SnapshotDisk(diskCID apiv1.DiskCID, meta apiv1.DiskMeta) (apiv1.SnapshotCID, error) {
	id, err := c.uuidGen.Generate()
	if err != nil {
		return apiv1.SnapshotCID{}, err
	}

	snapshotID := SNAPSHOT_PREFIX + id

	err = c.adapter.CreateStoragePoolVolumeSnapshot(c.config.Server.StoragePool, diskCID.AsString(), snapshotID, diskCID.AsString())

	// We need both the volumeName as well as the snapshotName for deletion
	// and the only way to encode that seems to be in the snapshotCID.
	return apiv1.NewSnapshotCID(diskCID.AsString() + "_" + snapshotID), err
}
