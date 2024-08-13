package cpi

import "strings"

func (c CPI) checkError(err error) error {
	if err != nil && strings.Contains(err.Error(), "LXD VM agent") {
		return nil
	}
	return err
}
