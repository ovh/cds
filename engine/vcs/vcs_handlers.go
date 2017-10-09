package vcs

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api"
)

func (s *Service) getAllVCSServersHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		servers := make([]ServerConfiguration, len(s.servers))
		copy(servers, s.servers)

		// Remove all secret data
		// TODO, this ugly, it would be nice to have a json tag to omit this
		for i := range servers {
			switch {
			case servers[i].Bitbucket != nil:
				servers[i].Bitbucket.ConsumerKey = ""
				servers[i].Bitbucket.PrivateKey = ""
			case servers[i].Github != nil:
				servers[i].Github.Secret = ""
			case servers[i].Gitlab != nil:
				servers[i].Gitlab.Secret = ""
			}
		}
		return api.WriteJSON(w, r, servers, http.StatusOK)
	}
}
