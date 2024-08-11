package main

import (
	"bosh-lxd-cpi/config"
	"bosh-lxd-cpi/cpi"
	"flag"
	"os"

	"github.com/cloudfoundry/bosh-cpi-go/rpc"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

var (
	configPathOpt = flag.String("configPath", "", "Path to configuration file")
	logLevelOpt   = flag.String("logLevel", "WARN", "Set log level (NONE, ERROR, WARN, INFO, DEBUG)")
)

func main() {
	loglevel, err := boshlog.Levelify(*logLevelOpt)
	if err != nil {
		loglevel = boshlog.LevelError
	}

	logger := boshlog.NewLogger(loglevel)
	fs := boshsys.NewOsFileSystem(logger)

	flag.Parse()

	config, err := config.NewConfigFromPath(*configPathOpt, fs)
	if err != nil {
		logger.Error("main", "Loading config %s", err.Error())
		os.Exit(1)
	}

	cpiFactory := cpi.NewFactory(config, logger)

	cli := rpc.NewFactory(logger).NewCLI(cpiFactory)

	err = cli.ServeOnce()
	if err != nil {
		logger.Error("main", "Serving once: %s", err)
		os.Exit(1)
	}
}
