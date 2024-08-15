package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

func (c CPI) DeleteStemcell(cid apiv1.StemcellCID) error {
	alias := cid.AsString()
	imageAlias, _, err := c.client.GetImageAlias(alias)
	if err != nil {
		return err
	}
	return wait(c.client.DeleteImage(imageAlias.Target))
}
