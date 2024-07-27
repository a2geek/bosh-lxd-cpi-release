package main

import (
	"bytes"
	"io"

	lxd "github.com/canonical/lxd/client"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

const AGENT_PATH = "/var/vcap/bosh/warden-cpi-agent-env.json"

func (c CPI) writeAgentFileToVM(vmCID apiv1.VMCID, agentEnv apiv1.AgentEnv) error {
	agentEnvContents, err := agentEnv.AsBytes()
	if err != nil {
		return bosherr.WrapError(err, "AgentEnv as Bytes")
	}
	agentConfigFileArgs := lxd.InstanceFileArgs{
		Content:   bytes.NewReader(agentEnvContents),
		UID:       0,    // root
		GID:       0,    // root
		Mode:      0644, // rw-r--r--
		Type:      "file",
		WriteMode: "overwrite",
	}
	err = c.client.CreateInstanceFile(vmCID.AsString(), AGENT_PATH, agentConfigFileArgs)
	if err != nil {
		return bosherr.WrapError(err, "Write AgentEnv")
	}

	return nil
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
