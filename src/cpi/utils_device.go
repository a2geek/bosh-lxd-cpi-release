package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c CPI) addDevice(vmCID apiv1.VMCID, name string, device map[string]string) error {
	instance, etag, err := c.client.GetInstance(vmCID.AsString())
	if err != nil {
		return bosherr.WrapError(err, "Get instance state")
	}

	// Check if the device already exists
	_, ok := instance.Devices[name]
	if ok {
		return bosherr.WrapError(err, "Device already exists: "+name)
	}

	instance.Devices[name] = device

	op, err := c.client.UpdateInstance(vmCID.AsString(), instance.Writable(), etag)
	if err != nil {
		return bosherr.WrapError(err, "Update instance state")
	}

	err = op.Wait()
	if err != nil {
		return bosherr.WrapError(err, "Update instance state - wait")
	}

	return nil
}
