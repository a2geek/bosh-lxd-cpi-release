package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path"
	"strconv"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	lxdclient "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/canonical/lxd/shared/ioprogress"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	yaml "gopkg.in/yaml.v2"
)

func (c CPI) CreateStemcell(imagePath string, scprops apiv1.StemcellCloudProps) (apiv1.StemcellCID, error) {
	id, err := c.uuidGen.Generate()
	if err != nil {
		return apiv1.StemcellCID{}, bosherr.WrapError(err, "Generating stemcell id")
	}

	props := LXDCloudProperties{}
	err = scprops.As(&props)
	if err != nil {
		return apiv1.StemcellCID{}, bosherr.WrapError(err, "error while reading stemcell cloud properties")
	}

	description := props.Name + "-" + props.Version

	images, err := c.client.GetImages()
	if err != nil {
		return apiv1.StemcellCID{}, bosherr.WrapError(err, "error while inspecting images")
	}
	for _, image := range images {
		if description == image.Properties["description"] {
			alias := image.Aliases[0].Name
			return apiv1.NewStemcellCID(alias), nil
		}
	}

	alias := "img-" + id
	image := api.ImagesPost{
		ImagePut: api.ImagePut{
			Public:     false,
			AutoUpdate: false,
		},
		Filename: path.Base(imagePath),
	}
	fmt.Fprintf(os.Stderr, "%v\n", image)

	tarGzip, err := os.Open(imagePath)
	if err != nil {
		return apiv1.StemcellCID{}, bosherr.WrapError(err, "tgz open")
	}
	defer tarGzip.Close()

	gz, err := gzip.NewReader(tarGzip)
	if err != nil {
		return apiv1.StemcellCID{}, bosherr.WrapError(err, "gzip open")
	}
	defer gz.Close()

	// Search for a biggish file (100MiB+) to bypass the smaller metadata details. Which are ignored. Bug or feature?
	tarFile := tar.NewReader(gz)
	found := false
	createDate := int64(0)
	for !found {
		h, err := tarFile.Next()
		if err != nil {
			return apiv1.StemcellCID{}, bosherr.WrapErrorf(err, "tar.next for imagePath '%s'", imagePath)
		}
		found = h.Size > 104857600
		createDate = h.ModTime.Unix()
	}
	if !found {
		return apiv1.StemcellCID{}, bosherr.WrapError(err, "unable to locate stemcell in archive")
	}

	metadata := api.ImageMetadata{
		Architecture: props.Architecture,
		CreationDate: createDate,
		Properties: map[string]string{
			"architecture":     props.Architecture,
			"description":      description,
			"os":               cases.Title(language.English).String(props.OsDistro),
			"root_device_name": props.RootDeviceName,
			"root_disk_size":   strconv.Itoa(props.Disk),
		},
	}
	metadataYaml, err := yaml.Marshal(metadata)
	if err != nil {
		return apiv1.StemcellCID{}, bosherr.WrapError(err, "creating metadata yaml file")
	}

	var buf bytes.Buffer
	theader := &tar.Header{
		Name: "metadata.yaml",
		Mode: 0600,
		Size: int64(len(metadataYaml)),
	}
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(theader); err != nil {
		return apiv1.StemcellCID{}, bosherr.WrapError(err, "tar write header")
	}
	if _, err := tw.Write(metadataYaml); err != nil {
		return apiv1.StemcellCID{}, bosherr.WrapError(err, "tar write metadata file")
	}
	if err := tw.Close(); err != nil {
		return apiv1.StemcellCID{}, bosherr.WrapError(err, "tar close")
	}

	args := lxdclient.ImageCreateArgs{
		MetaFile:        bytes.NewReader(buf.Bytes()),
		RootfsFile:      tarFile,
		ProgressHandler: dummyProgressHandler,
		Type:            string(api.InstanceTypeVM),
	}
	op, err := c.client.CreateImage(image, &args)
	if err != nil {
		return apiv1.StemcellCID{}, bosherr.WrapError(err, "Importing image - start")
	}

	err = op.Wait()
	if err != nil {
		return apiv1.StemcellCID{}, bosherr.WrapError(err, "Importing image - processing")
	}

	opAPI := op.Get()
	fingerprint := opAPI.Metadata["fingerprint"].(string)

	imageAliasPost := api.ImageAliasesPost{}
	imageAliasPost.Name = alias
	imageAliasPost.Description = "bosh image"
	imageAliasPost.Target = fingerprint
	err = c.client.CreateImageAlias(imageAliasPost)
	if err != nil {
		return apiv1.StemcellCID{}, bosherr.WrapError(err, "setting alias")
	}

	return apiv1.NewStemcellCID(alias), nil
}

func (c CPI) DeleteStemcell(cid apiv1.StemcellCID) error {
	alias := cid.AsString()
	imageAlias, _, err := c.client.GetImageAlias(alias)
	if err != nil {
		return err
	}
	op, err := c.client.DeleteImage(imageAlias.Target)
	if err != nil {
		return err
	}
	err = op.Wait()
	if err != nil {
		return err
	}
	return nil
}

func dummyProgressHandler(progress ioprogress.ProgressData) {
	// DO NOTHING!
}
