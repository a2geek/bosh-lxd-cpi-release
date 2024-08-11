package agentmgr

import (
	"fmt"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

// AgentManager is an abstraction to update the AgentEnv into a VM
type AgentManager interface {
	Read(apiv1.VMCID) (apiv1.AgentEnv, error)
	Write(apiv1.VMCID, apiv1.AgentEnv) ([]byte, error)
}

// NewAgentManager will initialize a new config drive for AgentEnv settings
func NewAgentManager(config Config) (AgentManager, error) {
	var a AgentManager
	switch config.SourceType {
	case "FAT32":
		a = NewFAT32Manager(config)
	case "CDROM":
		a = NewCDROMManager(config)
	}
	if a == nil {
		return nil, fmt.Errorf("unknown stemcell configuration type '%s'", config.SourceType)
	}
	return a, nil
}
