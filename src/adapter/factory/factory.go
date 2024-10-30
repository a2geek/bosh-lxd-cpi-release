package factory

import (
	"bosh-lxd-cpi/adapter"
	"bosh-lxd-cpi/adapter/incus"
	"bosh-lxd-cpi/adapter/lxd"
	"fmt"
)

func NewAdapter(config adapter.Config) (adapter.ApiAdapter, error) {
	switch config.Type {
	case "lxd":
		return lxd.NewLXDAdapter(config)
	case "incus":
		return incus.NewIncusAdapter(config)
	}
	return nil, fmt.Errorf("unknown adapter type: '%s'", config.Type)
}
