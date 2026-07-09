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

	// Look for the predeploy VM and delete it if it exists.
	predeployVMName := c.makeStemcellPredeployVMName(alias)
	exists, err := c.HasVM(apiv1.NewVMCID(predeployVMName))
	if err != nil {
		return nil
	}
	if exists {
		err = c.adapter.DeleteInstance(predeployVMName)
		if err != nil {
			return err
		}
	}

	return c.adapter.DeleteImage(alias)
}
