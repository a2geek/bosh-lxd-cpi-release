package cpi

import (
	"bosh-lxd-cpi/agentmgr"
	"bosh-lxd-cpi/config"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"

	lxdclient "github.com/canonical/lxd/client"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
)

// CPI implementation
type CPI struct {
	client   lxdclient.InstanceServer
	uuidGen  boshuuid.Generator
	config   config.Config
	logger   boshlog.Logger
	agentMgr agentmgr.AgentManager
}

func NewCPI(client lxdclient.InstanceServer, cfg config.Config, logger boshlog.Logger) (CPI, error) {
	am, err := agentmgr.NewAgentManager(cfg.AgentConfig)
	if err != nil {
		return CPI{}, err
	}
	cpi := CPI{
		client:   client,
		uuidGen:  boshuuid.NewGenerator(),
		config:   cfg,
		logger:   logger,
		agentMgr: am,
	}
	return cpi, err
}

func (c CPI) Info() (apiv1.Info, error) {
	return apiv1.Info{StemcellFormats: []string{"openstack-qcow2"}}, nil
}
