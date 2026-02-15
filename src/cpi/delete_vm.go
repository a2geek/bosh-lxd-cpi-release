package cpi

import (
	"bosh-lxd-cpi/adapter"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c CPI) DeleteVM(vmCID apiv1.VMCID) error {
	err := c.adapter.SetInstanceAction(vmCID.AsString(), adapter.StopAction)
	if err != nil {
		return bosherr.WrapError(err, "Delete VM - stop")
	}

	disks, err := c.findDisksAttachedToVm(vmCID)
	if err != nil {
		return bosherr.WrapError(err, "Delete VM - enumerate disks")
	}

	err = c.adapter.DeleteInstance(vmCID.AsString())
	if err != nil {
		return bosherr.WrapError(err, "Delete VM")
	}

	for name, disk := range disks {
		if name == DISK_DEVICE_CONFIG || name == DISK_DEVICE_EPHEMERAL {
			diskId := disk["source"]
			err = c.adapter.DeleteStoragePoolVolume(c.config.Server.StoragePool, diskId)
			if err != nil {
				return bosherr.WrapErrorf(err, "Delete VM - attached disk - %s", diskId)
			}
		}
	}

	return c.agentMgrVM.Delete(vmCID)
}
