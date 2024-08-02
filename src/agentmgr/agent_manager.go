package agentmgr

import (
	"fmt"
	"os"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

// AgentManager is an abstraction to update the AgentEnv into a VM
type AgentManager interface {
	Update(apiv1.AgentEnv) error
	ToBytes() ([]byte, error)
}

// NewAgentManager will initialize a new config drive for AgentEnv settings
func NewAgentManager(config Config) (AgentManager, error) {
	var a AgentManager
	var err error
	switch config.SourceType {
	case "ConfigDrive":
		a, err = NewConfigDriveManager(config)
	case "CDROM":
		a, err = NewCDROMManager(config)
	}
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, fmt.Errorf("Unknown stemcell configuration type '%s'", config.SourceType)
	}
	return a, nil
}

// NewAgentManagerFromData will allow AgentEnv settings updates on an existing config drive
func NewAgentManagerFromData(config Config, data []byte) (AgentManager, error) {
	var a AgentManager
	var err error
	switch config.SourceType {
	case "ConfigDrive":
		a, err = NewConfigDriveManagerFromData(config, data)
	}
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, fmt.Errorf("Unknown stemcell configuration type '%s'", config.SourceType)
	}
	return a, nil
}

func tempFileName(ext string) (string, error) {
	pattern := fmt.Sprintf("config-*.%s", ext)

	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	name := f.Name()
	err = f.Close()
	if err != nil {
		return "", err
	}
	err = os.Remove(name)
	if err != nil {
		return "", err
	}
	return name, nil
}
