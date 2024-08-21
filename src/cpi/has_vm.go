package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c CPI) HasVM(cid apiv1.VMCID) (bool, error) {
	_, _, err := c.client.GetInstance(cid.AsString())
	if err != nil {
		return false, bosherr.WrapError(err, "HasVM")
	}
	return true, nil
}
