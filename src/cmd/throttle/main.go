package main

import (
	"bosh-lxd-cpi/config"
	"bosh-lxd-cpi/throttle"
	"flag"
	"log"
	"os"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

var (
	configPathOpt = flag.String("configPath", "", "Path to configuration file")
	logLevelOpt   = flag.String("logLevel", "WARN", "Set log level (NONE, ERROR, WARN, INFO, DEBUG)")
)

func main() {
	flag.Parse()

	loglevel, err := boshlog.Levelify(*logLevelOpt)
	if err != nil {
		loglevel = boshlog.LevelError
	}

	logger := boshlog.NewLogger(loglevel)
	fs := boshsys.NewOsFileSystem(logger)

	config, err := config.NewConfigFromPath(*configPathOpt, fs)
	if err != nil {
		logger.Error("main", "Loading config %s", err.Error())
		os.Exit(1)
	}

	server, err := throttle.NewThrottleServer(config.ThrottleConfig, logger)
	if err != nil {
		logger.Error("main", "Unable to create server: %s", err.Error())
		os.Exit(2)
	}

	logger.Info("main", "Now serving traffic on socket %s", config.ThrottleConfig.Path)
	log.Fatal(server.Serve())
}
