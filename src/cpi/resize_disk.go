package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

func (c CPI) ResizeDisk(cid apiv1.DiskCID, size int) error {
	return c.adapter.ResizeStoragePoolVolume(c.config.Server.StoragePool, cid.AsString(), size)
}
