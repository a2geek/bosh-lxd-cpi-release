package main

import (
	"bosh-lxd-cpi/config"
	"bosh-lxd-cpi/cpi"
	"flag"
	"os"
	"strings"

	lxdclient "github.com/canonical/lxd/client"
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

	connectionArgs := &lxdclient.ConnectionArgs{
		TLSClientCert:      config.Server.TLSClientCert,
		TLSClientKey:       config.Server.TLSClientKey,
		InsecureSkipVerify: config.Server.InsecureSkipVerify,
	}
	c, err := lxdclient.ConnectLXD(config.Server.URL, connectionArgs)
	if err != nil {
		logger.Error("main", "ConnectingLXD: %s", err.Error())
		os.Exit(2)
	}

	// If a project has been specified, we use it _always_
	if len(config.Server.Project) != 0 {
		c = c.UseProject(config.Server.Project)
	}

	volumes, err := c.GetStoragePoolVolumes(config.Server.StoragePool)
	if err != nil {
		logger.Error("main", "GetStoragePoolVolumes: %s", err.Error())
		os.Exit(3)
	}

	processed := 0
	removed := 0
	failed := 0
	for _, volume := range volumes {
		if strings.HasPrefix(volume.Name, cpi.DISK_CONFIGURATION_PREFIX) && len(volume.UsedBy) == 0 {
			processed++
			if *dryRunOpt {
				logger.Info("main", "%s (skipping, dry run)", volume.Name)
			} else {
				err = c.DeleteStoragePoolVolume(config.Server.StoragePool, volume.Type, volume.Name)
				if err != nil {
					failed++
					logger.Warn("main", "Unable to delete %s: %s", volume.Name, err.Error())
				} else {
					removed++
					logger.Info("main", "Removed %s", volume.Name)
				}
			}
		}
	}

	logger.Info("main", "Complete. %d processed, %d removed, %d failed.", processed, removed, failed)

	c.Disconnect()
}
