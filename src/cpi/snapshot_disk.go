package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

func (c CPI) SnapshotDisk(cid apiv1.DiskCID, meta apiv1.DiskMeta) (apiv1.SnapshotCID, error) {
	return apiv1.NewSnapshotCID("snap-cid"), nil
}
