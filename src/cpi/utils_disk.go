package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

const (
	DISK_EPHEMERAL_PREFIX     = "vol-e-"
	DISK_PERSISTENT_PREFIX    = "vol-p-"
	DISK_CONFIGURATION_PREFIX = "vol-c-"
	SNAPSHOT_PREFIX           = "snap-"
	DISK_DEVICE_CONFIG        = "config"
	DISK_DEVICE_EPHEMERAL     = "ephemeral"
	DISK_DEVICE_PERSISTENT1   = "persistent-1"
	DISK_DEVICE_PERSISTENT2   = "persistent-2"
)

func (c CPI) attachDiskDeviceToVM(vmCID apiv1.VMCID, name, diskId string) error {
	device := map[string]string{
		"type":   "disk",
		"pool":   c.config.Server.StoragePool,
		"source": diskId,
	}
	return c.adapter.AttachDevice(vmCID.AsString(), name, device)
}

func (c CPI) findDisksAttachedToVm(vmCID apiv1.VMCID) (map[string]map[string]string, error) {
	devices, err := c.adapter.GetDevices(vmCID.AsString())
	if err != nil {
		return nil, err
	}

	disks := make(map[string]map[string]string)
	for name, device := range devices {
		if device["type"] == "disk" {
			disks[name] = device
		}
	}
	return disks, nil
}
