package incus

import (
	"bytes"
	"io"

	incus "github.com/lxc/incus/v6/client"
)

func (a *incusApiAdapter) ContainerFileRead(containerName, filePath string) (io.ReadCloser, error) {
	readCloser, _, err := a.client.GetInstanceFile(containerName, filePath)
	return readCloser, err
}

func (a *incusApiAdapter) ContainerFileWrite(containerName, filePath string, agentEnvContents []byte) error {
	args := incus.InstanceFileArgs{
		Content:   bytes.NewReader(agentEnvContents),
		UID:       0,    // root
		GID:       0,    // root
		Mode:      0644, // rw-r--r--
		Type:      "file",
		WriteMode: "overwrite",
	}
	return a.client.CreateInstanceFile(containerName, filePath, args)
}
