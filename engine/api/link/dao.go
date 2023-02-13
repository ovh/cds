package link

import (
  "time"

  "github.com/go-gorp/gorp"
  "github.com/rockbears/log"
  "golang.org/x/net/context"

  "github.com/ovh/cds/engine/api/database/gorpmapping"
  "github.com/ovh/cds/engine/gorpmapper"
  "github.com/ovh/cds/sdk"
)

func get(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) (*sdk.UserLink, error) {
  var dbLink dbUserLink
  found, err := gorpmapping.Get(ctx, db, query, &dbLink)
  if err != nil {
    return nil, err
  }
  if !found {
    return nil, sdk.WithStack(sdk.ErrNotFound)
  }

  isValid, err := gorpmapping.CheckSignature(dbLink, dbLink.Signature)
  if err != nil {
    return nil, err
  }
  if !isValid {
    log.Error(ctx, "UserLink %d data corrupted", dbLink.ID)
    return nil, sdk.WithStack(sdk.ErrNotFound)
  }
  return &dbLink.UserLink, nil
}

func getAll(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.UserLink, error) {
  var dbLinks []dbUserLink
  if err := gorpmapping.GetAll(ctx, db, query, &dbLinks); err != nil {
    return nil, err
  }
  userLinks := make([]sdk.UserLink, 0, len(dbLinks))
  for _, dbL := range dbLinks {
    isValid, err := gorpmapping.CheckSignature(dbL, dbL.Signature)
    if err != nil {
      return nil, err
    }
    if !isValid {
      log.Error(ctx, "UserLinks %d data corrupted", dbL.ID)
      continue
    }
    userLinks = append(userLinks, dbL.UserLink)
  }
  return userLinks, nil
}

func Insert(ctx context.Context, db gorpmapper.SqlExecutorWithTx, uLink *sdk.UserLink) error {
  uLink.Created = time.Now()
  dbULink := &dbUserLink{UserLink: *uLink}
  return gorpmapping.InsertAndSign(ctx, db, dbULink)
}

func Update(ctx context.Context, db gorpmapper.SqlExecutorWithTx, uLink *sdk.UserLink) error {
  dbULink := &dbUserLink{UserLink: *uLink}
  return gorpmapping.UpdateAndSign(ctx, db, dbULink)
}

func LoadUserLinksByUserID(ctx context.Context, db gorp.SqlExecutor, userID string) ([]sdk.UserLink, error) {
  query := gorpmapping.NewQuery(`SELECT * FROM user_link WHERE authentified_user_id = $1`).Args(userID)
  return getAll(ctx, db, query)
}

func LoadUserLinkByTypeAndExternalID(ctx context.Context, db gorp.SqlExecutor, t string, externalID string) (*sdk.UserLink, error) {
  query := gorpmapping.NewQuery(`SELECT * FROM user_link WHERE type = $1 AND external_id = $2`).Args(t, externalID)
  return get(ctx, db, query)
}
