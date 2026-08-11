package hatchery

import (
	"fmt"
	"slices"
	"strings"

	"golang.org/x/crypto/ssh"
)

// CheckInjectSSHPublicKeys validates the SSH public keys a hatchery injects into
// the workers it spawns. Each entry must be a complete authorized_keys line
// carrying a from= option: the keys grant shell access to a machine running CI
// payloads, so they must state which source addresses may use them rather than
// being usable from anywhere.
func CheckInjectSSHPublicKeys(publicKeys []string) error {
	for _, publicKey := range publicKeys {
		_, _, options, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
		if err != nil {
			return fmt.Errorf("invalid public key %q: %w", publicKey, err)
		}
		if len(options) == 0 || !slices.ContainsFunc(options, func(o string) bool { return strings.HasPrefix(o, "from=") }) {
			return fmt.Errorf("invalid public key %q: from option is missing", publicKey)
		}
	}

	return nil
}
