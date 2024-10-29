package cpi

import (
	"bosh-lxd-cpi/adapter"
	"bosh-lxd-cpi/adapter/lxd"
	"bosh-lxd-cpi/config"
	"fmt"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

// CPIFactory implementation.
type CPIFactory struct {
	config config.Config
	logger boshlog.Logger
}

func NewFactory(config config.Config, logger boshlog.Logger) CPIFactory {
	return CPIFactory{config, logger}
}

func (f CPIFactory) New(_ apiv1.CallContext) (apiv1.CPI, error) {
	var apiAdapter adapter.ApiAdapter
	var err error
	switch f.config.Server.Type {
	case "lxd":
		apiAdapter, err = lxd.NewLXDAdapter(f.config.Server)
	default:
		err = fmt.Errorf("unknown api adapter: %s", f.config.Server.Type)
	}
	if err != nil {
		return nil, err
	}

	return NewCPI(apiAdapter, f.config, f.logger)
}
