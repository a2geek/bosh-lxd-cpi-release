package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c CPI) HasDisk(cid apiv1.DiskCID) (bool, error) {
	etag, err := c.adapter.GetStoragePoolVolume(c.config.Server.StoragePool, cid.AsString())
	if err != nil {
		return false, bosherr.WrapError(err, "Locating storage volume")
	}

	return len(etag) > 0, nil
}
