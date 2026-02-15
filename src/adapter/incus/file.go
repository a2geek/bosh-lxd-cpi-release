package incus

import (
	"bytes"
	"io"

	incus "github.com/lxc/incus/v6/client"
)

const AGENT_PATH = "/var/vcap/bosh/warden-cpi-agent-env.json"

func (a *incusApiAdapter) ContainerFileRead(containerName string) (io.ReadCloser, error) {
	readCloser, _, err := a.client.GetInstanceFile(containerName, AGENT_PATH)
	return readCloser, err
}

func (a *incusApiAdapter) ContainerFileWrite(containerName string, agentEnvContents []byte) error {
	args := incus.InstanceFileArgs{
		Content:   bytes.NewReader(agentEnvContents),
		UID:       0,    // root
		GID:       0,    // root
		Mode:      0644, // rw-r--r--
		Type:      "file",
		WriteMode: "overwrite",
	}
	return a.client.CreateInstanceFile(containerName, AGENT_PATH, args)
}
