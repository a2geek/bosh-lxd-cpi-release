package cpi

import (
	"bosh-lxd-cpi/adapter"
	"bosh-lxd-cpi/agentmgr"
	"bosh-lxd-cpi/config"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
)

// CPI implementation
type CPI struct {
	adapter  adapter.ApiAdapter
	uuidGen  boshuuid.Generator
	config   config.Config
	logger   boshlog.Logger
	agentMgr agentmgr.AgentManager
}

func NewCPI(adapter adapter.ApiAdapter, cfg config.Config, logger boshlog.Logger) (CPI, error) {
	agentMgrVM, err := agentmgr.NewAgentManager(cfg.AgentConfig)
	if err != nil {
		return CPI{}, err
	}

	agentMgrContainer := agentmgr.NewContainerFileManager(adapter)

	agentMgr := agentmgr.NewSwitchManager(adapter, agentMgrVM, agentMgrContainer)

	cpi := CPI{
		adapter:  adapter,
		uuidGen:  boshuuid.NewGenerator(),
		config:   cfg,
		logger:   logger,
		agentMgr: agentMgr,
	}
	return cpi, err
}
