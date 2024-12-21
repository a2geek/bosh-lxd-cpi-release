package incus

import (
	"bosh-lxd-cpi/adapter"

	client "github.com/lxc/incus/client"
)

func NewIncusAdapter(config adapter.Config) (adapter.ApiAdapter, error) {
	connectionArgs := &client.ConnectionArgs{
		TLSClientCert:      config.TLSClientCert,
		TLSClientKey:       config.TLSClientKey,
		InsecureSkipVerify: config.InsecureSkipVerify,
	}
	c, err := client.ConnectIncus(config.URL, connectionArgs)
	if err != nil {
		return nil, err
	}

	// If a project has been specified, we use it _always_
	if len(config.Project) != 0 {
		c = c.UseProject(config.Project)
	}
	if len(config.Target) != 0 {
		c = c.UseTarget(config.Target)
	}
	return &incusApiAdapter{
		client: c,
	}, nil
}

type incusApiAdapter struct {
	client client.InstanceServer
}
