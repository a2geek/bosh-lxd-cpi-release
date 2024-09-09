package cpi

import (
	"fmt"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

func (c CPI) ResizeDisk(cid apiv1.DiskCID, size int) error {
	volume, etag, err := c.client.GetStoragePoolVolume(c.config.Server.StoragePool, "custom", cid.AsString())
	if err != nil {
		return err
	}

	writable := volume.Writable()
	writable.Config["size"] = fmt.Sprintf("%dMiB", size)

	return c.client.UpdateStoragePoolVolume(c.config.Server.StoragePool, "custom", cid.AsString(), writable, etag)
}
