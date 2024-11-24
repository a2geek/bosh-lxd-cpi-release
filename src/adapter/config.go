package adapter

type Config struct {
	Type               string
	URL                string
	TLSClientCert      string
	TLSClientKey       string
	InsecureSkipVerify bool
	Project            string
	Profile            string
	Target             string
	Managed            bool
	Network            string
	StoragePool        string
	BIOSPath           string
}

func (lxd Config) Validate() error {
	// DO NOTHING AT THIS TIME!
	return nil
}
