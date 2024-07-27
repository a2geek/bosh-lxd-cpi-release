package main

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

		op, err := c.client.UpdateInstanceState(cid.AsString(), req, "")
		if err != nil {
			return bosherr.WrapError(err, "Set VM Action - update - "+action)
		}

		err = op.Wait()
		if err != nil {
			return bosherr.WrapError(err, "Set VM Action - wait - "+action)
		}
	}

	return nil
}

// checkVMAction tests if this action has already been done or completed.
func (c CPI) isVMAtRequestedState(cid apiv1.VMCID, action string) (bool, error) {
	currentState, _, err := c.client.GetInstanceState(cid.AsString())
	if err != nil {
		return false, bosherr.WrapError(err, "Check VM Action - "+action)
	}

	atRequestedState := false

	switch action {
	case "stop":
		atRequestedState = currentState.StatusCode == api.Stopped || currentState.StatusCode == api.Stopping
	case "start":
		atRequestedState = currentState.StatusCode == api.Started || currentState.StatusCode == api.Starting
	}

	return atRequestedState, nil
}
