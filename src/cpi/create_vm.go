package cpi

import (
	"bosh-lxd-cpi/adapter"
	"fmt"

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
		if c.config.Server.Managed {
			settings["ipv4.address"] = net.IP()
			// Hypothesis: manual (we assign IP) but map over to dynamic so the bosh agent just lets LXD fix the IP
			newNet := apiv1.NewNetwork(apiv1.NetworkOpts{
				Type:    "dynamic",
				IP:      net.IP(),
				Netmask: net.Netmask(),
				Gateway: net.Gateway(),
				DNS:     net.DNS(),
				Default: net.Default(),
			})
			newNetworks[key] = newNet
		}
		devices[name] = settings

		eth++
	}

	devices["root"] = map[string]string{
		"type": "disk",
		"pool": c.config.Server.StoragePool,
		"path": "/",
	}

	err = c.adapter.CreateInstance(adapter.InstanceMetadata{
		Name:          theCid,
		StemcellAlias: stemcellCID.AsString(),
		InstanceType:  vmProps.InstanceType,
		Project:       c.config.Server.Project,
		Profiles:      []string{c.config.Server.Profile},
		Target:        vmProps.Target,
		Devices:       devices,
		Config: map[string]string{
			"raw.qemu": "-bios " + c.config.Server.BIOSPath,
		},
	})
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Creating VM")
	}

	defer func() {
		if err != nil {
			c.DeleteVM(vmCID)
		}
	}()

	target, err := c.adapter.GetInstanceLocation(theCid)
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

		err = c.adapter.CreateStoragePoolVolume(target, c.config.Server.StoragePool, diskCid, vmProps.EphemeralDisk)
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

	err = c.adapter.SetInstanceAction(vmCID.AsString(), adapter.StartAction)
	return vmCID, newNetworks, err
}
