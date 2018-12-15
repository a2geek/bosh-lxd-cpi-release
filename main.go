package main

import (
	"os"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
	"github.com/cppforlife/bosh-cpi-go/apiv1"
	"github.com/cppforlife/bosh-cpi-go/rpc"
	lxdclient "github.com/lxc/lxd/client"
)

func main() {
	logger := boshlog.NewLogger(boshlog.LevelNone)

	cli := rpc.NewFactory(logger).NewCLI(CPIFactory{})

	err := cli.ServeOnce()
	if err != nil {
		logger.Error("main", "Serving once: %s", err)
		os.Exit(1)
	}
}

// CPIFactory implementation.
type CPIFactory struct{}

func (f CPIFactory) New(_ apiv1.CallContext) (apiv1.CPI, error) {
	c, err := lxdclient.ConnectLXDUnix("", nil)
	if err != nil {
		return nil, err
	}
	cpi := CPI{
		client:  c,
		uuidGen: boshuuid.NewGenerator(),
	}
	return cpi, nil
}

// CPI implementation
type CPI struct {
	client  lxdclient.ContainerServer
	uuidGen boshuuid.Generator
}

func (c CPI) Info() (apiv1.Info, error) {
	return apiv1.Info{StemcellFormats: []string{"warden-tar"}}, nil
}

func (c CPI) GetDisks(cid apiv1.VMCID) ([]apiv1.DiskCID, error) {
	return []apiv1.DiskCID{}, nil
}

func (c CPI) CreateDisk(size int,
	cloudProps apiv1.DiskCloudProps, associatedVMCID *apiv1.VMCID) (apiv1.DiskCID, error) {

	return apiv1.NewDiskCID("disk-cid"), nil
}

func (c CPI) DeleteDisk(cid apiv1.DiskCID) error {
	return nil
}

func (c CPI) AttachDisk(vmCID apiv1.VMCID, diskCID apiv1.DiskCID) error {
	return nil
}

func (c CPI) AttachDiskV2(vmCID apiv1.VMCID, diskCID apiv1.DiskCID) (apiv1.DiskHint, error) {
	return apiv1.NewDiskHintFromString(""), nil
}

func (c CPI) DetachDisk(vmCID apiv1.VMCID, diskCID apiv1.DiskCID) error {
	return nil
}

func (c CPI) HasDisk(cid apiv1.DiskCID) (bool, error) {
	return false, nil
}

func (c CPI) SetDiskMetadata(cid apiv1.DiskCID, metadata apiv1.DiskMeta) error {
	return nil
}

func (c CPI) ResizeDisk(cid apiv1.DiskCID, size int) error {
	return nil
}

func (c CPI) SnapshotDisk(cid apiv1.DiskCID, meta apiv1.DiskMeta) (apiv1.SnapshotCID, error) {
	return apiv1.NewSnapshotCID("snap-cid"), nil
}

func (c CPI) DeleteSnapshot(cid apiv1.SnapshotCID) error {
	return nil
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
// stemcell in CreateStemcell.
type LXDVMCloudProperties struct {
	InstanceType  string `json:"instance_type" yaml:"instance_type"`
	EphemeralDisk string `json:"ephemeral_disk" yaml:"ephemeral_disk"`
}
