package keychain

import (
	"fmt"
	"strings"

	"github.com/docker/docker-credential-helpers/credentials"
)

func init() {
	credentials.CredsLabel = "CDS Credentials"
}

// Errors
var (
	ErrLibSecretUnavailable = fmt.Errorf("Shared library libsecret not found. Please install libsecret-1 from your package manager")
	ErrLdd                  = fmt.Errorf("Unable to check shared object dependencies")
	ErrExecNotFound         = fmt.Errorf("Unable to get current binary file")
)

// getServerURL we store http://.../#username to allow having many users managed in libsecret
func getServerURL(url, username string) string {
	return fmt.Sprintf("%s/#%s", strings.TrimRight(url, "/"), username)
}
