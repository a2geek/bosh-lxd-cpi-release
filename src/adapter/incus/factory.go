package incus

import (
	"bosh-lxd-cpi/adapter"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	client "github.com/lxc/incus/client"
)

func NewIncusAdapter(config adapter.Config, logger boshlog.Logger) (adapter.ApiAdapter, error) {
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
		logger: logger,
	}, nil
}

type incusApiAdapter struct {
	client client.InstanceServer
	logger boshlog.Logger
}
