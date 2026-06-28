package cpi

import (
	"archive/tar"
	"bosh-lxd-cpi/adapter"
	"compress/gzip"
	"fmt"
	"os"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c CPI) CreateStemcell(imagePath string, scprops apiv1.StemcellCloudProps) (apiv1.StemcellCID, error) {
	err := c.adapter.IsConnected()
	if err != nil {
		return apiv1.StemcellCID{}, err
	}

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
	if err != nil {
		return apiv1.StemcellCID{}, bosherr.WrapError(err, "error while creating and uploading image")
	}

	if c.config.Server.PredeployStemcell {
		c.logger.Info("create_stemcell", "Predeploying stemcell '%s'", alias)

		// Predeploy a very basic VM. Don't even start it. We just want LXD/Incus to do the image processing so it's ready for multiple VMs to be created at once.
		// This is a workaround for the fact that LXD/Incus does not support multiple concurrent image processing from multiple hosts (I think a single host + remote is ok).
		vmID := fmt.Sprintf("vm-%s-predeploy", id)
		devices := make(map[string]map[string]string)
		devices["root"] = map[string]string{
			"type": "disk",
			"pool": c.config.Server.StoragePool,
			"path": "/",
		}

		err = c.adapter.CreateInstance(adapter.InstanceMetadata{
			Name:          vmID,
			StemcellAlias: alias,
			InstanceType:  "c2-m4", // Just enough to get the stemcell preprocessed.
			Project:       c.config.Server.Project,
			Profiles:      []string{c.config.Server.Profile},
			Target:        c.config.Server.Target,
			Devices:       devices,
			// TODO uncertain if we need Config. It primarily determines the type of bootloader (BIOS vs UEFI). But we're not booting this VM, so it may not matter.
			//Config:        instanceConfig,
		})
		if err != nil {
			return apiv1.StemcellCID{}, bosherr.WrapError(err, "Creating predeploy VM")
		}

		defer func() {
			c.DeleteVM(apiv1.NewVMCID(vmID))
		}()
	}

	return apiv1.NewStemcellCID(alias), nil
}
