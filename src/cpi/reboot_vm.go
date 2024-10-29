package cpi

import (
	"bosh-lxd-cpi/adapter"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

func (c CPI) RebootVM(cid apiv1.VMCID) error {
	return c.adapter.SetInstanceAction(cid.AsString(), adapter.RestartAction)
}
