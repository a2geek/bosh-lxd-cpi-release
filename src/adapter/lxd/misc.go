package lxd

func (a *lxdApiAdapter) Disconnect() {
	a.client.Disconnect()
}
