package main

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cppforlife/bosh-cpi-go/apiv1"
	"github.com/lxc/lxd/shared/api"
)

func (c CPI) stopVM(cid apiv1.VMCID) error {
	return c.setVMAction(cid, "stop")
}

func (c CPI) startVM(cid apiv1.VMCID) error {
	return c.setVMAction(cid, "start")
}

func (c CPI) setVMAction(cid apiv1.VMCID, action string) error {
	req := api.ContainerStatePut{
		Action:   action,
		Timeout:  30,
		Force:    true,
		Stateful: false,
	}

	op, err := c.client.UpdateContainerState(cid.AsString(), req, "")
	if err != nil {
		return bosherr.WrapError(err, "Set VM Action - "+action)
	}

	err = op.Wait()
	if err != nil {
		return bosherr.WrapError(err, "Set VM Action - wait - "+action)
	}

	return nil
}
