package group

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func getLinksGroupProject(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadLinkGroupProjectOptionFunc) (LinksGroupProject, error) {
	gps := []*LinkGroupProject{}

	if err := gorpmapping.GetAll(ctx, db, q, &gps); err != nil {
		return nil, sdk.WrapError(err, "cannot links group project")
	}

	var pgps []*LinkGroupProject
	for i := range gps {
		isValid, err := gorpmapping.CheckSignature(gps[i], gps[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "project_group %d data corrupted", gps[i].ID)
			continue
		}
		pgps = append(pgps, gps[i])
	}

	if len(pgps) > 0 {
		for i := range opts {
			if err := opts[i](ctx, db, pgps...); err != nil {
				return nil, err
			}
		}
	}

	var result = make([]LinkGroupProject, len(pgps))
	for i := range pgps {
		result[i] = *pgps[i]
	}

	return result, nil
}

// LoadLinksGroupProjectForGroupID returns data from project_group table for given group id.
func LoadLinksGroupProjectForGroupID(ctx context.Context, db gorp.SqlExecutor, groupID int64, opts ...LoadLinkGroupProjectOptionFunc) (LinksGroupProject, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM project_group
		WHERE group_id = $1
	`).Args(groupID)
	return getLinksGroupProject(ctx, db, query, opts...)
}

// LoadLinksGroupProjectForProjectIDs returns data from project_group table for given group id.
func LoadLinksGroupProjectForProjectIDs(ctx context.Context, db gorp.SqlExecutor, projectIDs []int64, opts ...LoadLinkGroupProjectOptionFunc) (LinksGroupProject, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM project_group
		WHERE project_id = ANY(string_to_array($1, ',')::int[])
  `).Args(gorpmapping.IDsToQueryString(projectIDs))
	return getLinksGroupProject(ctx, db, query, opts...)
}

// LoadLinkGroupProjectForGroupIDAndProjectID returns a link from project_group if exists for given group and project ids.
func LoadLinkGroupProjectForGroupIDAndProjectID(ctx context.Context, db gorp.SqlExecutor, groupID, projectID int64) (*LinkGroupProject, error) {
	var l LinkGroupProject

	query := gorpmapping.NewQuery(`
    SELECT *
    FROM project_group
    WHERE group_id = $1 AND project_id = $2
  `).Args(groupID, projectID)

	found, err := gorpmapping.Get(ctx, db, query, &l)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get link between group and project")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := gorpmapping.CheckSignature(l, l.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "group.LoadLinkGroupProjectForGroupIDAndProjectID> project_group %d data corrupted", l.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	return &l, nil
}

// InsertLinkGroupProject inserts given link group-project into database.
func InsertLinkGroupProject(ctx context.Context, db gorpmapper.SqlExecutorWithTx, l *LinkGroupProject) error {
	return sdk.WrapError(gorpmapping.InsertAndSign(ctx, db, l), "unable to insert link between group and project")
}

// updateDBLinkGroupProject updates given link group-project into database.
func updateDBLinkGroupProject(ctx context.Context, db gorpmapper.SqlExecutorWithTx, l *LinkGroupProject) error {
	return sdk.WrapError(gorpmapping.UpdateAndSign(ctx, db, l), "unable to update link between group and project")
}

// deleteDBLinkGroupProject deletes given link group-project into database.
func deleteDBLinkGroupProject(ctx context.Context, db gorp.SqlExecutor, l *LinkGroupProject) error {
	return sdk.WrapError(gorpmapping.Delete(db, l), "unable to delete link between group and project")
}
