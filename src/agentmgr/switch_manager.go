package agentmgr

import (
	"bosh-lxd-cpi/adapter"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

// The "switch" manager switches AgentManager based on the instance being a VM or a Container
func NewSwitchManager(adapter adapter.ApiAdapter, vmManager, containerManager AgentManager) AgentManager {
	return switchManager{
		adapter:          adapter,
		vmManager:        vmManager,
		containerManager: containerManager,
	}
}

type switchManager struct {
	adapter          adapter.ApiAdapter
	vmManager        AgentManager
	containerManager AgentManager
}

func (m switchManager) IsContainer(vmCID apiv1.VMCID) (bool, error) {
	info, err := m.adapter.GetInstanceInfo(vmCID.AsString())
	if err != nil {
		return false, err
	}

	return info.Type == adapter.InstanceContainer, nil
}

func (m switchManager) Read(vmCID apiv1.VMCID) (apiv1.AgentEnv, error) {
	isContainer, err := m.IsContainer(vmCID)
	if err != nil {
		return nil, err
	}

	if isContainer {
		return m.containerManager.Read(vmCID)
	} else {
		return m.vmManager.Read(vmCID)
	}
}

func (m switchManager) Write(vmCID apiv1.VMCID, agentEnv apiv1.AgentEnv) ([]byte, error) {
	isContainer, err := m.IsContainer(vmCID)
	if err != nil {
		return nil, err
	}

	if isContainer {
		return m.containerManager.Write(vmCID, agentEnv)
	} else {
		return m.vmManager.Write(vmCID, agentEnv)
	}
}

func (m switchManager) Delete(vmCID apiv1.VMCID) error {
	isContainer, err := m.IsContainer(vmCID)
	if err != nil {
		return err
	}

	if isContainer {
		return m.containerManager.Delete(vmCID)
	} else {
		return m.vmManager.Delete(vmCID)
	}
}
