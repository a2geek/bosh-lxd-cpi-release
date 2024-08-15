package cpi

import (
	"fmt"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

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