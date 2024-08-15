package cpi

import (
	"github.com/canonical/lxd/shared/api"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c CPI) stopVM(cid apiv1.VMCID) error {
	return c.setVMAction(cid, "stop")
}

func (c CPI) startVM(cid apiv1.VMCID) error {
	return c.setVMAction(cid, "start")
}

func (c CPI) freezeVM(cid apiv1.VMCID) error {
	return c.setVMAction(cid, "freeze")
}

func (c CPI) unfreezeVM(cid apiv1.VMCID) error {
	return c.setVMAction(cid, "unfreeze")
}

func (c CPI) setVMAction(cid apiv1.VMCID, action string) error {
	atCurrentState, err := c.isVMAtRequestedState(cid, action)
	if err != nil {
		return err
	}
	if !atCurrentState {
		req := api.InstanceStatePut{
			Action:   action,
			Timeout:  30,
			Force:    true,
			Stateful: false,
		}

		err = wait(c.client.UpdateInstanceState(cid.AsString(), req, ""))
		if err != nil {
			return bosherr.WrapErrorf(err, "setVMAction(%s) - update", action)
		}
	}

	return nil
}

// isVMAtRequestedState tests if this action has already been done or completed.
func (c CPI) isVMAtRequestedState(cid apiv1.VMCID, action string) (bool, error) {
	currentState, _, err := c.client.GetInstanceState(cid.AsString())
	if err != nil {
		return false, bosherr.WrapErrorf(err, "isVMAtRequestedState(%s)", action)
	}

	atRequestedState := false

	switch action {
	case "stop":
		atRequestedState = c.hasMatch(currentState.StatusCode, api.Stopped, api.Stopping)
	case "start":
		atRequestedState = c.hasMatch(currentState.StatusCode, api.Started, api.Starting, api.Running)
	case "freeze":
		atRequestedState = c.hasMatch(currentState.StatusCode, api.Frozen, api.Freezing)
	case "unfreeze":
		atRequestedState = c.hasMatch(currentState.StatusCode, api.Running, api.Thawed)
	}

	return atRequestedState, nil
}

func (c CPI) hasMatch(actual api.StatusCode, values ...api.StatusCode) bool {
	for _, expected := range values {
		if actual == expected {
			return true
		}
	}
	return false
}
