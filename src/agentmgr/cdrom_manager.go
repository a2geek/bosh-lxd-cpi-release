package agentmgr

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
)

func NewCDROMManager(config Config) (AgentManager, error) {
	name, err := tempFileName("iso")
	if err != nil {
		return nil, err
	}
	mgr := cdromManager{
		diskFileName: name,
		config:       config,
	}
	return mgr, nil
}

type cdromManager struct {
	diskFileName string
	config       Config
}

func (m cdromManager) Update(agentEnv apiv1.AgentEnv) error {
	// Based on sample: https://github.com/diskfs/go-diskfs/blob/master/examples/iso_create.go

	var diskSize int64 = 5 * 1024 * 1024 // 5 MB
	image, err := diskfs.Create(m.diskFileName, diskSize, diskfs.Raw, diskfs.SectorSizeDefault)
	if err != nil {
		return err
	}

	image.LogicalBlocksize = 2048
	fspec := disk.FilesystemSpec{
		Partition:   0,
		FSType:      filesystem.TypeISO9660,
		VolumeLabel: m.config.Label,
	}
	fs, err := image.CreateFilesystem(fspec)
	if err != nil {
		return err
	}

	// The AgentEnv goes into userdata
	buf, err := agentEnv.AsBytes()
	if err != nil {
		return err
	}

	err = m.writeFile(fs, m.config.UserdataPath, buf)
	if err != nil {
		return err
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
		return err
	}

	err = m.writeFile(fs, m.config.MetadataPath, metaDataContent)
	if err != nil {
		return err
	}

	iso, ok := fs.(*iso9660.FileSystem)
	if !ok {
		return fmt.Errorf("not an iso9660 filesystem")
	}
	return iso.Finalize(iso9660.FinalizeOptions{})
}

func (m cdromManager) ToBytes() ([]byte, error) {
	return os.ReadFile(m.diskFileName)
}

func (m cdromManager) writeFile(fs filesystem.FileSystem, path string, contents []byte) error {
	if path == "" {
		return nil
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	rw, err := fs.OpenFile(path, os.O_CREATE|os.O_RDWR)
	if err != nil {
		return err
	}

	_, err = rw.Write(contents)
	if err != nil {
		return err
	}

	return nil
}
