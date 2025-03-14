package agentmgr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
)

func NewCDROMManager(config Config) AgentManager {
	mgr := cdromManager{
		agentFileManager{
			config: config,
		},
	}
	return mgr
}

type cdromManager struct {
	agentFileManager
}

func (m cdromManager) Write(vmCID apiv1.VMCID, agentEnv apiv1.AgentEnv) ([]byte, error) {
	// Based on sample: https://github.com/diskfs/go-diskfs/blob/master/examples/iso_create.go

	cdromFileName, err := m.tempFileName("iso")
	if err != nil {
		return nil, err
	}

	var diskSize int64 = 5 * 1024 * 1024 // 5 MB
	image, err := diskfs.Create(cdromFileName, diskSize, diskfs.SectorSizeDefault)
	if err != nil {
		return nil, err
	}

	image.LogicalBlocksize = 2048
	fspec := disk.FilesystemSpec{
		Partition:   0,
		FSType:      filesystem.TypeISO9660,
		VolumeLabel: m.config.Label,
	}
	fs, err := image.CreateFilesystem(fspec)
	if err != nil {
		return nil, err
	}

	// The AgentEnv goes into userdata
	buf, err := m.writeAgentEnv(vmCID, agentEnv)
	if err != nil {
		return nil, err
	}

	err = m.mkdirAll(fs, filepath.Dir(m.config.UserdataPath))
	if err != nil {
		return nil, err
	}

	err = m.writeFile(fs, m.config.UserdataPath, buf)
	if err != nil {
		return nil, err
	}

	// Metadata contains the SSH key
	metadata := metadataContentsType{
		// TODO uncertain if this is actually needed for our configurations...
		// PublicKeys: map[string]publicKeyType{
		// 	"0": {
		// 		"openssh-key": c.config.VMPublicKey,
		// 	},
		// },
	}
	metaDataContent, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	err = m.mkdirAll(fs, filepath.Dir(m.config.MetadataPath))
	if err != nil {
		return nil, err
	}

	err = m.writeFile(fs, m.config.MetadataPath, metaDataContent)
	if err != nil {
		return nil, err
	}

	iso, ok := fs.(*iso9660.FileSystem)
	if !ok {
		return nil, fmt.Errorf("not an iso9660 filesystem")
	}
	err = iso.Finalize(iso9660.FinalizeOptions{
		RockRidge:        true,
		VolumeIdentifier: m.config.Label,
	})
	if err != nil {
		return nil, err
	}
	return os.ReadFile(cdromFileName)
}
