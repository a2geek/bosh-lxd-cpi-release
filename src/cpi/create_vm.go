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

	vmProps := LXDVMCloudProperties{}
	err = cloudProps.As(&vmProps)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Cloud Props")
	}

	devices := make(map[string]map[string]string)
	eth := 0
	target := ""
	for _, net := range networks {
		name := fmt.Sprintf("eth%d", eth)
		devices[name] = map[string]string{
			"name":         name,
			"nictype":      "bridged",
			"parent":       c.config.Server.Network,
			"type":         "nic",
			"ipv4.address": net.IP(),
		}

		networkProps := LXDNetworkCloudProperties{}
		err = net.CloudProps().As(networkProps)
		if err != nil {
			return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Network Props")
		}
		target = networkProps.Target

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
		Target:        target,
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

	agentEnv := apiv1.AgentEnvFactory{}.ForVM(agentID, vmCID, networks, env, c.config.Agent)
	agentEnv.AttachSystemDisk(apiv1.NewDiskHintFromString("/dev/sda"))

	if vmProps.EphemeralDisk > 0 {
		diskId, err := c.uuidGen.Generate()
		if err != nil {
			return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Creating Disk id")
		}
		diskCid := DISK_EPHEMERAL_PREFIX + diskId

		err = c.adapter.CreateStoragePoolVolume(c.config.Server.StoragePool, diskCid, vmProps.EphemeralDisk)
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
	return vmCID, networks, err
}
