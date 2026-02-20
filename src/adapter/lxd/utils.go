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

	// If the original error was swallowed by checkError (e.g., "LXD VM agent" error),
	// but op is nil, we cannot wait - just return successfully since the error was
	// deemed non-fatal.
	if op == nil {
		return nil
	}

	err = op.Wait()
	if checkError(err) != nil {
		return bosherr.WrapErrorf(err, "Wait")
	}

	return nil
}
