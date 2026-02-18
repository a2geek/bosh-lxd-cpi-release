package incus

import (
	"bosh-lxd-cpi/adapter"
	"strings"
	"time"

	"github.com/lxc/incus/v6/shared/api"
)

// isInvalidPIDError checks if the error is an "Invalid PID" error from Incus.
// This occurs when trying to stop a VM that has no valid QEMU process,
// typically because the VM is already stopped or in a race condition.
func isInvalidPIDError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "Invalid PID")
}

func (a *incusApiAdapter) CreateInstance(meta adapter.InstanceMetadata) error {
	instancesPost := api.InstancesPost{
		InstancePut: api.InstancePut{
			Devices:  meta.Devices,
			Profiles: meta.Profiles,
			Config:   meta.Config,
		},
		Name:         meta.Name,
		InstanceType: meta.InstanceType,
		Source: api.InstanceSource{
			Type:    "image",
			Alias:   meta.StemcellAlias,
			Project: meta.Project,
		},
		Type: api.InstanceTypeVM,
	}
	c := a.client
	if meta.Target != "" {
		c = c.UseTarget(meta.Target)
	}
	return wait(c.CreateInstance(instancesPost))
}

func (a *incusApiAdapter) DeleteInstance(name string) error {
	return wait(a.client.DeleteInstance(name))
}

func (a *incusApiAdapter) GetInstanceLocation(name string) (string, error) {
	instance, _, err := a.client.GetInstance(name)
	return instance.Location, err
}

func (a *incusApiAdapter) UpdateInstanceDescription(name, newDescription string) error {
	// Retry up to 5 times with 1 second delay to handle "instance busy" errors
	// that occur when the instance is in the middle of another operation.
	var lastErr error
	for i := 0; i < 5; i++ {
		instance, etag, err := a.client.GetInstance(name)
		if err != nil {
			return err
		}

		instance.Description = newDescription

		err = wait(a.client.UpdateInstance(name, instance.Writable(), etag))
		if err == nil {
			return nil
		}

		lastErr = err
		// If instance is busy with another operation, wait and retry
		if isInstanceBusyError(err) {
			time.Sleep(1 * time.Second)
			continue
		}
		// For other errors, return immediately
		return err
	}
	return lastErr
}

// isInstanceBusyError checks if the error indicates the instance is busy with another operation.
func isInstanceBusyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "Instance is busy")
}

func (a *incusApiAdapter) SetInstanceAction(instanceName string, action adapter.Action) error {
	atCurrentState, err := a.isVMAtRequestedState(instanceName, string(action))
	if err != nil {
		return err
	}
	if !atCurrentState {
		req := api.InstanceStatePut{
			Action:   string(action),
			Timeout:  30,
			Force:    true,
			Stateful: false,
		}

		err = wait(a.client.UpdateInstanceState(instanceName, req, ""))
		if err != nil {
			// Handle "Invalid PID" errors during stop action - this occurs when the
			// VM is already stopped or stopping but Incus has a stale QEMU process state.
			// Treat this as "already stopped" and continue.
			if action == adapter.StopAction && isInvalidPIDError(err) {
				return nil
			}
			return err
		}
	}

	return nil
}

func (a *incusApiAdapter) IsInstanceStopped(instanceName string) (bool, error) {
	return a.isVMAtRequestedState(instanceName, "stop")
}

// isVMAtRequestedState tests if this action has already been done or completed.
func (a *incusApiAdapter) isVMAtRequestedState(instanceName, state string) (bool, error) {
	currentState, _, err := a.client.GetInstanceState(instanceName)
	if err != nil {
		return false, err
	}

	atRequestedState := false

	switch state {
	case "stop":
		atRequestedState = a.hasMatch(currentState.StatusCode, api.Stopped, api.Stopping)
	case "start":
		atRequestedState = a.hasMatch(currentState.StatusCode, api.Started, api.Starting, api.Running)
	}

	return atRequestedState, nil
}

func (a *incusApiAdapter) hasMatch(actual api.StatusCode, values ...api.StatusCode) bool {
	for _, expected := range values {
		if actual == expected {
			return true
		}
	}
	return false
}
