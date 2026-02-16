package cpi

import (
	"bosh-lxd-cpi/adapter"
	"fmt"
	"strings"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c CPI) CreateVM(
	agentID apiv1.AgentID, stemcellCID apiv1.StemcellCID,
	cloudProps apiv1.VMCloudProps, networks apiv1.Networks,
	associatedDiskCIDs []apiv1.DiskCID, env apiv1.VMEnv) (apiv1.VMCID, error) {

	vmCID, _, err := c.CreateVMV2(agentID, stemcellCID, cloudProps, networks, associatedDiskCIDs, env)
	return vmCID, err
}

func (c CPI) CreateVMV2(
	agentID apiv1.AgentID, stemcellCID apiv1.StemcellCID,
	cloudProps apiv1.VMCloudProps, networks apiv1.Networks,
	associatedDiskCIDs []apiv1.DiskCID, env apiv1.VMEnv) (apiv1.VMCID, apiv1.Networks, error) {

	id, err := c.uuidGen.Generate()
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Creating VM id")
	}
	theCid := "vm-" + id
	vmCID := apiv1.NewVMCID(theCid)

	// Default to global configuration
	vmProps := LXDVMCloudProperties{
		Target:  c.config.Server.Target,
		Network: c.config.Server.Network,
	}
	err = cloudProps.As(&vmProps)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Cloud Props")
	}

	managedNetwork, err := c.adapter.IsManagedNetwork(vmProps.Network)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Managed Network")
	}

	devices := make(map[string]map[string]string)
	eth := 0
	newNetworks := apiv1.Networks{}
	for key, net := range networks {
		name := fmt.Sprintf("eth%d", eth)
		settings := map[string]string{
			"name":    name,
			"nictype": "bridged",
			"parent":  vmProps.Network,
			"type":    "nic",
		}
		newNetworks[key] = net
		if managedNetwork {
			// Capture the "planned" IP -- but don't use if we have a BOSH dynamic network
			// We skip BOSH dynamic networks because those shouldn't assign (and can assign wrong IPs - specifically the Ubuntu FAN)
			plannedIP := net.IP()
			if net.IsDynamic() {
				plannedIP = ""
			}
			// If we have a managed network, let's ensure the expected IP address gets set
			if plannedIP != "" {
				settings["ipv4.address"] = plannedIP
			}
			if c.config.Server.ManagedNetworkAssignment == "dhcp" {
				// Remap network to be a BOSH 'dynamic' network so BOSH/Agent reports the DHCP assigned IP
				newNet := apiv1.NewNetwork(apiv1.NetworkOpts{
					Type:    "dynamic",
					IP:      plannedIP,
					Netmask: net.Netmask(),
					Gateway: net.Gateway(),
					DNS:     net.DNS(),
					Default: net.Default(),
				})
				newNetworks[key] = newNet
			}
		}
		devices[name] = settings

		eth++
	}

	devices["root"] = map[string]string{
		"type": "disk",
		"pool": c.config.Server.StoragePool,
		"path": "/",
	}

	// Figure out which stemcell line this is and apply corresponding configuration
	imageInfo, err := c.adapter.GetStemcellInfo(stemcellCID.AsString())
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapErrorf(err, "find stemcell description for '%s'", stemcellCID.AsString())
	}
	var instanceConfig map[string]string
	var defaultConfig map[string]string
	for k, v := range c.config.Server.InstanceConfig {
		if "default" == k {
			defaultConfig = v
		} else if strings.Contains(imageInfo.Description, k) {
			instanceConfig = v
		}
	}
	if len(instanceConfig) == 0 {
		instanceConfig = defaultConfig
	}
	if len(instanceConfig) == 0 {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.Error("no matching stemcell configuration found")
	}

	contentType := adapter.BlockContent
	if imageInfo.Type == adapter.InstanceContainer {
		contentType = adapter.FilesystemContent
		// HACK! Not certain what to do at this point
		instanceConfig = map[string]string{}
		instanceConfig["security.privileged"] = "true"
		instanceConfig["security.nesting"] = "true"
		instanceConfig["raw.lxc"] = "lxc.mount.auto = proc:rw sys:rw cgroup:rw\nlxc.apparmor.profile = unconfined\nlxc.cap.drop ="
	}

	err = c.adapter.CreateInstance(adapter.InstanceMetadata{
		Name:          theCid,
		StemcellAlias: stemcellCID.AsString(),
		Type:          imageInfo.Type,
		InstanceType:  vmProps.InstanceType,
		Project:       c.config.Server.Project,
		Profiles:      []string{c.config.Server.Profile},
		Target:        vmProps.Target,
		Devices:       devices,
		Config:        instanceConfig,
	})
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Creating VM")
	}

	defer func() {
		if err != nil {
			c.DeleteVM(vmCID)
		}
	}()

	target, err := c.adapter.GetInstanceInfo(theCid)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "VM Location")
	}
	c.logger.Debug("create_vm", "locating disks at '%s' after creation", target)

	agentEnv := apiv1.AgentEnvFactory{}.ForVM(agentID, vmCID, newNetworks, env, c.config.Agent)
	agentEnv.AttachSystemDisk(apiv1.NewDiskHintFromString("/dev/sda"))

	if vmProps.EphemeralDisk > 0 {
		diskId, err := c.uuidGen.Generate()
		if err != nil {
			return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Creating Disk id")
		}
		diskCid := DISK_EPHEMERAL_PREFIX + diskId

		err = c.adapter.CreateStoragePoolVolume(target.Location, c.config.Server.StoragePool, diskCid, contentType, vmProps.EphemeralDisk)
		if err != nil {
			return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Create ephemeral disk")
		}

		err = c.attachDiskDeviceToVM(vmCID, DISK_DEVICE_EPHEMERAL, diskCid)
		if err != nil {
			return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Attach ephemeral disk")
		}

		agentEnv.AttachEphemeralDisk(apiv1.NewDiskHintFromString("/dev/sdb"))
	}

	err = c.writeAgentFileToVM(vmCID, agentEnv)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Write AgentEnv")
	}

	if target.Type == adapter.InstanceContainer {
		for eth := 0; eth < len(networks); eth++ {
			ethN := fmt.Sprintf("# Using DHCP to statically assign our IP address\n"+
				"auto eth%d\n"+
				"iface eth%d inet dhcp\n", eth, eth)
			fileName := fmt.Sprintf("/etc/network/interfaces.d/eth%d", eth)
			c.adapter.ContainerFileWrite(vmCID.AsString(), fileName, []byte(ethN))
		}
	}

	err = c.adapter.SetInstanceAction(vmCID.AsString(), adapter.StartAction)
	return vmCID, newNetworks, err
}
