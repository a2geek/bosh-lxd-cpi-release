package cpi

import (
	"fmt"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c CPI) SetVMMetadata(cid apiv1.VMCID, metadata apiv1.VMMeta) error {
	actual, err := NewActualVMMeta(metadata)
	if err != nil {
		return bosherr.WrapError(err, "SetVMMetadata - Unmarshal VMMeta to ActualVMMeta")
	}

	description := fmt.Sprintf("%s/%s", actual.Job, actual.Index)
	err = c.adapter.UpdateInstanceDescription(cid.AsString(), description)
	if err != nil {
		return bosherr.WrapError(err, "SetVMMetadata - update instance description")
	}

	disks, err := c.findEphemeralDisksAttachedToVM(cid)
	if err != nil {
		return bosherr.WrapError(err, "SetVMMetadata - enumerate ephemeral disks")
	}

	for _, disk := range disks {
		err = c.adapter.UpdateStoragePoolVolumeDescription(c.config.Server.StoragePool, disk, description)
		if err != nil {
			return bosherr.WrapError(err, "SetVMMetadata - Update storage volume description")
		}
	}

	return nil
}
