package lxd

func (a *lxdApiAdapter) IsManagedNetwork(name string) (bool, error) {
	network, _, err := a.client.GetNetwork(name)
	if err != nil {
		return false, err
	}

	return network.Managed, nil
}
