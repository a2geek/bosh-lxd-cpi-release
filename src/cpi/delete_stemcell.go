package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

func (c CPI) DeleteStemcell(cid apiv1.StemcellCID) error {
	err := c.adapter.IsConnected()
	if err != nil {
		return err
	}

	alias := cid.AsString()
	return c.adapter.DeleteImage(alias)
}
