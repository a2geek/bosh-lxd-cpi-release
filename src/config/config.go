package config

import (
	"bosh-lxd-cpi/agentmgr"
	"encoding/json"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type Config struct {
	Server      LXD
	Agent       apiv1.AgentOptions
	AgentConfig agentmgr.Config
	Throttle    ThrottleConfig
}
type LXD struct {
	URL                string
	TLSClientCert      string
	TLSClientKey       string
	InsecureSkipVerify bool
	Project            string
	Profile            string
	Network            string
	StoragePool        string
}
type ThrottleConfig struct {
	Path  string
	Limit int
	Hold  string
}

func NewConfigFromPath(path string, fs boshsys.FileSystem) (Config, error) {
	// This includes any default values
	config := Config{
		Server: LXD{
			URL:                "https://localhost:8443",
			InsecureSkipVerify: false,
			Project:            "default",
			Profile:            "default",
			Network:            "lxdbr0", // Default network bridge?
			StoragePool:        "default",
		},
		Throttle: ThrottleConfig{
			Path:  "/usr/vcap/sys/run/lxd_cpi/throttle.sock",
			Limit: 25,
			Hold:  "1m",
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

func (lxd LXD) Validate() error {
	// DO NOTHING AT THIS TIME!
	return nil
}
