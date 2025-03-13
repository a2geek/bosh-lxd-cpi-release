package lxd

import (
	"strings"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

// General utilities; not dependent on the CPI structure

func checkError(err error) error {
	if err != nil && strings.Contains(err.Error(), "LXD VM agent") {
		return nil
	}
	return err
}

// Reducing multiple interfaces to what we use
type WaitOperation interface {
	Wait() (err error)
}

func wait(op WaitOperation, err error) error {
	if checkError(err) != nil {
		return err
	}

	err = op.Wait()
	if checkError(err) != nil {
		return bosherr.WrapErrorf(err, "Wait")
	}

	return nil
}
