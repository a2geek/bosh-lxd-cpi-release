package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c CPI) DeleteVM(vmCID apiv1.VMCID) error {
	err := c.adapter.SetInstanceAction(vmCID.AsString(), "stop")
	if err != nil {
		return bosherr.WrapError(err, "Delete VM - stop")
	}

	ephemeralDisks, err := c.findEphemeralDisksAttachedToVM(vmCID)
	if err != nil {
		return bosherr.WrapError(err, "Delete VM - enumerate ephemeral disks")
	}

	configurationDisks, err := c.findConfigurationDisksAttachedToVM(vmCID)
	if err != nil {
		return bosherr.WrapError(err, "Delete VM - enumerate configuration disks")
	}

	err = c.adapter.DeleteInstance(vmCID.AsString())
	if err != nil {
		return bosherr.WrapError(err, "Delete VM")
	}

	disks := append(ephemeralDisks, configurationDisks...)
	for _, disk := range disks {
		err = c.adapter.DeleteStoragePoolVolume(c.config.Server.StoragePool, "custom", disk)
		if err != nil {
			return bosherr.WrapError(err, "Delete VM - attached disk - "+disk)
		}
	}

	return c.agentMgr.Delete(vmCID)
}
