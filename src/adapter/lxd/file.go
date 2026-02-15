package lxd

import (
	"bytes"
	"io"

	lxd "github.com/canonical/lxd/client"
)

const AGENT_PATH = "/var/vcap/bosh/warden-cpi-agent-env.json"

func (a *lxdApiAdapter) ContainerFileRead(containerName string) (io.ReadCloser, error) {
	readCloser, _, err := a.client.GetContainerFile(containerName, AGENT_PATH)
	return readCloser, err
}

func (a *lxdApiAdapter) ContainerFileWrite(containerName string, agentEnvContents []byte) error {
	args := lxd.ContainerFileArgs{
		Content:   bytes.NewReader(agentEnvContents),
		UID:       0,    // root
		GID:       0,    // root
		Mode:      0644, // rw-r--r--
		Type:      "file",
		WriteMode: "overwrite",
	}
	return a.client.CreateContainerFile(containerName, AGENT_PATH, args)
}
