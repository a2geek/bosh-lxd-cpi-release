package main

import (
	"bosh-lxd-cpi/adapter/factory"
	"bosh-lxd-cpi/config"
	"bosh-lxd-cpi/cpi"
	"flag"
	"os"
	"strings"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

var (
	configPathOpt = flag.String("configPath", "", "Path to configuration file")
	logLevelOpt   = flag.String("logLevel", "INFO", "Set log level (NONE, ERROR, WARN, INFO, DEBUG)")
	dryRunOpt     = flag.Bool("dryRun", false, "Perform a dry run")
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

	adapter, err := factory.NewAdapter(config.Server, logger)
	if err != nil {
		logger.Error("main", "API Adapter: %s", err.Error())
		os.Exit(2)
	}

	volumes, err := adapter.GetStoragePoolVolumeUsage(config.Server.StoragePool)
	if err != nil {
		logger.Error("main", "GetStoragePoolVolumeUsage: %s", err.Error())
		os.Exit(3)
	}

	processed := 0
	removed := 0
	failed := 0
	for name, count := range volumes {
		if strings.HasPrefix(name, cpi.DISK_CONFIGURATION_PREFIX) && count == 0 {
			processed++
			if *dryRunOpt {
				logger.Info("main", "%s (skipping, dry run)", name)
			} else {
				err = adapter.DeleteStoragePoolVolume(config.Server.StoragePool, name)
				if err != nil {
					failed++
					logger.Warn("main", "Unable to delete %s: %s", name, err.Error())
				} else {
					removed++
					logger.Info("main", "Removed %s", name)
				}
			}
		}
	}

	logger.Info("main", "Complete. %d processed, %d removed, %d failed.", processed, removed, failed)

	adapter.Disconnect()
}
