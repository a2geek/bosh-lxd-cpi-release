package main

import (
	"fmt"
	"strings"

	"github.com/cppforlife/bosh-cpi-go/apiv1"
	"github.com/lxc/lxd/shared/api"
)

func (c CPI) createDisk(size int, name string) error {
	storageVolumeRequest := api.StorageVolumesPost{
		Name: name,
		Type: "custom",
		StorageVolumePut: api.StorageVolumePut{
			Config: map[string]string{
				"size": fmt.Sprintf("%dMB", size),
			},
		},
	}

	return c.client.CreateStoragePoolVolume(c.config.Server.StoragePool, storageVolumeRequest)
}

func (c CPI) attachDiskToVM(vmCID apiv1.VMCID, diskId string) (string, error) {
	return c.attachDiskDeviceToVM(vmCID, diskId, "/warden-cpi-dev/"+diskId)
}

func (c CPI) attachDiskDeviceToVM(vmCID apiv1.VMCID, diskId string, devicePath string) (string, error) {
	device := map[string]string{
		"type":   "disk",
		"pool":   c.config.Server.StoragePool,
		"path":   devicePath,
		"source": diskId,
	}
	return devicePath, c.addDevice(vmCID, diskId, device)
}

func (c CPI) findDisksAttachedToVm(vmCID apiv1.VMCID) (map[string]map[string]string, error) {
	container, _, err := c.client.GetContainer(vmCID.AsString())
	if err != nil {
		return nil, err
	}

	devices := make(map[string]map[string]string)
	for name, device := range container.Devices {
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

func (c CPI) setDiskMetadata(cid apiv1.DiskCID, description string) error {
	volume, etag, err := c.client.GetStoragePoolVolume(c.config.Server.StoragePool, "custom", cid.AsString())
	if err != nil {
		return err
	}

	volume.Description = description

	return c.client.UpdateStoragePoolVolume(c.config.Server.StoragePool, "custom", cid.AsString(), volume.Writable(), etag)
}
