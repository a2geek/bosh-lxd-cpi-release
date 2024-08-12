package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c CPI) DetachDisk(vmCID apiv1.VMCID, diskCID apiv1.DiskCID) error {
	err := c.stopVM(vmCID)
	if err != nil {
		return bosherr.WrapError(err, "Stopping instance")
	}

	instance, etag, err := c.client.GetInstance(vmCID.AsString())
	if err != nil {
		return bosherr.WrapError(err, "Get instance state")
	}

	// Check if the device already exists
	_, ok := instance.Devices[diskCID.AsString()]
	if !ok {
		return bosherr.WrapError(err, "Device already exists: "+diskCID.AsString())
	}

	delete(instance.Devices, diskCID.AsString())

	op, err := c.client.UpdateInstance(vmCID.AsString(), instance.Writable(), etag)
	if err != nil {
		return bosherr.WrapError(err, "Update instance state")
	}

	err = op.Wait()
	if err != nil {
		return bosherr.WrapError(err, "Update instance state - wait")
	}

	agentEnv, err := c.readAgentFileFromVM(vmCID)
	if err != nil {
		return bosherr.WrapError(err, "Retrieve AgentEnv")
	}

	agentEnv.DetachPersistentDisk(diskCID)

	err = c.writeAgentFileToVM(vmCID, agentEnv)
	if err != nil {
		return bosherr.WrapError(err, "Write AgentEnv")
	}

	err = c.startVM(vmCID)
	if err != nil {
		return bosherr.WrapError(err, "Starting instance")
	}

	return nil
}
