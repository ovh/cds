package keychain

import "fmt"

// Errors
var (
	ErrLibSecretUnavailable = fmt.Errorf("Shared library libsecret not found. Please install libsecret-1 from your package manager")
	ErrLdd                  = fmt.Errorf("Unable to check shared object dependencies")
	ErrExecNotFound         = fmt.Errorf("Unable to get current binary file")
)
