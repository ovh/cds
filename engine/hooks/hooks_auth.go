package hooks

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/sdk"
)

func (s *Service) authMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *api.HandlerConfig) (context.Context, error) {
	if rc.Options["auth"] != "true" {
		return ctx, nil
	}

	hash, err := base64.StdEncoding.DecodeString(req.Header.Get(sdk.AuthHeader))
	if err != nil {
		return ctx, fmt.Errorf("bad header syntax: %s", err)
	}

	if s.hash == string(hash) {
		return ctx, nil
	}

	return ctx, sdk.ErrUnauthorized
}
