package project

import (
	"context"
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func LoadRbacProjectIDsByUserID(_ context.Context, db gorp.SqlExecutor, role string, userID string) ([]sdk.Project, error) {
	query := `
		WITH userRbac as (
			SELECT distinct(rpi.project_id) as id
			FROM rbac_project_ids rpi
			JOIN rbac_project rp ON rp.id = rpi.rbac_project_id AND rp.role = $1
			JOIN rbac_project_users rpu ON rpu.rbac_project_id = rp.id AND rpu.user_id = $2
		),
		groupRbac as (
			SELECT distinct(rpi.project_id) as id
			FROM rbac_project_ids rpi
			JOIN rbac_project rp ON rp.id = rpi.rbac_project_id AND rp.role = $1
			JOIN rbac_project_groups rpg ON rpg.rbac_project_id = rp.id
			JOIN "group" g ON g.id = rpg.group_id
			JOIN group_authentified_user gau ON gau.group_id = g.id AND gau.authentified_user_id = $2
		),
		userAllRbac as (
			SELECT distinct(p.id) as id
			FROM project p
			JOIN rbac_project_ids rpi ON rpi.project_id = p.id
			JOIN rbac_project rp ON rp.id = rpi.rbac_project_id AND rp.all = true AND rp.role = $1
			JOIN rbac_project_users rpu ON rpu.rbac_project_id = rp.id AND rpu.user_id = $2
		),
		groupAllRbac as (
			SELECT distinct(p.id) as id
			FROM project p
			JOIN rbac_project_ids rpi ON rpi.project_id = p.id
			JOIN rbac_project rp ON rp.id = rpi.rbac_project_id AND rp.role = $1 AND rp.all = true
			JOIN rbac_project_groups rpg ON rpg.rbac_project_id = rp.id
			JOIN "group" g ON g.id = rpg.group_id
			JOIN group_authentified_user gau ON gau.group_id = g.id AND gau.authentified_user_id = $2
		),
		concat as (
			SELECT distinct(id) as id FROM (
				SELECT id FROM userRbac UNION SELECT id FROM groupRbac UNION SELECT id FROM userAllRbac UNION SELECT id FROM groupAllRbac
			) tmp
		)
		SELECT p.* FROM concat c
		JOIN project p ON p.id = c.id`
	var projects []sdk.Project
	if _, err := db.Select(&projects, query, role, userID); err != nil {
		return nil, err
	}
	return projects, nil
}

func LoadProjectByRbacProject(ctx context.Context, db gorp.SqlExecutor, rbacProjectID int64) ([]sdk.ProjectIdentifiers, error) {
	query := `
		SELECT p.id, p.projectkey
		FROM rbac_project_ids rpi
		JOIN project p ON p.id = rpi.project_id
		WHERE rpi.rbac_project_id = $1`
	var res []sdk.ProjectIdentifiers
	if _, err := db.Select(&res, query, rbacProjectID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WithStack(err)
	}
	return res, nil
}
