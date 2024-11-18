package cpi

// LXDCloudProperties represents the StemcellCloudProps supplied by BOSH.
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

// LXDVMCloudProperties represents the StemcellCloudProps supplied by BOSH.
type LXDVMCloudProperties struct {
	// InstanceType as described at https://github.com/dustinkirkland/instance-type
	InstanceType string `json:"instance_type" yaml:"instance_type"`
	// EphemeralDisk sized in megabytes.
	EphemeralDisk int `json:"ephemeral_disk" yaml:"ephemeral_disk"`
}

// LXDNetworkCloudProperties represents the NetworkCloudProps supplied by BOSH.
type LXDNetworkCloudProperties struct {
	Target string `json:"target" yaml:"target"`
}
