package cpi

import (
	"bytes"
	"fmt"

	lxdclient "github.com/canonical/lxd/client"

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

	diskName := DISK_CONFIGURATION_PREFIX + uuid

	buf := bytes.NewBuffer(diskImage)
	err = wait(c.client.CreateStoragePoolVolumeFromISO(c.config.Server.StoragePool, lxdclient.StoragePoolVolumeBackupArgs{
		Name:       diskName,
		BackupFile: buf,
	}))
	if err != nil {
		return bosherr.WrapErrorf(err, "writeAgentFileToVm(%s) - Create", vmCID.AsString())
	}

	configuration, err := c.findConfigurationDisksAttachedToVM(vmCID)
	if err != nil {
		return bosherr.WrapErrorf(err, "writeAgentFileToVm(%s) - Find", vmCID.AsString())
	}
	for _, configDiskName := range configuration {
		err = c.removeDevice(vmCID, configDiskName)
		if err != nil {
			return bosherr.WrapErrorf(err, "writeAgentFileToVm(%s) - Remove", vmCID.AsString())
		}
		err = c.client.DeleteStoragePoolVolume(c.config.Server.StoragePool, "custom", configDiskName)
		if err != nil {
			return bosherr.WrapErrorf(err, "writeAgentFileToVm(%s) - Delete", vmCID.AsString())
		}
	}

	err = c.attachDiskDeviceToVM(vmCID, diskName)
	if err != nil {
		return bosherr.WrapErrorf(err, "writeAgentFileToVm(%s) - Attach", vmCID.AsString())
	}
	return nil
}
