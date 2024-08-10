package agentmgr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/partition/mbr"
)

// NewFAT32Manager will initialize a new config drive for AgentEnv settings
func NewFAT32Manager(config Config) (AgentManager, error) {
	mgr := fat32Manager{
		config: config,
	}
	return mgr, nil
}

// These are "stolen" out of the Bosh Agent itself.
type metadataContentsType struct {
	PublicKeys map[string]publicKeyType `json:"public-keys"`
}
type publicKeyType map[string]string

type fat32Manager struct {
	agentFileManager
	config Config
}

func (c fat32Manager) Write(vmCID apiv1.VMCID, agentEnv apiv1.AgentEnv) ([]byte, error) {
	diskFileName, err := c.tempFileName("fat32")
	if err != nil {
		return nil, err
	}

	// Note that the sizes are rough guesstimates.
	// diskSize = 35MB; minimum size is 32MB but...
	// partition start of 2048 is ~1MB into disk.
	// partition size of 68000 is about 33.25MB.

	diskSize := uint64(35 * 1024 * 1024)
	image, err := diskfs.Create(diskFileName, int64(diskSize), diskfs.Raw, diskfs.SectorSizeDefault)
	if err != nil {
		return nil, err
	}

	table := &mbr.Table{
		LogicalSectorSize:  512,
		PhysicalSectorSize: 512,
		Partitions: []*mbr.Partition{
			{
				Bootable: false,
				Type:     mbr.Fat32LBA,
				Start:    2048,
				Size:     68000,
			},
		},
	}
	err = image.Partition(table)
	if err != nil {
		return nil, err
	}

	fs, err := image.CreateFilesystem(disk.FilesystemSpec{
		Partition:   1,
		FSType:      filesystem.TypeFat32,
		VolumeLabel: c.config.Label,
	})
	if err != nil {
		return nil, err
	}

	configPath := filepath.Dir(c.config.UserdataPath)
	if !strings.HasPrefix(configPath, "/") {
		configPath = "/" + configPath
	}
	err = fs.Mkdir(configPath)
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

	err = c.writeFile(fs, c.config.MetadataPath, metaDataContent)
	if err != nil {
		return nil, err
	}

	// The AgentEnv appears to be what goes into userdata
	userDataContent, err := c.writeAgentEnv(vmCID, agentEnv)
	if err != nil {
		return nil, err
	}

	err = c.writeFile(fs, c.config.UserdataPath, userDataContent)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(diskFileName)
}
