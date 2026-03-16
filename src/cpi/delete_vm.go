package cpi

import (
	"bosh-lxd-cpi/adapter"
	"strings"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

// isNotFoundError checks if the error indicates a resource was not found.
// This is used to gracefully handle cleanup of resources that may not exist.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "not found") || strings.Contains(errStr, "Not found")
}

func (c CPI) DeleteVM(vmCID apiv1.VMCID) error {
	c.logger.Debug("delete_vm", "Starting DeleteVM for '%s'", vmCID.AsString())

	err := c.adapter.SetInstanceAction(vmCID.AsString(), adapter.StopAction)
	if err != nil {
		// If VM is not found, it's already deleted - treat as success
		if isNotFoundError(err) {
			c.logger.Debug("delete_vm", "VM '%s' not found during stop, treating as already deleted", vmCID.AsString())
			// Still try to cleanup agent files
			_ = c.agentMgr.Delete(vmCID)
			return nil
		}
		c.logger.Error("delete_vm", "Failed to stop VM '%s': %v", vmCID.AsString(), err)
		return bosherr.WrapError(err, "Delete VM - stop")
	}
	c.logger.Debug("delete_vm", "VM '%s' stopped", vmCID.AsString())

	disks, err := c.findDisksAttachedToVm(vmCID)
	if err != nil {
		// If VM is not found during disk enumeration, it may have been deleted by another process
		if isNotFoundError(err) {
			c.logger.Debug("delete_vm", "VM '%s' not found during disk enumeration, treating as already deleted", vmCID.AsString())
			_ = c.agentMgr.Delete(vmCID)
			return nil
		}
		c.logger.Error("delete_vm", "Failed to enumerate disks for VM '%s': %v", vmCID.AsString(), err)
		return bosherr.WrapError(err, "Delete VM - enumerate disks")
	}
	c.logger.Debug("delete_vm", "Found %d disk devices on VM '%s'", len(disks), vmCID.AsString())

	err = c.adapter.DeleteInstance(vmCID.AsString())
	if err != nil {
		// If instance is not found, it's already deleted
		if isNotFoundError(err) {
			c.logger.Debug("delete_vm", "Instance '%s' not found during delete, treating as already deleted", vmCID.AsString())
		} else {
			c.logger.Error("delete_vm", "Failed to delete instance '%s': %v", vmCID.AsString(), err)
			return bosherr.WrapError(err, "Delete VM")
		}
	} else {
		c.logger.Debug("delete_vm", "Instance '%s' deleted", vmCID.AsString())
	}

	for name, disk := range disks {
		if name == DISK_DEVICE_CONFIG || name == DISK_DEVICE_EPHEMERAL {
			diskId := disk["source"]
			c.logger.Debug("delete_vm", "Deleting attached disk '%s' (device: %s) from VM '%s'", diskId, name, vmCID.AsString())
			err = c.adapter.DeleteStoragePoolVolume(c.config.Server.StoragePool, diskId)
			if err != nil {
				// If disk is not found, it may have been cleaned up already
				if isNotFoundError(err) {
					c.logger.Debug("delete_vm", "Disk '%s' not found, treating as already deleted", diskId)
					continue
				}
				c.logger.Error("delete_vm", "Failed to delete disk '%s': %v", diskId, err)
				return bosherr.WrapErrorf(err, "Delete VM - attached disk - %s", diskId)
			}
		}
	}

	c.logger.Debug("delete_vm", "Cleaning up agent files for VM '%s'", vmCID.AsString())
	err = c.agentMgr.Delete(vmCID)
	if err != nil {
		c.logger.Error("delete_vm", "Failed to delete agent files for VM '%s': %v", vmCID.AsString(), err)
	} else {
		c.logger.Debug("delete_vm", "DeleteVM completed successfully for '%s'", vmCID.AsString())
	}
	return err
}
