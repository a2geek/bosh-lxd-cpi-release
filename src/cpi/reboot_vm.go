package cpi

import (
	"bosh-lxd-cpi/adapter"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

func (c CPI) RebootVM(cid apiv1.VMCID) error {
	err := c.adapter.IsConnected()
	if err != nil {
		return err
	}

	return c.adapter.SetInstanceAction(cid.AsString(), adapter.RestartAction)
}
