package lxd

import (
	"fmt"

	"github.com/canonical/lxd/shared/api"
)

func (a *lxdApiAdapter) IsConnected() error {
	server, _, err := a.client.GetServer()
	if err != nil {
		return err
	}
	if api.AuthTrusted != server.Auth {
		return fmt.Errorf("not connected with trusted connection; using '%s'", server.Auth)
	}
	return nil
}

func (a *lxdApiAdapter) Disconnect() {
	a.client.Disconnect()
}
