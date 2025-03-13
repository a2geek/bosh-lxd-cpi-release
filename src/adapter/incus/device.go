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

func (a *incusApiAdapter) DetachDeviceByName(instanceName, deviceName string) error {
	instance, _, err := a.client.GetInstance(instanceName)
	if err != nil {
		return err
	}

	// Check that the device already exists
	_, ok := instance.Devices[deviceName]
	if ok {
		delete(instance.Devices, deviceName)

		return wait(a.client.UpdateInstance(instanceName, instance.Writable(), ""))
	} else {
		a.logger.Warn("incusApiAdapter", "device does not exist: '%s'", deviceName)
		return nil
	}
}

func (a *incusApiAdapter) DetachDeviceBySource(instanceName, sourceName string) error {
	instance, _, err := a.client.GetInstance(instanceName)
	if err != nil {
		return err
	}

	// Check that the device already exists
	deviceName := ""
	for name, device := range instance.Devices {
		if device["source"] == sourceName {
			deviceName = name
			break
		}
	}
	if deviceName == "" {
		a.logger.Warn("incusApiAdapter", "device source does not exist: '%s'", sourceName)
		return nil
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
