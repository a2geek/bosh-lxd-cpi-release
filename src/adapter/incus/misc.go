package incus

func (a *incusApiAdapter) Disconnect() {
	a.client.Disconnect()
}
