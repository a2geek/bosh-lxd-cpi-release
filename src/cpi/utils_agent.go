package cpi

import (
	"bosh-lxd-cpi/agentmgr"
	"bytes"
	"fmt"
	"io"

	lxdclient "github.com/canonical/lxd/client"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

const AGENT_PATH = "/var/vcap/bosh/warden-cpi-agent-env.json"

func (c CPI) writeAgentFileToVM(vmCID apiv1.VMCID, agentEnv apiv1.AgentEnv) error {
	agentmgr, err := agentmgr.NewCDROMManager(c.config.AgentConfig)
	if err != nil {
		return err
	}

	uuid, err := c.uuidGen.Generate()
	if err != nil {
		return err
	}

	err = agentmgr.Update(agentEnv)
	if err != nil {
		return err
	}

	diskName := DISK_CONFIGURATION_PREFIX + uuid
	diskImage, err := agentmgr.ToBytes()
	if err != nil {
		return err
	}
	if len(diskImage) == 0 {
		return fmt.Errorf("ISO image is empty")
	}

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

	_, err = c.attachDiskDeviceToVM(vmCID, diskName, "")
	return err

	// var buf bytes.Buffer
	// tw := tar.NewWriter(gzip.NewWriter(&buf))
	// err = tw.WriteHeader(&tar.Header{
	// 	Name: diskName,
	// 	Size: int64(len(diskImage)),
	// })
	// if err != nil {
	// 	return err
	// }
	// _, err = tw.Write(diskImage)
	// if err != nil {
	// 	return err
	// }
	// err = tw.Close()
	// if err != nil {
	// 	return err
	// }

	// c.logger.Debug("writeAgentFileToVM", ">>> CREATE STORAGE POOL FROM BACKUP")
	// op, err := c.client.CreateStoragePoolVolumeFromBackup(c.config.Server.StoragePool, lxdclient.StoragePoolVolumeBackupArgs{
	// 	Name:       diskName,
	// 	BackupFile: &buf,
	// })
	// if err != nil {
	// 	return err
	// }
	// c.logger.Debug("writeAgentFileToVM", "<<< DONE")

	// return op.Wait()
}

func (c CPI) readAgentFileFromVM(vmCID apiv1.VMCID) (apiv1.AgentEnv, error) {
	reader, _, err := c.client.GetInstanceFile(vmCID.AsString(), AGENT_PATH)
	if err != nil {
		return nil, bosherr.WrapError(err, "Retrieve AgentEnv")
	}
	defer reader.Close()

	bytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, bosherr.WrapError(err, "Read AgentEnv bytes")
	}

	factory := apiv1.NewAgentEnvFactory()
	agentEnv, err := factory.FromBytes(bytes)
	if err != nil {
		return nil, bosherr.WrapError(err, "Make AgentEnv from bytes")
	}

	return agentEnv, nil
}
