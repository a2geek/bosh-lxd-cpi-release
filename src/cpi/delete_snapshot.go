package cpi

import (
	"strings"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c CPI) DeleteSnapshot(snapshotCID apiv1.SnapshotCID) error {
	parts := strings.Split(snapshotCID.AsString(), "_")
	if len(parts) != 2 {
		return bosherr.Error("expecting snapshot CID to be two parts separated by '/'")
	}

	volumeName, snapshotName := parts[0], parts[1]
	if volumeName == "" || snapshotName == "" {
		return bosherr.Error("expecting snapshot CID to include both volume and snapshot names")
	}

	return c.adapter.DeleteStoragePoolVolumeSnapshot(c.config.Server.StoragePool, volumeName, snapshotName)
}
