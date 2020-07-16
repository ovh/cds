package repositoriesmanager

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gorpmapping"
	"github.com/ovh/cds/sdk/log"
)

func init() {
	gorpmapping.Register(
		gorpmapping.New(dbProjectVCSServerLink{}, "project_vcs_server_link", true, "id"),
		gorpmapping.New(dbProjectVCSServerLinkData{}, "project_vcs_server_link_data", true, "id"),
	)
}

type dbProjectVCSServerLink struct {
	gorpmapping.SignedEntity
	sdk.ProjectVCSServerLink
}

func (e dbProjectVCSServerLink) Canonical() gorpmapping.CanonicalForms {
	var _ = []interface{}{e.ID, e.Name, e.ProjectID, e.VCSProject}
	return gorpmapping.CanonicalForms{
		"{{.ID}}{{.Name}}{{.ProjectID}}{{.VCSProject}}",
	}
}

type dbProjectVCSServerLinkData struct {
	gorpmapping.SignedEntity
	sdk.ProjectVCSServerLinkData
}

func (e dbProjectVCSServerLinkData) Canonical() gorpmapping.CanonicalForms {
	var _ = []interface{}{e.ID, e.ProjectVCSServerLinkID, e.Key}
	return gorpmapping.CanonicalForms{
		"{{.ID}}{{.ProjectVCSServerLinkID}}{{.Key}}",
	}
}

func InsertProjectVCSServerLink(ctx context.Context, db gorpmapping.SqlExecutorWithTx, l *sdk.ProjectVCSServerLink) error {
	var dbProjectVCSServerLink = dbProjectVCSServerLink{ProjectVCSServerLink: *l}
	if err := gorpmapping.InsertAndSign(ctx, db, &dbProjectVCSServerLink); err != nil {
		return err
	}
	*l = dbProjectVCSServerLink.ProjectVCSServerLink

	for i := range l.ProjectVCSServerLinkData {
		data := &l.ProjectVCSServerLinkData[i]
		data.ProjectVCSServerLinkID = l.ID
		dbData := dbProjectVCSServerLinkData{ProjectVCSServerLinkData: *data}
		if err := gorpmapping.InsertAndSign(ctx, db, &dbData); err != nil {
			return err
		}
		*data = dbData.ProjectVCSServerLinkData
	}

	return nil
}

func UpdateProjectVCSServerLink(ctx context.Context, db gorpmapping.SqlExecutorWithTx, l *sdk.ProjectVCSServerLink) error {
	var dbProjectVCSServerLink = dbProjectVCSServerLink{ProjectVCSServerLink: *l}
	if err := gorpmapping.UpdateAndSign(ctx, db, &dbProjectVCSServerLink); err != nil {
		return err
	}
	*l = dbProjectVCSServerLink.ProjectVCSServerLink

	for i := range l.ProjectVCSServerLinkData {
		data := &l.ProjectVCSServerLinkData[i]
		dbData := dbProjectVCSServerLinkData{ProjectVCSServerLinkData: *data}
		if dbData.ID == 0 {
			if err := gorpmapping.InsertAndSign(ctx, db, &dbData); err != nil {
				return err
			}
		} else {
			if err := gorpmapping.UpdateAndSign(ctx, db, &dbData); err != nil {
				return err
			}
		}
		*data = dbData.ProjectVCSServerLinkData
	}

	return nil
}

func DeleteProjectVCSServerLink(ctx context.Context, db gorp.SqlExecutor, l *sdk.ProjectVCSServerLink) error {
	var dbProjectVCSServerLink = dbProjectVCSServerLink{ProjectVCSServerLink: *l}
	if err := gorpmapping.Delete(db, &dbProjectVCSServerLink); err != nil {
		return err
	}
	return nil
}

