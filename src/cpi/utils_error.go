package cpi

import "strings"

// CleanInstanceNotFoundError returns nil if the error is an instance not found error, otherwise returns the original error
func CleanInstanceNotFoundError(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "instance not found") {
		return nil
	}
	return err
}
