package factory

import (
	"bosh-lxd-cpi/adapter"
	"bosh-lxd-cpi/adapter/incus"
	"bosh-lxd-cpi/adapter/lxd"
	"fmt"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

func NewAdapter(config adapter.Config, logger boshlog.Logger) (adapter.ApiAdapter, error) {
	switch config.Type {
	case "lxd":
		return lxd.NewLXDAdapter(config, logger)
	case "incus":
		return incus.NewIncusAdapter(config, logger)
	}
	return nil, fmt.Errorf("unknown adapter type: '%s'", config.Type)
}
