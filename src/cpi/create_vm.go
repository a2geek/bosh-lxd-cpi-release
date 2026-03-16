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

	c.logger.Debug("create_vm", "Starting CreateVMV2 for agent '%s' with stemcell '%s'", agentID.AsString(), stemcellCID.AsString())

	// Track ephemeral disk for cleanup on failure
	var ephemeralDiskCid string

	id, err := c.uuidGen.Generate()
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Creating VM id")
	}
	theCid := "vm-" + id
	vmCID := apiv1.NewVMCID(theCid)
	c.logger.Debug("create_vm", "Generated VM CID: '%s'", theCid)

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
	imageDescription, err := c.adapter.GetStemcellDescription(stemcellCID.AsString())
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapErrorf(err, "find stemcell description for '%s'", stemcellCID.AsString())
	}
	var instanceConfig map[string]string
	var defaultConfig map[string]string
	for k, v := range c.config.Server.InstanceConfig {
		if "default" == k {
			defaultConfig = v
		} else if strings.Contains(imageDescription, k) {
			instanceConfig = v
		}
	}
	if len(instanceConfig) == 0 {
		instanceConfig = defaultConfig
	}
	if len(instanceConfig) == 0 {
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.Error("no matching stemcell configuration found")
	}

	err = c.adapter.CreateInstance(adapter.InstanceMetadata{
		Name:          theCid,
		StemcellAlias: stemcellCID.AsString(),
		InstanceType:  vmProps.InstanceType,
		Project:       c.config.Server.Project,
		Profiles:      []string{c.config.Server.Profile},
		Target:        vmProps.Target,
		Devices:       devices,
		Config:        instanceConfig,
	})
	if err != nil {
		c.logger.Error("create_vm", "Failed to create instance '%s': %v", theCid, err)
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Creating VM")
	}
	c.logger.Debug("create_vm", "Instance '%s' created successfully", theCid)

	defer func() {
		if err != nil {
			c.logger.Error("create_vm", "CreateVMV2 failed for '%s', starting cleanup. Error: %v", theCid, err)
			// Clean up ephemeral disk if it was created but not yet attached
			if ephemeralDiskCid != "" {
				c.logger.Debug("create_vm", "Cleaning up unattached ephemeral disk '%s'", ephemeralDiskCid)
				cleanupErr := c.adapter.DeleteStoragePoolVolume(c.config.Server.StoragePool, ephemeralDiskCid)
				if cleanupErr != nil {
					c.logger.Error("create_vm", "Failed to cleanup ephemeral disk '%s': %v", ephemeralDiskCid, cleanupErr)
				} else {
					c.logger.Debug("create_vm", "Successfully cleaned up ephemeral disk '%s'", ephemeralDiskCid)
				}
			}
			c.logger.Debug("create_vm", "Calling DeleteVM for cleanup of '%s'", vmCID.AsString())
			cleanupErr := c.DeleteVM(vmCID)
			if cleanupErr != nil {
				c.logger.Error("create_vm", "Failed to cleanup VM '%s': %v", vmCID.AsString(), cleanupErr)
			} else {
				c.logger.Debug("create_vm", "Successfully cleaned up VM '%s'", vmCID.AsString())
			}
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
		c.logger.Debug("create_vm", "Creating ephemeral disk of size %d MB for VM '%s'", vmProps.EphemeralDisk, theCid)
		diskId, err := c.uuidGen.Generate()
		if err != nil {
			return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Creating Disk id")
		}
		ephemeralDiskCid = DISK_EPHEMERAL_PREFIX + diskId

		err = c.adapter.CreateStoragePoolVolume(target, c.config.Server.StoragePool, ephemeralDiskCid, vmProps.EphemeralDisk)
		if err != nil {
			c.logger.Error("create_vm", "Failed to create ephemeral disk '%s': %v", ephemeralDiskCid, err)
			return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Create ephemeral disk")
		}
		c.logger.Debug("create_vm", "Ephemeral disk '%s' created, attaching to VM '%s'", ephemeralDiskCid, theCid)

		err = c.attachDiskDeviceToVM(vmCID, DISK_DEVICE_EPHEMERAL, ephemeralDiskCid)
		if err != nil {
			c.logger.Error("create_vm", "Failed to attach ephemeral disk '%s' to VM '%s': %v", ephemeralDiskCid, theCid, err)
			return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Attach ephemeral disk")
		}
		// Disk is now attached, DeleteVM will clean it up
		c.logger.Debug("create_vm", "Ephemeral disk '%s' attached to VM '%s'", ephemeralDiskCid, theCid)
		ephemeralDiskCid = ""

		agentEnv.AttachEphemeralDisk(apiv1.NewDiskHintFromString("/dev/sdb"))
	}

	err = c.writeAgentFileToVM(vmCID, agentEnv)
	if err != nil {
		c.logger.Error("create_vm", "Failed to write agent env to VM '%s': %v", theCid, err)
		return apiv1.VMCID{}, apiv1.Networks{}, bosherr.WrapError(err, "Write AgentEnv")
	}
	c.logger.Debug("create_vm", "Agent env written to VM '%s'", theCid)

	c.logger.Debug("create_vm", "Starting VM '%s'", theCid)
	err = c.adapter.SetInstanceAction(vmCID.AsString(), adapter.StartAction)
	if err != nil {
		c.logger.Error("create_vm", "Failed to start VM '%s': %v", theCid, err)
	} else {
		c.logger.Debug("create_vm", "CreateVMV2 completed successfully for VM '%s'", theCid)
	}
	return vmCID, newNetworks, err
}
