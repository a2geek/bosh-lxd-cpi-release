package agentmgr

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/diskfs/go-diskfs/filesystem"
)

// Note that AGENT_DIR is assumed to be created by the cpi control script
const AGENT_DIR = "/var/vcap/store/agent-data"

type agentFileManager struct {
	agentEnvFactory apiv1.AgentEnvFactory
}

// tempFileName create a config file with the associated extension
func (afm agentFileManager) tempFileName(ext string) (string, error) {
	pattern := fmt.Sprintf("config-*.%s", ext)

	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	name := f.Name()
	err = f.Close()
	if err != nil {
		return "", err
	}
	err = os.Remove(name)
	if err != nil {
		return "", err
	}
	return name, nil
}

// agentFileName ensures we have a consistent name and path for the agent file
func (afm agentFileManager) agentFileName(vmCID apiv1.VMCID) string {
	return fmt.Sprintf("%s/%s.json", AGENT_DIR, vmCID.AsString())
}

// Read pulls an existing AgentEnv from our local copy
func (afm agentFileManager) Read(vmCID apiv1.VMCID) (apiv1.AgentEnv, error) {
	fileName := afm.agentFileName(vmCID)
	data, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	return afm.agentEnvFactory.FromBytes(data)
}

// writeAgentEnv has two functions: (1) persist local copy and (2) return the bytes for disk creation
func (afm agentFileManager) writeAgentEnv(vmCID apiv1.VMCID, agentEnv apiv1.AgentEnv) ([]byte, error) {
	fileName := afm.agentFileName(vmCID)

	data, err := agentEnv.AsBytes()
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(fileName, data, fs.ModePerm)
	return data, err
}

// writeFile is common code for writing into the go-diskfs FileSystem
func (afm agentFileManager) writeFile(fs filesystem.FileSystem, path string, contents []byte) error {
	if path == "" {
		return nil
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	rw, err := fs.OpenFile(path, os.O_CREATE|os.O_RDWR)
	if err != nil {
		return err
	}

	_, err = rw.Write(contents)
	if err != nil {
		return err
	}

	return rw.Close()
}

// mkdirAll is common code for creating all subdirectories for a go-diskfs FileSystem
func (afm agentFileManager) mkdirAll(fs filesystem.FileSystem, fullPath string) error {
	paths := strings.Split(fullPath, "/")
	var dir string
	for _, path := range paths {
		if dir != "" {
			dir += "/"
		}
		dir += path
		err := fs.Mkdir(dir)
		if err != nil {
			return err
		}
	}
	return nil
}
