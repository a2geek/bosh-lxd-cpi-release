package main

import (
	"bosh-lxd-cpi/config"
	"bosh-lxd-cpi/cpi"
	"flag"
	"os"

	lxdclient "github.com/canonical/lxd/client"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-cpi-go/rpc"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

var (
	configPathOpt = flag.String("configPath", "", "Path to configuration file")
)

func main() {
	logger := boshlog.NewLogger(boshlog.LevelNone)
	fs := boshsys.NewOsFileSystem(logger)

	flag.Parse()

	config, err := config.NewConfigFromPath(*configPathOpt, fs)
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
	config config.Config
}

func NewFactory(config config.Config) CPIFactory {
	return CPIFactory{config}
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

	return cpi.NewCPI(c, f.config), nil
}
