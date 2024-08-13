package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c CPI) addDevice(vmCID apiv1.VMCID, name string, device map[string]string) error {
	instance, _, err := c.client.GetInstance(vmCID.AsString())
	if err != nil {
		return bosherr.WrapErrorf(err, "addDevice(%s) - GetInstance", vmCID.AsString())
	}

	// Check if the device already exists
	_, ok := instance.Devices[name]
	if ok {
		return bosherr.WrapErrorf(err, "addDevice(%s) - %s device already exists ", vmCID.AsString(), name)
	}

	instance.Devices[name] = device

	op, err := c.client.UpdateInstance(vmCID.AsString(), instance.Writable(), "")
	if err != nil {
		return bosherr.WrapErrorf(err, "addDevice(%s) - Update instance state", vmCID.AsString())
	}

	err = op.Wait()
	if c.checkError(err) != nil {
		return bosherr.WrapErrorf(err, "addDevice(%s) - Update instance state - wait", vmCID.AsString())
	}

	return nil
}

func (c CPI) removeDevice(vmCID apiv1.VMCID, deviceName string) error {
	instance, _, err := c.client.GetInstance(vmCID.AsString())
	if err != nil {
		return bosherr.WrapErrorf(err, "removeDevice(%s) - Get instance state", vmCID.AsString())
	}

	delete(instance.Devices, deviceName)

	op, err := c.client.UpdateInstance(vmCID.AsString(), instance.Writable(), "")
	if c.checkError(err) != nil {
		return bosherr.WrapErrorf(err, "removeDevice(%s) - Update instance state", vmCID.AsString())
	}

	return op.Wait()
}
