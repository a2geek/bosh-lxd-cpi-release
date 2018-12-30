package main

import (
	"fmt"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cppforlife/bosh-cpi-go/apiv1"
	"github.com/lxc/lxd/shared/api"
)

func (c CPI) GetDisks(cid apiv1.VMCID) ([]apiv1.DiskCID, error) {
	return []apiv1.DiskCID{}, nil
}

func (c CPI) CreateDisk(size int,
	cloudProps apiv1.DiskCloudProps, associatedVMCID *apiv1.VMCID) (apiv1.DiskCID, error) {

	id, err := c.uuidGen.Generate()
	if err != nil {
		return apiv1.DiskCID{}, bosherr.WrapError(err, "Creating Disk id")
	}
	theCid := "vol-" + id
	diskCid := apiv1.NewDiskCID(theCid)

	storageVolumeRequest := api.StorageVolumesPost{
		Name: theCid,
		Type: "custom",
		StorageVolumePut: api.StorageVolumePut{
			Config: map[string]string{
				"size": fmt.Sprintf("%dMB", size),
			},
		},
	}

	// FIXME: default is assumed to be name
	err = c.client.CreateStoragePoolVolume("default", storageVolumeRequest)
	if err != nil {
		return apiv1.DiskCID{}, bosherr.WrapError(err, "Creating volume")
	}

	return diskCid, nil
}

func (c CPI) DeleteDisk(cid apiv1.DiskCID) error {
	err := c.client.DeleteStoragePoolVolume("default", "custom", cid.AsString())
	if err != nil {
		return bosherr.WrapError(err, "Deleting volume")
	}

	return nil
}

func (c CPI) AttachDisk(vmCID apiv1.VMCID, diskCID apiv1.DiskCID) error {
	return nil
}

func (c CPI) AttachDiskV2(vmCID apiv1.VMCID, diskCID apiv1.DiskCID) (apiv1.DiskHint, error) {
	return apiv1.NewDiskHintFromString(""), nil
}

func (c CPI) DetachDisk(vmCID apiv1.VMCID, diskCID apiv1.DiskCID) error {
	return nil
}

func (c CPI) HasDisk(cid apiv1.DiskCID) (bool, error) {
	return false, nil
}

func (c CPI) SetDiskMetadata(cid apiv1.DiskCID, metadata apiv1.DiskMeta) error {
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
