package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
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
