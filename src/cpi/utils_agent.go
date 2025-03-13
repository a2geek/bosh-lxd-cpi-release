package cpi

import (
	"bytes"
	"fmt"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c CPI) writeAgentFileToVM(vmCID apiv1.VMCID, agentEnv apiv1.AgentEnv) error {
	uuid, err := c.uuidGen.Generate()
	if err != nil {
		return bosherr.WrapErrorf(err, "writeAgentFileToVm(%s) - UUID", vmCID.AsString())
	}

	diskImage, err := c.agentMgr.Write(vmCID, agentEnv)
	if err != nil {
		return bosherr.WrapErrorf(err, "writeAgentFileToVm(%s) - Write", vmCID.AsString())
	}
	if len(diskImage) == 0 {
		return fmt.Errorf("ISO image is empty")
	}

	target, err := c.adapter.GetInstanceLocation(vmCID.AsString())
	if err != nil {
		return bosherr.WrapErrorf(err, "writeAgentFileToVm(%s) - Location", vmCID.AsString())
	}

	diskName := DISK_CONFIGURATION_PREFIX + uuid

	buf := bytes.NewBuffer(diskImage)
	err = c.adapter.CreateStoragePoolVolumeFromISO(target, c.config.Server.StoragePool, diskName, buf)
	if err != nil {
		return bosherr.WrapErrorf(err, "writeAgentFileToVm(%s) - Create", vmCID.AsString())
	}

	disks, err := c.findDisksAttachedToVm(vmCID)
	if err != nil {
		return bosherr.WrapErrorf(err, "writeAgentFileToVm(%s) - Find", vmCID.AsString())
	}

	if disk, ok := disks[DISK_DEVICE_CONFIG]; ok {
		err = c.adapter.DetachDeviceByName(vmCID.AsString(), DISK_DEVICE_CONFIG)
		if err != nil {
			return bosherr.WrapErrorf(err, "writeAgentFileToVm(%s) - Remove", vmCID.AsString())
		}

		stopped, err := c.adapter.IsInstanceStopped(vmCID.AsString())
		if err != nil {
			return bosherr.WrapErrorf(err, "writeAgentFileToVm(%s) - Check State", vmCID.AsString())
		} else if stopped {
			err = c.adapter.DeleteStoragePoolVolume(c.config.Server.StoragePool, disk["source"])
			if err != nil {
				return bosherr.WrapErrorf(err, "writeAgentFileToVm(%s) - Delete", vmCID.AsString())
			}
		}
	}

	err = c.attachDiskDeviceToVM(vmCID, DISK_DEVICE_CONFIG, diskName)
	if err != nil {
		return bosherr.WrapErrorf(err, "writeAgentFileToVm(%s) - Attach", vmCID.AsString())
	}
	return nil
}
