package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c CPI) CreateDisk(size int,
	cloudProps apiv1.DiskCloudProps, associatedVMCID *apiv1.VMCID) (apiv1.DiskCID, error) {

	id, err := c.uuidGen.Generate()
	if err != nil {
		return apiv1.DiskCID{}, bosherr.WrapError(err, "Creating Disk id")
	}
	theCid := DISK_PERSISTENT_PREFIX + id
	diskCid := apiv1.NewDiskCID(theCid)

	target, err := c.adapter.GetInstanceLocation(theCid)
	if err != nil {
		return apiv1.DiskCID{}, bosherr.WrapError(err, "Finding disk location")
	}

	err = c.adapter.CreateStoragePoolVolume(target, c.config.Server.StoragePool, theCid, size)
	if err != nil {
		return apiv1.DiskCID{}, bosherr.WrapError(err, "Creating volume")
	}

	return diskCid, nil
}
