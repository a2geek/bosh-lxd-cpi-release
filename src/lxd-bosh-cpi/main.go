package main

import (
	"flag"
	"os"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
	"github.com/cppforlife/bosh-cpi-go/apiv1"
	"github.com/cppforlife/bosh-cpi-go/rpc"
	lxdclient "github.com/lxc/lxd/client"
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
	if len(f.config.Profile) != 0 {
		c = c.UseProject(f.config.Project)
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
	client  lxdclient.ContainerServer
	uuidGen boshuuid.Generator
	config  Config
}

func (c CPI) Info() (apiv1.Info, error) {
	return apiv1.Info{StemcellFormats: []string{"warden-tar"}}, nil
}

// LXDCloudProperties represents the StemcellCloudProps supplied by the Bosh
// stemcell in CreateStemcell.
type LXDCloudProperties struct {
	Architecture    string `json:"architecture" yaml:"architecture"`
	ContainerFormat string `json:"container_format" yaml:"container_format"`
	Disk            int    `json:"disk" yaml:"disk"`
	DiskFormat      string `json:"disk_format" yaml:"disk_format"`
	Hypervisor      string `json:"hypervisor" yaml:"hypervisor"`
	Infrastructure  string `json:"infrastructure" yaml:"infrastructure"`
	Name            string `json:"name" yaml:"name"`
	OsDistro        string `json:"os_distro" yaml:"os_distro"`
	OsType          string `json:"os_type" yaml:"os_type"`
	RootDeviceName  string `json:"root_device_name" yaml:"root_device_name"`
	Version         string `json:"version" yaml:"version"`
}

// LXDVMCloudProperties represents the StemcellCloudProps supplied by the Bosh
// stemcell in CreateVM.
type LXDVMCloudProperties struct {
	InstanceType  string `json:"instance_type" yaml:"instance_type"`
	EphemeralDisk string `json:"ephemeral_disk" yaml:"ephemeral_disk"`
}
