package adapter

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"path"

	yaml "gopkg.in/yaml.v2"

	client "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func NewLXDAdapter(config Config) (ApiAdapter, error) {
	connectionArgs := &client.ConnectionArgs{
		TLSClientCert:      config.TLSClientCert,
		TLSClientKey:       config.TLSClientKey,
		InsecureSkipVerify: config.InsecureSkipVerify,
	}
	c, err := client.ConnectLXD(config.URL, connectionArgs)
	if err != nil {
		return nil, err
	}

	// If a project has been specified, we use it _always_
	if len(config.Project) != 0 {
		c = c.UseProject(config.Project)
	}
	return &lxdApiAdapter{
		client: c,
	}, nil
}

type lxdApiAdapter struct {
	client client.InstanceServer
}

func (a *lxdApiAdapter) FindExistingImage(description string) (string, error) {
	images, err := a.client.GetImages()
	if err != nil {
		return "", err
	}
	for _, image := range images {
		if description == image.Properties["description"] {
			return image.Aliases[0].Name, nil
		}
	}
	return "", nil
}

func (a *lxdApiAdapter) CreateAndUploadImage(meta ImageMetadata) error {
	image := api.ImagesPost{
		ImagePut: api.ImagePut{
			Public:     false,
			AutoUpdate: false,
		},
		Filename: path.Base(meta.ImagePath),
	}

	metadata := api.ImageMetadata{
		Architecture: meta.Architecture,
		CreationDate: meta.CreateDate,
		Properties: map[string]string{
			"architecture":     meta.Architecture,
			"description":      meta.Description,
			"os":               cases.Title(language.English).String(meta.OsDistro),
			"root_device_name": meta.RootDeviceName,
			"root_disk_size":   fmt.Sprintf("%dMiB", meta.DiskInMB),
		},
	}
	metadataYaml, err := yaml.Marshal(metadata)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	theader := &tar.Header{
		Name: "metadata.yaml",
		Mode: 0600,
		Size: int64(len(metadataYaml)),
	}
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(theader); err != nil {
		return err
	}
	if _, err := tw.Write(metadataYaml); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}

	args := client.ImageCreateArgs{
		MetaFile:   bytes.NewReader(buf.Bytes()),
		RootfsFile: meta.TarFile,
		Type:       string(api.InstanceTypeVM),
	}
	op, err := a.client.CreateImage(image, &args)
	if err != nil {
		return err
	}

	err = op.Wait()
	if err != nil {
		return err
	}

	opAPI := op.Get()
	fingerprint := opAPI.Metadata["fingerprint"].(string)

	imageAliasPost := api.ImageAliasesPost{
		ImageAliasesEntry: api.ImageAliasesEntry{
			Name:        meta.Alias,
			Description: "bosh image",
			Target:      fingerprint,
		},
	}
	return a.client.CreateImageAlias(imageAliasPost)
}

func (a *lxdApiAdapter) CreateInstance(meta InstanceMetadata) error {
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
	return wait(a.client.CreateInstance(instancesPost))
}

func (a *lxdApiAdapter) DeleteStoragePoolVolume(pool, volType, name string) error {
	return a.client.DeleteStoragePoolVolume(pool, volType, name)
}

func (a *lxdApiAdapter) DeleteStoragePoolVolumeSnapshot(pool, volType, volumeName, snapshotName string) error {
	return wait(a.client.DeleteStoragePoolVolumeSnapshot(pool, volType, volumeName, snapshotName))
}

func (a *lxdApiAdapter) DeleteImage(alias string) error {
	imageAlias, _, err := a.client.GetImageAlias(alias)
	if err != nil {
		return err
	}
	return wait(a.client.DeleteImage(imageAlias.Target))
}

func (a *lxdApiAdapter) DeleteInstance(name string) error {
	return wait(a.client.DeleteInstance(name))
}

