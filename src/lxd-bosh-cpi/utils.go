package main

import (
	"fmt"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cppforlife/bosh-cpi-go/apiv1"
	"github.com/lxc/lxd/shared/api"
)

func (c CPI) stopVM(cid apiv1.VMCID) error {
	return c.setVMAction(cid, "stop")
}

func (c CPI) startVM(cid apiv1.VMCID) error {
	return c.setVMAction(cid, "start")
}

func (c CPI) setVMAction(cid apiv1.VMCID, action string) error {
	req := api.ContainerStatePut{
		Action:   action,
		Timeout:  30,
		Force:    true,
		Stateful: false,
	}

	op, err := c.client.UpdateContainerState(cid.AsString(), req, "")
	if err != nil {
		return bosherr.WrapError(err, "Set VM Action - "+action)
	}

	err = op.Wait()
	if err != nil {
		return bosherr.WrapError(err, "Set VM Action - wait - "+action)
	}

	return nil
}

func (c CPI) addDevice(vmCID apiv1.VMCID, name string, device map[string]string) error {
	container, etag, err := c.client.GetContainer(vmCID.AsString())
	if err != nil {
		return bosherr.WrapError(err, "Get container state")
	}

	// Check if the device already exists
	_, ok := container.Devices[name]
	if ok {
		return bosherr.WrapError(err, "Device already exists: "+name)
	}

	container.Devices[name] = device

	op, err := c.client.UpdateContainer(vmCID.AsString(), container.Writable(), etag)
	if err != nil {
		return bosherr.WrapError(err, "Update container state")
	}

	err = op.Wait()
	if err != nil {
		return bosherr.WrapError(err, "Update container state - wait")
	}

	return nil
}

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

	// FIXME: default is assumed to be name
	return c.client.CreateStoragePoolVolume("default", storageVolumeRequest)
}

func (c CPI) attachDiskToVM(vmCID apiv1.VMCID, diskId string) (string, error) {
	return c.attachDiskDeviceToVM(vmCID, diskId, "/warden-cpi-dev/"+diskId)
}

func (c CPI) attachDiskDeviceToVM(vmCID apiv1.VMCID, diskId string, devicePath string) (string, error) {
	device := map[string]string{
		"type":   "disk",
		"pool":   "default",
		"path":   devicePath,
		"source": diskId,
	}
	return devicePath, c.addDevice(vmCID, diskId, device)
}
