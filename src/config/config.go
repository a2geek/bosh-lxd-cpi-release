package config

import (
	"bosh-lxd-cpi/adapter"
	"bosh-lxd-cpi/agentmgr"
	"bosh-lxd-cpi/throttle"
	"encoding/json"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type Config struct {
	Server         adapter.Config
	Agent          apiv1.AgentOptions
	AgentConfig    agentmgr.Config
	ThrottleConfig throttle.Config
}

func NewConfigFromPath(path string, fs boshsys.FileSystem) (Config, error) {
	// This includes any default values
	config := Config{
		Server: adapter.Config{
			Type:               "lxd",
			URL:                "https://localhost:8443",
			InsecureSkipVerify: false,
			Project:            "default",
			Profile:            "default",
			Network:            "lxdbr0", // Default network bridge?
			StoragePool:        "default",
			BIOSPath:           "bios-256k.bin",
		},
		ThrottleConfig: throttle.Config{
			Enabled: false,
		},
	}

	bytes, err := fs.ReadFile(path)
	if err != nil {
		return config, bosherr.WrapErrorf(err, "Reading config '%s'", path)
	}

	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return config, bosherr.WrapError(err, "Unmarshalling config")
	}

	err = config.Validate()
	if err != nil {
		return config, bosherr.WrapError(err, "Validating config")
	}

	return config, nil
}

func (c Config) Validate() error {
	err := c.Server.Validate()
	if err != nil {
		return bosherr.WrapError(err, "Validating Actions configuration")
	}

	return nil
}

