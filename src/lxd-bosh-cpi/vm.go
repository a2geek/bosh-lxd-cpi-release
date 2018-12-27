package main

import (
	"bytes"
	"fmt"
	"strings"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cppforlife/bosh-cpi-go/apiv1"
	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
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
	theCid := "c-" + id
	vmCid := apiv1.NewVMCID(theCid)

	containerSource := api.ContainerSource{
		Type:  "image",
		Alias: stemcellCID.AsString(),
	}
	props := LXDVMCloudProperties{}
	err = cloudProps.As(&props)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Cloud Props")
	}
	containersPost := api.ContainersPost{
		ContainerPut: api.ContainerPut{
			Devices: map[string]map[string]string{
				"eth0": map[string]string{
					"name":         "eth0",
					"nictype":      "bridged",
					"parent":       c.config.Network,
					"type":         "nic",
					"ipv4.address": networks["default"].IP(),
				},
			},
			Profiles:    []string{c.config.Profile},
			Description: "hello world",
		},
		Name:         theCid,
		InstanceType: props.InstanceType,
		Source:       containerSource,
	}
	op, err := c.client.CreateContainer(containersPost)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Creating VM")
	}
	err = op.Wait()
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Creating VM")
	}

	_, etag, err := c.client.GetContainerState(theCid)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Retrieve state of VM")
	}

	// Write the eth0 file for auto configuration. This is likely a bug waiting to happen. :-(
	eth0FileArgs := lxd.ContainerFileArgs{
		Content:   strings.NewReader("# Using LXD DHCP to statically assign our IP address\nauto eth0\niface eth0 inet dhcp\n"),
		UID:       0,    // root
		GID:       0,    // root
		Mode:      0644, // rw-r--r--
		Type:      "file",
		WriteMode: "overwrite",
	}
	c.client.CreateContainerFile(theCid, "/etc/network/interfaces.d/eth0", eth0FileArgs)

	agentEnv := apiv1.AgentEnvFactory{}.ForVM(agentID, vmCid, networks, env, c.config.Agent)
	//agentEnv.AttachSystemDisk("")
	agentEnvContents, err := agentEnv.AsBytes()
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "AgentEnv as Bytes")
	}
	agentConfigFileArgs := lxd.ContainerFileArgs{
		Content:   bytes.NewReader(agentEnvContents),
		UID:       0,    // root
		GID:       0,    // root
		Mode:      0644, // rw-r--r--
		Type:      "file",
		WriteMode: "overwrite",
	}
	c.client.CreateContainerFile(theCid, "/var/vcap/bosh/warden-cpi-agent-env.json", agentConfigFileArgs)

	containerStatePut := api.ContainerStatePut{
		Action: "start",
	}
	op, err = c.client.UpdateContainerState(theCid, containerStatePut, etag)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Update state of VM")
	}
	// Don't have to wait

	return vmCid, networks, nil
}

func (c CPI) DeleteVM(cid apiv1.VMCID) error {
	return nil
}

func (c CPI) CalculateVMCloudProperties(res apiv1.VMResources) (apiv1.VMCloudProps, error) {
	props := make(map[string]interface{})
	props["instance_type"] = fmt.Sprintf("c%d-m%d", res.CPU, res.RAM/1024)
	props["ephemeral_disk"] = res.EphemeralDiskSize
	return apiv1.NewVMCloudPropsFromMap(props), nil
}

func (c CPI) SetVMMetadata(cid apiv1.VMCID, metadata apiv1.VMMeta) error {
	return nil
}

func (c CPI) HasVM(cid apiv1.VMCID) (bool, error) {
	_, _, err := c.client.GetContainer(cid.AsString())
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (c CPI) RebootVM(cid apiv1.VMCID) error {
	req := api.ContainerStatePut{
		Action:   "restart",
		Timeout:  30,
		Force:    true,
		Stateful: false,
	}

	op, err := c.client.UpdateContainerState(cid.AsString(), req, "")
	if err != nil {
		return bosherr.WrapError(err, "reboot vm")
	}

	err = op.Wait()
	if err != nil {
		return bosherr.WrapError(err, "reboot vm")
	}

	return nil
}
