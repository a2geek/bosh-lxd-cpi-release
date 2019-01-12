package main

import (
	"fmt"
	"strconv"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cppforlife/bosh-cpi-go/apiv1"
	"github.com/lxc/lxd/shared/api"
)

const (
	FIXPERMISSIONS = "# Hack to fix permission issues\n" +
		"mkdir -p /var/vcap/data/sys\n" +
		"chmod 755 /var/vcap/data\n" +
		"chmod 755 /var/vcap/data/sys\n"
	MOUNTDISK = "# Manually re-mount persistent disk\n" +
		"if [ -d /warden-cpi-dev/vol-p-* ]\n" +
		"then\n" +
		"  mount --bind /warden-cpi-dev/vol-p-* /var/vcap/store\n" +
		"fi\n"
	ATTACHETH = "# Using LXD DHCP to statically assign our IP address\n" +
		"auto %s\n" +
		"iface %s inet dhcp\n"
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
			"parent":       c.config.Server.Network,
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
		"pool": c.config.Server.StoragePool,
		"path": "/",
		"size": fmt.Sprintf("%dMB", rootDeviceSize),
	}

	for name, settings := range props.Devices {
		devices[name] = settings
	}

	containersPost := api.ContainersPost{
		ContainerPut: api.ContainerPut{
			Devices:  devices,
			Config:   props.Config,
			Profiles: []string{c.config.Server.Profile},
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
		content := fmt.Sprintf(ATTACHETH, name, name)
		path := fmt.Sprintf("/etc/network/interfaces.d/%s", name)
		err = c.writeFilesAsRootToVM(vmCid, 0644 /* rw-r--r-- */, path, content)
		if err != nil {
			return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Creating network file "+path)
		}
	}

	err = c.writeFilesAsRootToVM(vmCid, 0755 /* rwxr-xr-x */, "/etc/init.d/bosh-fix-permissions.sh", FIXPERMISSIONS)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Creating firboot file")
	}

	err = c.writeFilesAsRootToVM(vmCid, 0755 /* rwxr-xr-x */, "/etc/init.d/bosh-mount-persistent.sh", MOUNTDISK)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Creating mount persistent disk file")
	}

	agentEnv := apiv1.AgentEnvFactory{}.ForVM(agentID, vmCid, networks, env, c.config.Agent)
	agentEnv.AttachSystemDisk(apiv1.NewDiskHintFromString(""))

	if props.EphemeralDisk > 0 {
		diskId, err := c.uuidGen.Generate()
		if err != nil {
			return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Creating Disk id")
		}
		diskCid := DISK_EPHEMERAL_PREFIX + diskId

		err = c.createDisk(props.EphemeralDisk, diskCid)
		if err != nil {
			return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Create ephemeral disk")
		}

		path, err := c.attachDiskDeviceToVM(vmCid, diskCid, "/var/vcap/data")
		if err != nil {
			return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Attach ephemeral disk")
		}

		agentEnv.AttachEphemeralDisk(apiv1.NewDiskHintFromMap(map[string]interface{}{"path": path}))
	}

	err = c.writeAgentFileToVM(vmCid, agentEnv)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Write AgentEnv")
	}

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
	err := c.stopVM(cid)
	if err != nil {
		return bosherr.WrapError(err, "Delete VM - stop")
	}

	disks, err := c.findEphemeralDisksAttachedToVM(cid)
	if err != nil {
		return bosherr.WrapError(err, "Delete VM - enumerate ephemeral disks")
	}

	op, err := c.client.DeleteContainer(cid.AsString())
	if err != nil {
		return bosherr.WrapError(err, "Delete VM")
	}
	err = op.Wait()
	if err != nil {
		return bosherr.WrapError(err, "Delete VM - wait")
	}

	for _, disk := range disks {
		err = c.client.DeleteStoragePoolVolume(c.config.Server.StoragePool, "custom", disk)
		if err != nil {
			return bosherr.WrapError(err, "Delete VM - attached disk - "+disk)
		}
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
	actual, err := NewActualVMMeta(metadata)
	if err != nil {
		return bosherr.WrapError(err, "Unmarshal VMMeta to ActualVMMeta")
	}

	container, etag, err := c.client.GetContainer(cid.AsString())
	if err != nil {
		return bosherr.WrapError(err, "Get container state")
	}

	description := fmt.Sprintf("%s/%s", actual.Job, actual.Index)
	container.Description = description

	op, err := c.client.UpdateContainer(cid.AsString(), container.Writable(), etag)
	if err != nil {
		return bosherr.WrapError(err, "Update container state")
	}

	err = op.Wait()
	if err != nil {
		return bosherr.WrapError(err, "Update container state - wait")
	}

	disks, err := c.findEphemeralDisksAttachedToVM(cid)
	if err != nil {
		return bosherr.WrapError(err, "Delete VM - enumerate ephemeral disks")
	}

	for _, disk := range disks {
		err = c.setDiskMetadata(apiv1.NewDiskCID(disk), description)
		if err != nil {
			return bosherr.WrapError(err, "Update storage volume description")
		}
	}

	return nil
}

func (c CPI) HasVM(cid apiv1.VMCID) (bool, error) {
	_, _, err := c.client.GetContainer(cid.AsString())
	if err != nil {
		return false, bosherr.WrapError(err, "HasVM")
	}
	return true, nil
}

func (c CPI) RebootVM(cid apiv1.VMCID) error {
	return c.setVMAction(cid, "restart")
}
