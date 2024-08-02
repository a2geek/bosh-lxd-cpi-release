package agentmgr

import (
	"io"
	"os"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/rn/iso9660wrap"
)

func NewCDROMManager(config Config) (AgentManager, error) {
	name, err := tempFileName("iso")
	if err != nil {
		return nil, err
	}
	mgr := cdromManager{
		filename: name,
		config:   config,
	}
	return mgr, nil
}

type cdromManager struct {
	filename string
	config   Config
}

func (m cdromManager) Update(agentEnv apiv1.AgentEnv) error {
	buf, err := agentEnv.AsBytes()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(m.filename, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	return iso9660wrap.WriteBuffer(f, buf, m.config.Filename)
}

func (m cdromManager) ToBytes() ([]byte, error) {
	f, err := os.Open(m.filename)
	defer f.Close()
	if err != nil {
		return nil, err
	}

	return io.ReadAll(f)
}
