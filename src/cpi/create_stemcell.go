package cpi

import (
	"archive/tar"
	"bosh-lxd-cpi/adapter"
	"compress/gzip"
	"os"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
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

	existing, err := c.adapter.FindExistingImage(description)
	if err != nil {
		return apiv1.StemcellCID{}, bosherr.WrapError(err, "error while locating image")
	}
	if existing != "" {
		return apiv1.NewStemcellCID(existing), nil
	}

	alias := "img-" + id

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

	rootDeviceName := props.RootDeviceName
	if rootDeviceName == "" {
		rootDeviceName = "/"
	}

	err = c.adapter.CreateAndUploadImage(adapter.ImageMetadata{
		Alias:          alias,
		Description:    description,
		OsDistro:       props.OsDistro,
		ImagePath:      imagePath,
		Architecture:   props.Architecture,
		CreateDate:     createDate,
		RootDeviceName: rootDeviceName,
		DiskInMB:       props.Disk,
		TarFile:        tarFile,
	})
	return apiv1.NewStemcellCID(alias), err
}
