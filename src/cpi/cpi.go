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
	adapter           adapter.ApiAdapter
	uuidGen           boshuuid.Generator
	config            config.Config
	logger            boshlog.Logger
	agentMgrVM        agentmgr.AgentManager
	agentMgrContainer agentmgr.AgentManager
}

func NewCPI(adapter adapter.ApiAdapter, cfg config.Config, logger boshlog.Logger) (CPI, error) {
	am, err := agentmgr.NewAgentManager(cfg.AgentConfig)
	if err != nil {
		return CPI{}, err
	}
	cpi := CPI{
		adapter:           adapter,
		uuidGen:           boshuuid.NewGenerator(),
		config:            cfg,
		logger:            logger,
		agentMgrVM:        am,
		agentMgrContainer: agentmgr.NewFileManager(adapter),
	}
	return cpi, err
}
