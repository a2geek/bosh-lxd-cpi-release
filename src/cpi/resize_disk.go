package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

func (c CPI) ResizeDisk(cid apiv1.DiskCID, size int) error {
	err := c.adapter.IsConnected()
	if err != nil {
		return err
	}

	return c.adapter.ResizeStoragePoolVolume(c.config.Server.StoragePool, cid.AsString(), size)
}
