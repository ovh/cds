package group

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// LoadLinksGroupProjectForGroupID returns data from project_group table for given group id.
func LoadLinksGroupProjectForGroupID(ctx context.Context, db gorp.SqlExecutor, groupID int64) (LinksGroupProject, error) {
	ls := []LinkGroupProject{}

	query := gorpmapping.NewQuery(`
		SELECT *
		FROM project_group
		WHERE group_id = $1
	`).Args(groupID)

	if err := gorpmapping.GetAll(ctx, db, query, &ls); err != nil {
		return nil, sdk.WrapError(err, "cannot get links between group %d and projects", groupID)
	}

	var result []LinkGroupProject
	for _, l := range ls {
		isValid, err := gorpmapping.CheckSignature(l, l.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "group.LoadLinksGroupProjectForGroupID> project_group %d data corrupted", l.ID)
			continue
		}
		result = append(result, l)
	}

	return result, nil
}

// LoadLinksGroupProjectForProjectIDs returns data from project_group table for given group id.
func LoadLinksGroupProjectForProjectIDs(ctx context.Context, db gorp.SqlExecutor, projectIDs []int64) (LinksGroupProject, error) {
	ls := []LinkGroupProject{}

	query := gorpmapping.NewQuery(`
		SELECT *
		FROM project_group
		WHERE project_id = ANY(string_to_array($1, ',')::int[])
  `).Args(gorpmapping.IDsToQueryString(projectIDs))

	if err := gorpmapping.GetAll(ctx, db, query, &ls); err != nil {
		return nil, sdk.WrapError(err, "cannot get links between group and project")
	}

	var result []LinkGroupProject
	for _, l := range ls {
		isValid, err := gorpmapping.CheckSignature(l, l.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "group.LoadLinksGroupProjectForProjectIDs> project_group %d data corrupted", l.ID)
			continue
		}
		result = append(result, l)
	}

	return result, nil
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
func InsertLinkGroupProject(ctx context.Context, db gorp.SqlExecutor, l *LinkGroupProject) error {
	return sdk.WrapError(gorpmapping.InsertAndSign(ctx, db, l), "unable to insert link between group and project")
}

// updateDBLinkGroupProject updates given link group-project into database.
func updateDBLinkGroupProject(ctx context.Context, db gorp.SqlExecutor, l *LinkGroupProject) error {
	return sdk.WrapError(gorpmapping.UpdateAndSign(ctx, db, l), "unable to update link between group and project")
}

// deleteDBLinkGroupProject deletes given link group-project into database.
func deleteDBLinkGroupProject(ctx context.Context, db gorp.SqlExecutor, l *LinkGroupProject) error {
	return sdk.WrapError(gorpmapping.Delete(db, l), "unable to delete link between group and project")
}
