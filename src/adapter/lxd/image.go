package lxd

import (
	"archive/tar"
	"bosh-lxd-cpi/adapter"
	"bytes"
	"fmt"
	"path"

	yaml "gopkg.in/yaml.v2"

	client "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

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

func (a *lxdApiAdapter) GetStemcellInfo(alias string) (*adapter.ImageInfo, error) {
	entry, _, err := a.client.GetImageAlias(alias)
	if err != nil {
		return nil, err
	}
	image, _, err := a.client.GetImage(entry.Target)
	if err != nil {
		return nil, err
	}
	description, b := image.Properties["description"]
	if !b {
		return nil, fmt.Errorf("no description for image '%s'", entry.Target)
	}

	instanceType := adapter.InstanceVM
	if image.Type == string(api.InstanceTypeContainer) {
		instanceType = adapter.InstanceContainer
	}

	info := &adapter.ImageInfo{
		Description: description,
		Type:        instanceType,
	}

	return info, nil
}

func (a *lxdApiAdapter) CreateAndUploadImage(meta adapter.ImageMetadata) error {
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

	instanceType := string(api.InstanceTypeVM)
	if meta.Type == adapter.InstanceContainer {
		instanceType = string(api.InstanceTypeContainer)
	}

	args := client.ImageCreateArgs{
		MetaFile:   bytes.NewReader(buf.Bytes()),
		RootfsFile: meta.TarFile,
		Type:       instanceType,
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
			Description: meta.Description,
			Target:      fingerprint,
		},
	}
	return a.client.CreateImageAlias(imageAliasPost)
}

func (a *lxdApiAdapter) DeleteImage(alias string) error {
	imageAlias, _, err := a.client.GetImageAlias(alias)
	if err != nil {
		return err
	}
	return wait(a.client.DeleteImage(imageAlias.Target))
}
