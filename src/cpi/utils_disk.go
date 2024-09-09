package cpi

import (
	"fmt"
	"strings"

	"github.com/canonical/lxd/shared/api"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

const (
	DISK_EPHEMERAL_PREFIX     = "vol-e-"
	DISK_PERSISTENT_PREFIX    = "vol-p-"
	DISK_CONFIGURATION_PREFIX = "vol-c-"
	SNAPSHOT_PREFIX           = "snap-"
)

func (c CPI) createDisk(size int, name string) error {
	storageVolumeRequest := api.StorageVolumesPost{
		Name:        name,
		Type:        "custom",
		ContentType: "block",
		StorageVolumePut: api.StorageVolumePut{
			Config: map[string]string{
				"size": fmt.Sprintf("%dMiB", size),
			},
		},
	}

	return c.client.CreateStoragePoolVolume(c.config.Server.StoragePool, storageVolumeRequest)
}

func (c CPI) attachDiskToVM(vmCID apiv1.VMCID, diskId string) error {
	return c.attachDiskDeviceToVM(vmCID, diskId)
}

func (c CPI) attachDiskDeviceToVM(vmCID apiv1.VMCID, diskId string) error {
	device := map[string]string{
		"type":   "disk",
		"pool":   c.config.Server.StoragePool,
		"source": diskId,
	}
	return c.addDevice(vmCID, diskId, device)
}

func (c CPI) findDisksAttachedToVm(vmCID apiv1.VMCID) (map[string]map[string]string, error) {
	instance, _, err := c.client.GetInstance(vmCID.AsString())
	if err != nil {
		return nil, err
	}

	devices := make(map[string]map[string]string)
	for name, device := range instance.Devices {
		if device["type"] == "disk" {
			devices[name] = device
		}
	}
	return devices, nil
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

func (c CPI) setDiskMetadata(cid apiv1.DiskCID, description string) error {
	volume, _, err := c.client.GetStoragePoolVolume(c.config.Server.StoragePool, "custom", cid.AsString())
	if err != nil {
		return err
	}

	volume.Description = description

	return c.client.UpdateStoragePoolVolume(c.config.Server.StoragePool, "custom", cid.AsString(), volume.Writable(), "")
}
