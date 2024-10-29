package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c CPI) DeleteDisk(cid apiv1.DiskCID) error {
	err := c.adapter.DeleteStoragePoolVolume(c.config.Server.StoragePool, cid.AsString())
	if err != nil {
		return bosherr.WrapError(err, "Deleting volume")
	}

	return nil
}