func (a *lxdApiAdapter) AttachDevice(instanceName, deviceName string, device map[string]string) error {
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

func (a *lxdApiAdapter) DetachDevice(instanceName, deviceName string) error {
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

func (a *lxdApiAdapter) GetStoragePoolVolume(pool, volType, name string) (string, error) {
	_, etag, err := a.client.GetStoragePoolVolume(pool, volType, name)
	return etag, err
}

func (a *lxdApiAdapter) GetInstance(name string) (string, error) {
	_, etag, err := a.client.GetInstance(name)
	return etag, err
}

func (a *lxdApiAdapter) ResizeStoragePoolVolume(pool, volType, name string, newSize int) error {
	volume, etag, err := a.client.GetStoragePoolVolume(pool, volType, name)
	if err != nil {
		return err
	}

	writable := volume.Writable()
	writable.Config["size"] = fmt.Sprintf("%dMiB", newSize)

	return a.client.UpdateStoragePoolVolume(pool, volType, name, writable, etag)
}

func (a *lxdApiAdapter) UpdateInstanceDescription(name, newDescription string) error {
	instance, etag, err := a.client.GetInstance(name)
	if err != nil {
		return err
	}

	instance.Description = newDescription

	return wait(a.client.UpdateInstance(name, instance.Writable(), etag))
}

func (a *lxdApiAdapter) CreateStoragePoolVolumeSnapshot(pool, volType, volumeName, snapshotName, description string) error {
	post := api.StorageVolumeSnapshotsPost{
		Name:        snapshotName,
		Description: description,
	}
	return wait(a.client.CreateStoragePoolVolumeSnapshot(pool, volType, volumeName, post))
}

func (a *lxdApiAdapter) CreateStoragePoolVolumeFromISO(pool, diskName string, backupFile io.Reader) error {
	return wait(a.client.CreateStoragePoolVolumeFromISO(pool, client.StoragePoolVolumeBackupArgs{
		Name:       diskName,
		BackupFile: backupFile,
	}))
}

func (a *lxdApiAdapter) CreateStoragePoolVolume(pool, name string, size int) error {
	storageVolumeRequest := api.StorageVolumesPost{
		Name:        name,
		Type:        "custom",
		ContentType: "block",
		StorageVolumePut: api.StorageVolumePut{
			Config: map[string]string{
				"size": fmt.Sprintf("%dMiB", size),
			},
		},
	}

	return a.client.CreateStoragePoolVolume(pool, storageVolumeRequest)
}

func (a *lxdApiAdapter) GetDevices(instanceName string) (map[string]map[string]string, error) {
	instance, _, err := a.client.GetInstance(instanceName)
	if err != nil {
		return nil, err
	}
	return instance.Devices, nil
}

func (a *lxdApiAdapter) UpdateStoragePoolVolumeDescription(poolName, diskName, description string) error {
	volume, etag, err := a.client.GetStoragePoolVolume(poolName, "custom", diskName)
	if err != nil {
		return err
	}

	volume.Description = description

	return a.client.UpdateStoragePoolVolume(poolName, "custom", diskName, volume.Writable(), etag)
}

func (a *lxdApiAdapter) SetInstanceAction(instanceName, action string) error {
	atCurrentState, err := a.isVMAtRequestedState(instanceName, action)
	if err != nil {
		return err
	}
	if !atCurrentState {
		req := api.InstanceStatePut{
			Action:   action,
			Timeout:  30,
			Force:    true,
			Stateful: false,
		}

		err = wait(a.client.UpdateInstanceState(instanceName, req, ""))
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *lxdApiAdapter) IsInstanceStopped(instanceName string) (bool, error) {
	return a.isVMAtRequestedState(instanceName, "stop")
}

// isVMAtRequestedState tests if this action has already been done or completed.
func (a *lxdApiAdapter) isVMAtRequestedState(instanceName, action string) (bool, error) {
	currentState, _, err := a.client.GetInstanceState(instanceName)
	if err != nil {
		return false, err
	}

	atRequestedState := false

	switch action {
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
