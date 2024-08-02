package cpi

import (
	"bytes"
	"strings"

	lxd "github.com/canonical/lxd/client"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

func (c CPI) writeFileAsRootToVM(vmCID apiv1.VMCID, filemode int, path string, content string) error {
	fileArgs := lxd.InstanceFileArgs{
		Content:   strings.NewReader(content),
		UID:       0, // root
		GID:       0, // root
		Mode:      filemode,
		Type:      "file",
		WriteMode: "overwrite",
	}
	return c.client.CreateInstanceFile(vmCID.AsString(), path, fileArgs)
}

func (c CPI) readFileFromVM(vmCID apiv1.VMCID, path string) (string, error) {
	reader, _, err := c.client.GetInstanceFile(vmCID.AsString(), path)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	return buf.String(), nil
}
