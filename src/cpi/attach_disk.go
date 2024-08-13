package cpi

import (
	"strings"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c CPI) AttachDisk(vmCID apiv1.VMCID, diskCID apiv1.DiskCID) error {
	_, err := c.AttachDiskV2(vmCID, diskCID)
	return err
}

func (c CPI) AttachDiskV2(vmCID apiv1.VMCID, diskCID apiv1.DiskCID) (apiv1.DiskHint, error) {
	err := c.freezeVM(vmCID)
	if err != nil {
		return apiv1.NewDiskHintFromString(""), bosherr.WrapError(err, "Stopping instance")
	}

	agentEnv, err := c.agentMgr.Read(vmCID)
	if err != nil {
		return apiv1.NewDiskHintFromString(""), bosherr.WrapError(err, "Read AgentEnv")
	}

	// Not seeing a device mapping in LXD itself, but it's buried in the AgentEnv (and not exposed, so searching the JSON)
	rawBytes, err := agentEnv.AsBytes()
	if err != nil {
		return apiv1.NewDiskHintFromString(""), err
	}
	json := string(rawBytes)
	var path string
	if !strings.Contains(json, "/dev/sdc") {
		path = "/dev/sdc"
	} else if !strings.Contains(json, "/dev/sdd") {
		path = "/dev/sdd"
	} else {
		return apiv1.NewDiskHintFromString(""), bosherr.Error("Unable to find device for persistent disk")
	}

	err = c.attachDiskToVM(vmCID, diskCID.AsString())
	if err != nil {
		return apiv1.NewDiskHintFromString(""), bosherr.WrapError(err, "Attach disk")
	}

	diskHint := apiv1.NewDiskHintFromMap(map[string]interface{}{"path": path})
	agentEnv.AttachPersistentDisk(diskCID, diskHint)

	err = c.writeAgentFileToVM(vmCID, agentEnv)
	if err != nil {
		return apiv1.NewDiskHintFromString(""), bosherr.WrapError(err, "Write AgentEnv")
	}

	err = c.unfreezeVM(vmCID)
	if err != nil {
		return apiv1.NewDiskHintFromString(""), bosherr.WrapError(err, "Starting instance")
	}

	return diskHint, nil
}
