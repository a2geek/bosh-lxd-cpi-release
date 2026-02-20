package lxd

import (
	"bosh-lxd-cpi/adapter"
	"strings"
	"time"

	"github.com/canonical/lxd/shared/api"
)

// isInvalidPIDError checks if the error is an "Invalid PID" error from LXD.
// This occurs when trying to stop a VM that has no valid QEMU process,
// typically because the VM is already stopped or in a race condition.
func isInvalidPIDError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "Invalid PID")
}

func (a *lxdApiAdapter) CreateInstance(meta adapter.InstanceMetadata) error {
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

func (a *lxdApiAdapter) DeleteInstance(name string) error {
	err := wait(a.client.DeleteInstance(name))
	if err != nil {
		return err
	}

	// Verify the instance is actually gone by polling until GetInstance returns "not found".
	// This handles the case where the operation completes but the instance is still
	// visible in the system for a brief period.
	for i := 0; i < 30; i++ {
		_, _, err := a.client.GetInstance(name)
		if err != nil && strings.Contains(err.Error(), "not found") {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return nil
}

func (a *lxdApiAdapter) GetInstanceLocation(name string) (string, error) {
	instance, _, err := a.client.GetInstance(name)
	if err != nil {
		return "", err
	}
	return instance.Location, nil
}

func (a *lxdApiAdapter) UpdateInstanceDescription(name, newDescription string) error {
	instance, etag, err := a.client.GetInstance(name)
	if err != nil {
		return err
	}

	instance.Description = newDescription

	return wait(a.client.UpdateInstance(name, instance.Writable(), etag))
}

func (a *lxdApiAdapter) SetInstanceAction(instanceName string, action adapter.Action) error {
	atCurrentState, err := a.isVMAtRequestedState(instanceName, string(action))
	if err != nil {
		// Handle "Invalid PID" errors during stop action - this can occur when
		// checking state of a VM that is in a bad state.
		if action == adapter.StopAction && isInvalidPIDError(err) {
			return nil
		}
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
			// VM is already stopped or stopping but LXD has a stale QEMU process state.
			// Treat this as "already stopped" and continue.
			if action == adapter.StopAction && isInvalidPIDError(err) {
				return nil
			}
			return err
		}
	}

	return nil
}

func (a *lxdApiAdapter) IsInstanceStopped(instanceName string) (bool, error) {
	return a.isVMAtRequestedState(instanceName, "stop")
}

// isVMAtRequestedState tests if this action has already been done or completed.
func (a *lxdApiAdapter) isVMAtRequestedState(instanceName, state string) (bool, error) {
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

func (a *lxdApiAdapter) hasMatch(actual api.StatusCode, values ...api.StatusCode) bool {
	for _, expected := range values {
		if actual == expected {
			return true
		}
	}
	return false
}
