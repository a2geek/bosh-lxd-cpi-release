package cpi

import (
	"fmt"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c CPI) SetVMMetadata(cid apiv1.VMCID, metadata apiv1.VMMeta) error {
	actual, err := NewActualVMMeta(metadata)
	if err != nil {
		return bosherr.WrapError(err, "Unmarshal VMMeta to ActualVMMeta")
	}

	instance, _, err := c.client.GetInstance(cid.AsString())
	if err != nil {
		return bosherr.WrapError(err, "Get instance state")
	}

	description := fmt.Sprintf("%s/%s", actual.Job, actual.Index)
	instance.Description = description

	err = wait(c.client.UpdateInstance(cid.AsString(), instance.Writable(), ""))
	if err != nil {
		return bosherr.WrapErrorf(err, "Update instance state; status=%s", instance.Status)
	}

	disks, err := c.findEphemeralDisksAttachedToVM(cid)
	if err != nil {
		return bosherr.WrapError(err, "Delete VM - enumerate ephemeral disks")
	}

	for _, disk := range disks {
		err = c.setDiskMetadata(apiv1.NewDiskCID(disk), description)
		if err != nil {
			return bosherr.WrapError(err, "Update storage volume description")
		}
	}

	return nil
}
