package cpi

import (
	"bosh-lxd-cpi/adapter"
	"bosh-lxd-cpi/agentmgr"
	"bosh-lxd-cpi/config"
	"time"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
	"github.com/gofrs/flock"
)

// CPI implementation
type CPI struct {
	adapter             adapter.ApiAdapter
	uuidGen             boshuuid.Generator
	config              config.Config
	logger              boshlog.Logger
	agentMgr            agentmgr.AgentManager
	createVMLock        *flock.Flock
	createVMLockTimeout time.Duration
}

func NewCPI(adapter adapter.ApiAdapter, cfg config.Config, logger boshlog.Logger) (CPI, error) {
	am, err := agentmgr.NewAgentManager(cfg.AgentConfig)
	if err != nil {
		return CPI{}, err
	}
	createVMLockTimeout, err := time.ParseDuration(cfg.Server.CreateVMLock.Timeout)
	if err != nil {
		return CPI{}, err
	}
	cpi := CPI{
		adapter:             adapter,
		uuidGen:             boshuuid.NewGenerator(),
		config:              cfg,
		logger:              logger,
		agentMgr:            am,
		createVMLock:        flock.New(cfg.Server.CreateVMLock.Path),
		createVMLockTimeout: createVMLockTimeout,
	}
	return cpi, err
}
