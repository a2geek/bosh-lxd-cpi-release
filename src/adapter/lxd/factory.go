package lxd

import (
	"bosh-lxd-cpi/adapter"

	client "github.com/canonical/lxd/client"
)

func NewLXDAdapter(config adapter.Config) (adapter.ApiAdapter, error) {
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
