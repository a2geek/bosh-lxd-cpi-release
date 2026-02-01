package adapter

type Config struct {
	Type                     string
	URL                      string
	TLSClientCert            string
	TLSClientKey             string
	InsecureSkipVerify       bool
	Project                  string
	Profile                  string
	Target                   string
	ManagedNetworkAssignment string
	Network                  string
	StoragePool              string
	// InstanceConfig is a map of a map of strings. That is, the first key is the
	// stemcell identifier and the map it contains is used for configuration.
	InstanceConfig map[string]map[string]string
}

func (lxd Config) Validate() error {
	// DO NOTHING AT THIS TIME!
	return nil
}
