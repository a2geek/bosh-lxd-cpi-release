package cpi

import (
	"fmt"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

const (
	DISK_EPHEMERAL_PREFIX     = "vol-e-"
	DISK_PERSISTENT_PREFIX    = "vol-p-"
	DISK_CONFIGURATION_PREFIX = "vol-c-"
)

func (c CPI) GetDisks(cid apiv1.VMCID) ([]apiv1.DiskCID, error) {
	disks, err := c.findDisksAttachedToVm(cid)
	if err != nil {
		return []apiv1.DiskCID{}, bosherr.WrapError(err, "GetDisks - locating disks")
	}

	var diskcids []apiv1.DiskCID
	for name := range disks {
		diskcids = append(diskcids, apiv1.NewDiskCID(name))
	}

	return diskcids, nil
}

func (c CPI) CreateDisk(size int,
	cloudProps apiv1.DiskCloudProps, associatedVMCID *apiv1.VMCID) (apiv1.DiskCID, error) {

	id, err := c.uuidGen.Generate()
	if err != nil {
		return apiv1.DiskCID{}, bosherr.WrapError(err, "Creating Disk id")
	}
	theCid := DISK_PERSISTENT_PREFIX + id
	diskCid := apiv1.NewDiskCID(theCid)

	// FIXME: default is assumed to be name
	err = c.createDisk(size, theCid)
	if err != nil {
		return apiv1.DiskCID{}, bosherr.WrapError(err, "Creating volume")
	}

	return diskCid, nil
}

func (c CPI) DeleteDisk(cid apiv1.DiskCID) error {
	err := c.client.DeleteStoragePoolVolume(c.config.Server.StoragePool, "custom", cid.AsString())
	if err != nil {
		return bosherr.WrapError(err, "Deleting volume")
	}

	return nil
}

func (c CPI) AttachDisk(vmCID apiv1.VMCID, diskCID apiv1.DiskCID) error {
	_, err := c.AttachDiskV2(vmCID, diskCID)
	return err
}

func (c CPI) AttachDiskV2(vmCID apiv1.VMCID, diskCID apiv1.DiskCID) (apiv1.DiskHint, error) {
	err := c.stopVM(vmCID)
	if err != nil {
		return apiv1.NewDiskHintFromString(""), bosherr.WrapError(err, "Stopping instance")
	}

	path, err := c.attachDiskToVM(vmCID, diskCID.AsString())
	if err != nil {
		return apiv1.NewDiskHintFromString(""), bosherr.WrapError(err, "Attach disk")
	}

	agentEnv, err := c.readAgentFileFromVM(vmCID)
	if err != nil {
		return apiv1.NewDiskHintFromString(""), bosherr.WrapError(err, "Read AgentEnv")
	}

	diskHint := apiv1.NewDiskHintFromMap(map[string]interface{}{"path": path})
	agentEnv.AttachPersistentDisk(diskCID, diskHint)

	err = c.writeAgentFileToVM(vmCID, agentEnv)
	if err != nil {
		return apiv1.NewDiskHintFromString(""), bosherr.WrapError(err, "Write AgentEnv")
	}

	err = c.startVM(vmCID)
	if err != nil {
		return apiv1.NewDiskHintFromString(""), bosherr.WrapError(err, "Starting instance")
	}

	return diskHint, nil
}

func (c CPI) DetachDisk(vmCID apiv1.VMCID, diskCID apiv1.DiskCID) error {
	err := c.stopVM(vmCID)
	if err != nil {
		return bosherr.WrapError(err, "Stopping instance")
	}

	instance, etag, err := c.client.GetInstance(vmCID.AsString())
	if err != nil {
		return bosherr.WrapError(err, "Get instance state")
	}

	// Check if the device already exists
	_, ok := instance.Devices[diskCID.AsString()]
	if !ok {
		return bosherr.WrapError(err, "Device already exists: "+diskCID.AsString())
	}

	delete(instance.Devices, diskCID.AsString())

	op, err := c.client.UpdateInstance(vmCID.AsString(), instance.Writable(), etag)
	if err != nil {
		return bosherr.WrapError(err, "Update instance state")
	}

	err = op.Wait()
	if err != nil {
		return bosherr.WrapError(err, "Update instance state - wait")
	}

	agentEnv, err := c.readAgentFileFromVM(vmCID)
	if err != nil {
		return bosherr.WrapError(err, "Retrieve AgentEnv")
	}

	agentEnv.DetachPersistentDisk(diskCID)

	err = c.writeAgentFileToVM(vmCID, agentEnv)
	if err != nil {
		return bosherr.WrapError(err, "Write AgentEnv")
	}

	err = c.startVM(vmCID)
	if err != nil {
		return bosherr.WrapError(err, "Starting instance")
	}

	return nil
}

func (c CPI) HasDisk(cid apiv1.DiskCID) (bool, error) {
	_, etag, err := c.client.GetStoragePoolVolume(c.config.Server.StoragePool, "custom", cid.AsString())
	if err != nil {
		return false, bosherr.WrapError(err, "Locating storage volume")
	}

	return len(etag) > 0, nil
}

func (c CPI) SetDiskMetadata(cid apiv1.DiskCID, metadata apiv1.DiskMeta) error {
	actual, err := NewActualDiskMeta(metadata)
	if err != nil {
		return bosherr.WrapError(err, "Unmarshalling DiskMeta")
	}

	err = c.setDiskMetadata(cid, fmt.Sprintf("%s/%s", actual.InstanceGroup, actual.InstanceIndex))
	if err != nil {
		return bosherr.WrapError(err, "Update storage volume description")
	}

	return nil
}

func (c CPI) ResizeDisk(cid apiv1.DiskCID, size int) error {
	return nil
}

func (c CPI) SnapshotDisk(cid apiv1.DiskCID, meta apiv1.DiskMeta) (apiv1.SnapshotCID, error) {
	return apiv1.NewSnapshotCID("snap-cid"), nil
}

func (c CPI) DeleteSnapshot(cid apiv1.SnapshotCID) error {
	return nil
}
