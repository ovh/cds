package migrate

import (
	"context"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/sdk"
)

func MigrateProjectRepsositories(ctx context.Context, db *gorp.DbMap) error {
	repos, err := repository.LoadAllRepositories(ctx, db)
	if err != nil {
		return err
	}
	for _, r := range repos {
		tx, err := db.Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		r.Name = strings.ToLower(r.Name)
		if err := repository.Update(ctx, tx, &r); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}
	}
	return nil
}
