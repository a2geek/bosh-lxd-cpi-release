package agentmgr

import (
	"bosh-lxd-cpi/adapter"
	"io"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

const AGENT_PATH = "/var/vcap/bosh/warden-cpi-agent-env.json"

func NewFileManager(adapter adapter.ApiAdapter) AgentManager {
	return fileManager{
		adapter: adapter,
	}
}

type fileManager struct {
	adapter adapter.ApiAdapter
}

func (m fileManager) Read(vmCID apiv1.VMCID) (apiv1.AgentEnv, error) {
	readCloser, err := m.adapter.ContainerFileRead(vmCID.AsString(), AGENT_PATH)
	if err != nil {
		return nil, err
	}

	bytes, err := io.ReadAll(readCloser)
	if err != nil {
		return nil, err
	}

	factory := apiv1.NewAgentEnvFactory()
	return factory.FromBytes(bytes)
}

func (m fileManager) Write(vmCID apiv1.VMCID, agentEnv apiv1.AgentEnv) ([]byte, error) {
	agentEnvContents, err := agentEnv.AsBytes()
	if err != nil {
		return nil, err
	}

	err = m.adapter.ContainerFileWrite(vmCID.AsString(), AGENT_PATH, agentEnvContents)
	// We're not creating a disk image, so no bytes are returned
	return nil, err
}

func (m fileManager) Delete(vmCID apiv1.VMCID) error {
	// N/A
	return nil
}
