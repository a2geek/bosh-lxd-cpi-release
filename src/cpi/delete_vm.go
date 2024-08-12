package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c CPI) DeleteVM(vmCID apiv1.VMCID) error {
	err := c.stopVM(vmCID)
	if err != nil {
		return bosherr.WrapError(err, "Delete VM - stop")
	}

	disks, err := c.findEphemeralDisksAttachedToVM(vmCID)
	if err != nil {
		return bosherr.WrapError(err, "Delete VM - enumerate ephemeral disks")
	}

	op, err := c.client.DeleteInstance(vmCID.AsString())
	if err != nil {
		return bosherr.WrapError(err, "Delete VM")
	}
	err = op.Wait()
	if err != nil {
		return bosherr.WrapError(err, "Delete VM - wait")
	}

	for _, disk := range disks {
		err = c.client.DeleteStoragePoolVolume(c.config.Server.StoragePool, "custom", disk)
		if err != nil {
			return bosherr.WrapError(err, "Delete VM - attached disk - "+disk)
		}
	}

	return c.agentMgr.Delete(vmCID)
}
