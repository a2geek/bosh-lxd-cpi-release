package incus

import (
	"fmt"
)

func (a *incusApiAdapter) AttachDevice(instanceName, deviceName string, device map[string]string) error {
	instance, etag, err := a.client.GetInstance(instanceName)
	if err != nil {
		return err
	}

	// Check if the device already exists
	_, ok := instance.Devices[deviceName]
	if ok {
		return fmt.Errorf("device already exists: '%s'", deviceName)
	}

	instance.Devices[deviceName] = device

	return wait(a.client.UpdateInstance(instanceName, instance.Writable(), etag))
}

func (a *incusApiAdapter) DetachDevice(instanceName, deviceName string) error {
	instance, _, err := a.client.GetInstance(instanceName)
	if err != nil {
		return err
	}

	// Check if the device already exists
	_, ok := instance.Devices[deviceName]
	if !ok {
		return fmt.Errorf("device does not exist: '%s'", deviceName)
	}

	delete(instance.Devices, deviceName)

	return wait(a.client.UpdateInstance(instanceName, instance.Writable(), ""))
}

func (a *incusApiAdapter) GetDevices(instanceName string) (map[string]map[string]string, error) {
	instance, _, err := a.client.GetInstance(instanceName)
	if err != nil {
		return nil, err
	}
	return instance.Devices, nil
}
