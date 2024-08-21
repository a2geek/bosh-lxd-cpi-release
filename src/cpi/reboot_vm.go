package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

func (c CPI) RebootVM(cid apiv1.VMCID) error {
	return c.setVMAction(cid, "restart")
}
