package main

import (
	"bytes"
	"fmt"
	"strconv"
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

	devices := make(map[string]map[string]string)
	eth := 0
	for _, net := range networks {
		net.SetPreconfigured()
		name := fmt.Sprintf("eth%d", eth)
		devices[name] = map[string]string{
			"name":         name,
			"nictype":      "bridged",
			"parent":       c.config.Network,
			"type":         "nic",
			"ipv4.address": net.IP(),
		}
		eth++
	}

	// Add root device
	imageAlias, _, err := c.client.GetImageAlias(containerSource.Alias)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Image Alias locate")
	}
	image, _, err := c.client.GetImage(imageAlias.Target)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Image retrieval")
	}
	rootDeviceSize, err := strconv.Atoi(image.Properties["root_disk_size"])
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Root device size not determined")
	}
	devices["root"] = map[string]string{
		"type": "disk",
		"pool": "default",
		"path": "/",
		"size": fmt.Sprintf("%dMB", rootDeviceSize),
	}

	containersPost := api.ContainersPost{
		ContainerPut: api.ContainerPut{
			Devices: devices,
			Config: map[string]string{
				"security.privileged": "true",
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
	for name, device := range devices {
		if device["type"] != "nic" {
			continue
		}
		template := "# Using LXD DHCP to statically assign our IP address\nauto %s\niface %s inet dhcp\n"
		content := fmt.Sprintf(template, name, name)
		fileArgs := lxd.ContainerFileArgs{
			Content:   strings.NewReader(content),
			UID:       0,    // root
			GID:       0,    // root
			Mode:      0644, // rw-r--r--
			Type:      "file",
			WriteMode: "overwrite",
		}
		path := fmt.Sprintf("/etc/network/interfaces.d/%s", name)
		c.client.CreateContainerFile(theCid, path, fileArgs)
	}

	agentEnv := apiv1.AgentEnvFactory{}.ForVM(agentID, vmCid, networks, env, c.config.Agent)
	agentEnv.AttachSystemDisk(apiv1.NewDiskHintFromString(""))
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
	req := api.ContainerStatePut{
		Action:   "stop",
		Timeout:  30,
		Force:    true,
		Stateful: false,
	}

	op, err := c.client.UpdateContainerState(cid.AsString(), req, "")
	if err != nil {
		return bosherr.WrapError(err, "stop vm")
	}

	err = op.Wait()
	if err != nil {
		return bosherr.WrapError(err, "stop vm2")
	}

	op, err = c.client.DeleteContainer(cid.AsString())
	if err != nil {
		return bosherr.WrapError(err, "Delete VM")
	}
	err = op.Wait()
	if err != nil {
		return bosherr.WrapError(err, "Delete VM2")
	}

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
