package cpi

import (
	"bosh-lxd-cpi/config"

	lxdclient "github.com/canonical/lxd/client"
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
	connectionArgs := &lxdclient.ConnectionArgs{
		TLSClientCert:      f.config.Server.TLSClientCert,
		TLSClientKey:       f.config.Server.TLSClientKey,
		InsecureSkipVerify: f.config.Server.InsecureSkipVerify,
	}
	c, err := lxdclient.ConnectLXD(f.config.Server.URL, connectionArgs)
	if err != nil {
		return nil, err
	}

	// If a project has been specified, we use it _always_
	if len(f.config.Server.Project) != 0 {
		c = c.UseProject(f.config.Server.Project)
	}

	return NewCPI(c, f.config, f.logger)
}
