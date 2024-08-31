package main

// This is a dirty test harness to debug the throttle client. Not used anywhere else.

import (
	"bosh-lxd-cpi/config"
	"bosh-lxd-cpi/throttle"
	"flag"
	"fmt"
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

	client, err := throttle.NewThrottleClient(config.ThrottleConfig)
	if err != nil {
		logger.Error("main", "Creating client %s", err.Error())
		os.Exit(1)
	}

	// Path 1
	transactionId, status, err := client.Lock()
	fmt.Printf("LOCK: transactionId=%s, status=%d, err=%v\n", transactionId, status, err)

	err = client.Unlock(transactionId)
	fmt.Printf("UNLOCK: transactionId=%s, err=%v\n", transactionId, err)

	// Path 2
	transactionId, err = client.LockAndWait()
	fmt.Printf("LOCK-AND-WAIT: transactionId=%s, err=%v\n", transactionId, err)

	err = client.Unlock(transactionId)
	fmt.Printf("UNLOCK: transactionId=%s, err=%v\n", transactionId, err)

}
