package api

import (
  "context"

  "github.com/go-gorp/gorp"

  "github.com/ovh/cds/engine/cache"
  "github.com/ovh/cds/sdk"
)

func (api *API) pluginRead(ctx context.Context, _ *sdk.AuthUserConsumer, _ cache.Store, _ gorp.SqlExecutor, _ map[string]string) error {
  // Old worker
  if isWorker(ctx) || getUserConsumer(ctx) != nil {
    return nil
  }
  // New worker
  if getWorker(ctx) != nil {
    return nil
  }

  return sdk.WithStack(sdk.ErrForbidden)
}