func getAllProjectVCSServerLinks(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.ProjectVCSServerLink, error) {
	var res []dbProjectVCSServerLink
	if err := gorpmapping.GetAll(ctx, db, query, &res); err != nil {
		return nil, err
	}
	links := make([]sdk.ProjectVCSServerLink, len(res))
	for i := range res {
		isValid, err := gorpmapping.CheckSignature(res[i], res[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "repostoriesmanager. getAllProjectVCSServerLinks> vcs_server_project_link %d data corrupted", res[i].ID)
			continue
		}

		a := res[i].ProjectVCSServerLink
		links[i] = a
	}
	return links, nil
}

func LoadAllProjectVCSServerLinksByProjectID(ctx context.Context, db gorp.SqlExecutor, projectID int64) ([]sdk.ProjectVCSServerLink, error) {
	var query = gorpmapping.NewQuery(`
	SELECT *
	FROM project_vcs_server_link
	WHERE project_id = $1
	ORDER BY name ASC
	`).Args(projectID)
	return getAllProjectVCSServerLinks(ctx, db, query)
}

func LoadAllProjectVCSServerLinksByProjectKey(ctx context.Context, db gorp.SqlExecutor, projectKey string) ([]sdk.ProjectVCSServerLink, error) {
	var query = gorpmapping.NewQuery(`
	SELECT project_vcs_server_link.*
	FROM project_vcs_server_link
	JOIN project on project.id = project_vcs_server_link.project_id
	WHERE project.projectkey = $1	ORDER BY project_vcs_server_link.name ASC
	`).Args(projectKey)
	return getAllProjectVCSServerLinks(ctx, db, query)
}

func LoadProjectVCSServerLinkByProjectKeyAndVCSServerName(ctx context.Context, db gorp.SqlExecutor, projectKey, rmName string) (sdk.ProjectVCSServerLink, error) {
	var query = gorpmapping.NewQuery(`
	SELECT project_vcs_server_link.*
	FROM project_vcs_server_link
	JOIN project on project.id = project_vcs_server_link.project_id
	WHERE project.projectkey = $1
	AND project_vcs_server_link.name = $2
	`).Args(projectKey, rmName)
	var res dbProjectVCSServerLink
	found, err := gorpmapping.Get(ctx, db, query, &res)
	if err != nil {
		return sdk.ProjectVCSServerLink{}, err
	}
	if !found {
		return sdk.ProjectVCSServerLink{}, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := gorpmapping.CheckSignature(res, res.Signature)
	if err != nil {
		return sdk.ProjectVCSServerLink{}, err
	}
	if !isValid {
		log.Error(ctx, "repostoriesmanager.LoadProjectVCSServerLinkByProjectKeyAndVCSServerName> vcs_server_project_link %d data corrupted", res.ID)
		return sdk.ProjectVCSServerLink{}, sdk.WithStack(sdk.ErrNotFound)
	}

	data, err := LoadProjectVCSServerLinksData(ctx, db, res.ID)
	if err != nil {
		return sdk.ProjectVCSServerLink{}, err
	}
	res.ProjectVCSServerLinkData = data

	return res.ProjectVCSServerLink, nil
}

func LoadProjectVCSServerLinksData(ctx context.Context, db gorp.SqlExecutor, projectVCSServerLinkID int64, opts ...gorpmapping.GetOptionFunc) ([]sdk.ProjectVCSServerLinkData, error) {
	var query = gorpmapping.NewQuery(`
		SELECT *
		FROM project_vcs_server_link_data
		WHERE project_vcs_server_link_id = $1
		`).Args(projectVCSServerLinkID)
	var res []dbProjectVCSServerLinkData
	if err := gorpmapping.GetAll(ctx, db, query, &res, opts...); err != nil {
		return nil, err
	}
	data := make([]sdk.ProjectVCSServerLinkData, len(res))
	for i := range res {
		isValid, err := gorpmapping.CheckSignature(res[i], res[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "repostoriesmanager.LoadProjectVCSServerLinksData> vcs_server_project_link_data %d data corrupted", res[i].ID)
			continue
		}

		a := res[i].ProjectVCSServerLinkData
		data[i] = a
	}
	return data, nil
}
