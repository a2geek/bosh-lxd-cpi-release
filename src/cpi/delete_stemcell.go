package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

func (c CPI) DeleteStemcell(cid apiv1.StemcellCID) error {
	alias := cid.AsString()
	return c.adapter.DeleteStemcell(alias)
}
