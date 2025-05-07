package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/service"
)

// configVCSGPGKeysHandler return all gpg public keys for all vcs server
func (api *API) configVCSGPGKeysHandler() ([]service.RbacChecker, service.Handler) {
	return nil,
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			keys := make(map[string][]sdk.Key)
			for vcsName, kSlice := range api.Config.VCS.GPGKeys {
				vcsKeys := make([]sdk.Key, 0, len(kSlice))
				for _, k := range kSlice {
					vcsK := sdk.Key{KeyID: k.ID, Public: k.PublicKey}
					vcsKeys = append(vcsKeys, vcsK)
				}
				keys[vcsName] = vcsKeys
			}
			return service.WriteJSON(w, keys, http.StatusOK)
		}
}
