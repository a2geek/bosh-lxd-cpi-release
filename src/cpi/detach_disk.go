package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c CPI) DetachDisk(vmCID apiv1.VMCID, diskCID apiv1.DiskCID) error {
	err := c.adapter.DetachDeviceBySource(vmCID.AsString(), diskCID.AsString())
	if err != nil {
		return bosherr.WrapError(err, "Detach Device")
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

	return nil
}
