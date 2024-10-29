package cpi

import (
	"strings"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

const (
	DISK_EPHEMERAL_PREFIX     = "vol-e-"
	DISK_PERSISTENT_PREFIX    = "vol-p-"
	DISK_CONFIGURATION_PREFIX = "vol-c-"
	SNAPSHOT_PREFIX           = "snap-"
)

// TODO remove
func (c CPI) attachDiskToVM(vmCID apiv1.VMCID, diskId string) error {
	return c.attachDiskDeviceToVM(vmCID, diskId)
}

func (c CPI) attachDiskDeviceToVM(vmCID apiv1.VMCID, diskId string) error {
	device := map[string]string{
		"type":   "disk",
		"pool":   c.config.Server.StoragePool,
		"source": diskId,
	}
	return c.adapter.AttachDevice(vmCID.AsString(), diskId, device)
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

func (c CPI) findEphemeralDisksAttachedToVM(cid apiv1.VMCID) ([]string, error) {
	disks, err := c.findDisksAttachedToVm(cid)
	if err != nil {
		return nil, err
	}

	var ephemeral []string
	for name := range disks {
		if strings.HasPrefix(name, DISK_EPHEMERAL_PREFIX) {
			ephemeral = append(ephemeral, name)
		}
	}
	return ephemeral, nil
}

func (c CPI) findConfigurationDisksAttachedToVM(cid apiv1.VMCID) ([]string, error) {
	disks, err := c.findDisksAttachedToVm(cid)
	if err != nil {
		return nil, err
	}

	// this should only be 1, but there are likely still bugs
	var configuration []string
	for name := range disks {
		if strings.HasPrefix(name, DISK_CONFIGURATION_PREFIX) {
			configuration = append(configuration, name)
		}
	}
	return configuration, nil
}
