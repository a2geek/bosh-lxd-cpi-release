package cpi

import (
	"bosh-lxd-cpi/adapter/factory"
	"bosh-lxd-cpi/config"

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
	apiAdapter, err := factory.NewAdapter(f.config.Server)
	if err != nil {
		return nil, err
	}

	return NewCPI(apiAdapter, f.config, f.logger)
}
