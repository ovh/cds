package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/service"
)

// configVCSGPGKeysHandler return all gpg public keys for all vcs server
func (api *API) configVCSGPGKeysHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBACNone(),
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

func (api *API) configV2CDNHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBACNone(),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			tcpURL, tcpURLEnableTLS, err := services.GetCDNPublicTCPAdress(ctx, api.mustDB())
			if err != nil {
				return err
			}
			httpURL, err := services.GetCDNPublicHTTPAdress(ctx, api.mustDB())
			if err != nil {
				return err
			}
			return service.WriteJSON(w,
				sdk.CDNConfig{TCPURL: tcpURL,
					TCPURLEnableTLS: tcpURLEnableTLS,
					HTTPURL:         httpURL},
				http.StatusOK)
		}
}
