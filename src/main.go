package main

import (
	"flag"
	"os"

	lxdclient "github.com/canonical/lxd/client"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-cpi-go/rpc"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
)

var (
	configPathOpt = flag.String("configPath", "", "Path to configuration file")
)

func main() {
	logger := boshlog.NewLogger(boshlog.LevelNone)
	fs := boshsys.NewOsFileSystem(logger)

	flag.Parse()

	config, err := NewConfigFromPath(*configPathOpt, fs)
	if err != nil {
		logger.Error("main", "Loading config %s", err.Error())
		os.Exit(1)
	}

	cpiFactory := NewFactory(config)

	cli := rpc.NewFactory(logger).NewCLI(cpiFactory)

	err = cli.ServeOnce()
	if err != nil {
		logger.Error("main", "Serving once: %s", err)
		os.Exit(1)
	}
}

// CPIFactory implementation.
type CPIFactory struct {
	config Config
}

func NewFactory(config Config) CPIFactory {
	return CPIFactory{config}
}

func (f CPIFactory) New(_ apiv1.CallContext) (apiv1.CPI, error) {
	c, err := lxdclient.ConnectLXDUnix(f.config.Server.Socket, nil)
	if err != nil {
		return nil, err
	}

	// If a project has been specified, we use it _always_
	if len(f.config.Server.Project) != 0 {
		c = c.UseProject(f.config.Server.Project)
	}

	cpi := CPI{
		client:  c,
		uuidGen: boshuuid.NewGenerator(),
		config:  f.config,
	}
	return cpi, nil
}

// CPI implementation
type CPI struct {
	client  lxdclient.InstanceServer
	uuidGen boshuuid.Generator
	config  Config
}

func (c CPI) Info() (apiv1.Info, error) {
	return apiv1.Info{StemcellFormats: []string{"warden-tar"}}, nil
}
