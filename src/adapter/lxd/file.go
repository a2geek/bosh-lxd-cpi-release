package lxd

import (
	"bytes"
	"io"

	lxd "github.com/canonical/lxd/client"
)

func (a *lxdApiAdapter) ContainerFileRead(containerName, filePath string) (io.ReadCloser, error) {
	readCloser, _, err := a.client.GetContainerFile(containerName, filePath)
	return readCloser, err
}

func (a *lxdApiAdapter) ContainerFileWrite(containerName, filePath string, agentEnvContents []byte) error {
	args := lxd.ContainerFileArgs{
		Content:   bytes.NewReader(agentEnvContents),
		UID:       0,    // root
		GID:       0,    // root
		Mode:      0644, // rw-r--r--
		Type:      "file",
		WriteMode: "overwrite",
	}
	return a.client.CreateContainerFile(containerName, filePath, args)
}
