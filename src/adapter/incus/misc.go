package incus

import (
	"fmt"
)

func (a *incusApiAdapter) IsConnected() error {
	server, _, err := a.client.GetServer()
	if err != nil {
		return err
	}
	// LXD has a constant but Incus does not
	if "trusted" != server.Auth {
		return fmt.Errorf("not connected with trusted connection; using '%s'", server.Auth)
	}
	return nil
}

func (a *incusApiAdapter) Disconnect() {
	a.client.Disconnect()
}
