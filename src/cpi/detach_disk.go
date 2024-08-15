package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c CPI) DetachDisk(vmCID apiv1.VMCID, diskCID apiv1.DiskCID) error {
	err := c.freezeVM(vmCID)
	if err != nil {
		return bosherr.WrapError(err, "Stopping instance")
	}

	instance, _, err := c.client.GetInstance(vmCID.AsString())
	if err != nil {
		return bosherr.WrapError(err, "Get instance state")
	}

	// Check if the device already exists
	_, ok := instance.Devices[diskCID.AsString()]
	if !ok {
		return bosherr.WrapError(err, "Device already exists: "+diskCID.AsString())
	}

	delete(instance.Devices, diskCID.AsString())

	err = wait(c.client.UpdateInstance(vmCID.AsString(), instance.Writable(), ""))
	if err != nil {
		return bosherr.WrapError(err, "Update instance state")
	}

	agentEnv, err := c.agentMgr.Read(vmCID)
	if err != nil {
		return bosherr.WrapError(err, "Retrieve AgentEnv")
	}

	agentEnv.DetachPersistentDisk(diskCID)

	err = c.writeAgentFileToVM(vmCID, agentEnv)
	if err != nil {
		return bosherr.WrapError(err, "Write AgentEnv")
	}

	err = c.unfreezeVM(vmCID)
	if err != nil {
		return bosherr.WrapError(err, "Starting instance")
	}

	return nil
}
