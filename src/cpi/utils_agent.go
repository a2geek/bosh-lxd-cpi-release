package cpi

import (
	"bytes"
	"fmt"

	lxdclient "github.com/canonical/lxd/client"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

func (c CPI) writeAgentFileToVM(vmCID apiv1.VMCID, agentEnv apiv1.AgentEnv) error {
	uuid, err := c.uuidGen.Generate()
	if err != nil {
		return err
	}

	diskImage, err := c.agentMgr.Write(vmCID, agentEnv)
	if err != nil {
		return err
	}
	if len(diskImage) == 0 {
		return fmt.Errorf("ISO image is empty")
	}

	diskName := DISK_CONFIGURATION_PREFIX + uuid

	buf := bytes.NewBuffer(diskImage)
	op, err := c.client.CreateStoragePoolVolumeFromISO(c.config.Server.StoragePool, lxdclient.StoragePoolVolumeBackupArgs{
		Name:       diskName,
		BackupFile: buf,
	})
	if err != nil {
		return err
	}

	err = op.Wait()
	if err != nil {
		return err
	}

	err = c.attachDiskDeviceToVM(vmCID, diskName)
	return err
}

func (c CPI) readAgentFileFromVM(vmCID apiv1.VMCID) (apiv1.AgentEnv, error) {
	return c.agentMgr.Read(vmCID)
}
