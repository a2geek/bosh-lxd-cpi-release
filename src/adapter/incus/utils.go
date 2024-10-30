package incus

import (
	"strings"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	client "github.com/lxc/incus/client"
)

// General utilities; not dependent on the CPI structure

func checkError(err error) error {
	if err != nil && strings.Contains(err.Error(), "LXD VM agent") {
		return nil
	}
	return err
}

func wait(op client.Operation, err error) error {
	if checkError(err) != nil {
		return err
	}

	err = op.Wait()
	if checkError(err) != nil {
		return bosherr.WrapErrorf(err, "Wait")
	}

	return nil
}
