package main

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cppforlife/bosh-cpi-go/apiv1"
)

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
