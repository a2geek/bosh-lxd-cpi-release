package main

import (
	"strings"

	"github.com/cppforlife/bosh-cpi-go/apiv1"
	lxd "github.com/lxc/lxd/client"
)

func (c CPI) writeFilesAsRootToVM(vmCID apiv1.VMCID, filemode int, path string, content string) error {
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
