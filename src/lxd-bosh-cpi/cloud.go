package main

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
	// InstantyType as described at https://github.com/dustinkirkland/instance-type
	InstanceType string `json:"instance_type" yaml:"instance_type"`
	// EphemeralDisk sized in megabytes.
	EphemeralDisk int `json:"ephemeral_disk" yaml:"ephemeral_disk"`
	// Devices is formatted as a map of maps. The key for the primary entry is the device name
	// and the secondary map keys are all the device properties.
	Devices map[string]map[string]string `json:"devices" yaml:"devices"`
	// Config is a map describing https://github.com/lxc/lxd/blob/master/doc/containers.md
	Config map[string]string `json:"config" yaml:"config"`
}
