package main

import (
	"bytes"
	"strings"

	"github.com/cppforlife/bosh-cpi-go/apiv1"
	lxd "github.com/lxc/lxd/client"
)

func (c CPI) writeFileAsRootToVM(vmCID apiv1.VMCID, filemode int, path string, content string) error {
	fileArgs := lxd.ContainerFileArgs{
		Content:   strings.NewReader(content),
		UID:       0, // root
		GID:       0, // root
		Mode:      filemode,
		Type:      "file",
		WriteMode: "overwrite",
	}
	return c.client.CreateContainerFile(vmCID.AsString(), path, fileArgs)
}

func (c CPI) readFileFromVM(vmCID apiv1.VMCID, path string) (string, error) {
	reader, _, err := c.client.GetContainerFile(vmCID.AsString(), path)
	defer reader.Close()
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	return buf.String(), nil
}
