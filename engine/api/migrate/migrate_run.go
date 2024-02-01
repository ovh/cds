package migrate

import (
	"context"
	"database/sql/driver"
	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/yaml"
)

type OldContext sdk.WorkflowRunContext

func (m OldContext) Value() (driver.Value, error) {
	j, err := yaml.Marshal(m)
	return j, sdk.WrapError(err, "cannot marshal OldContext")
}

func (m *OldContext) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.(string)
	if !ok {
		return sdk.WithStack(fmt.Errorf("type assertion .(string) failed (%T)", src))
	}
	err := yaml.Unmarshal([]byte(source), m)
	return sdk.WrapError(err, "cannot unmarshal OldContext")
}

func MigrateRunContextsToJSON(ctx context.Context, db *gorp.DbMap) error {
	runs, err := workflow_v2.LoadRunsUnsafe(ctx, db)
	if err != nil {
		return err
	}

	type OldContexts struct {
		ID          string     `db:"id"`
		OldContexts OldContext `db:"contexts"`
	}

	var oldContexts []OldContexts
	if _, err := db.Select(&oldContexts, "SELECT id, contexts FROM old_v2_workflow_run_context"); err != nil {
		return sdk.WithStack(err)
	}
	oldContextsMap := make(map[string]sdk.WorkflowRunContext)
	for _, c := range oldContexts {
		oldContextsMap[c.ID] = sdk.WorkflowRunContext(c.OldContexts)
	}
	for _, r := range runs {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		r.Contexts = oldContextsMap[r.ID]
		if err := workflow_v2.UpdateRun(ctx, tx, &r); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func MigrateRunRepositoryInfo(ctx context.Context, db *gorp.DbMap) error {
	runs, err := workflow_v2.LoadRunsUnsafe(ctx, db)
	if err != nil {
		return err
	}

	for _, r := range runs {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		vcs, err := vcs.LoadVCSByIDAndProjectKey(ctx, db, r.ProjectKey, r.VCSServerID)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		if vcs != nil {
			r.VCSServer = vcs.Name
		} else {
			r.VCSServer = r.VCSServerID
		}
		repo, err := repository.LoadRepositoryByVCSAndID(ctx, db, r.VCSServerID, r.RepositoryID)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		if repo != nil {
			r.Repository = repo.Name
		} else {
			r.Repository = r.RepositoryID
		}
		if err := workflow_v2.UpdateRun(ctx, tx, &r); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
