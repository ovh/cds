package navbar

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
)

func LoadNavbarAsAdmin(db gorp.SqlExecutor, userID int64) (data []sdk.NavbarProjectData, err error) {
	query := `
	(
		SELECT DISTINCT
			project.projectkey, project.name AS project_name, project.description, NULL AS name,
			CASE
				WHEN (SELECT project_id FROM project_favorite WHERE user_id = $1 AND project_id = project.id) IS NOT NULL THEN true
				ELSE false
			END AS favorite,
			'project' AS type
		FROM project
		ORDER BY project.name
	)
	UNION
	(
		SELECT DISTINCT
			project.projectkey, project.name AS project_name, application.description, application.name,
			false AS favorite,
			'application' AS type
		FROM project
		JOIN application ON application.project_id = project.id
		ORDER BY project.name
	)
	UNION
	(
		SELECT DISTINCT
			project.projectkey, project.name AS project_name, workflow.description, workflow.name,
			CASE
				WHEN (SELECT workflow_id FROM workflow_favorite WHERE user_id = $1 AND workflow_id = workflow.id) IS NOT NULL THEN true
				ELSE false
			END AS favorite,
			'workflow' AS type
		FROM project
		JOIN workflow ON workflow.project_id = project.id
		ORDER BY project.name
	)
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return data, sdk.WithStack(err)
	}
	defer rows.Close()

	for rows.Next() {
		var key, projectName, projectDescription, eltType string
		var favorite bool
		var name sql.NullString
		if err := rows.Scan(&key, &projectName, &projectDescription, &name, &favorite, &eltType); err != nil {
			return data, sdk.WithStack(err)
		}

		projData := sdk.NavbarProjectData{
			Key:         key,
			Name:        projectName,
			Description: projectDescription,
			Favorite:    favorite,
			Type:        eltType,
		}

		if name.Valid {
			switch eltType {
			case "workflow":
				projData.WorkflowName = name.String
			case "application":
				projData.ApplicationName = name.String
			}
		}

		data = append(data, projData)
	}

	return data, nil
}

func LoadNavbarAsUser(db gorp.SqlExecutor, userID int64, groupIDs []int64) (data []sdk.NavbarProjectData, err error) {
	query := `
	(
		SELECT DISTINCT
			project.projectkey, project.name AS project_name, project.description, NULL AS name,
			CASE
				WHEN (SELECT project_id FROM project_favorite WHERE user_id = $1 AND project_id = project.id) IS NOT NULL THEN true
				ELSE false
			END AS favorite,
			'project' AS type
		FROM project
		WHERE project.id IN (
				SELECT project_group.project_id
				FROM project_group
				WHERE
					project_group.group_id = ANY(string_to_array($2, ',')::int[])
					OR
					$3 = ANY(string_to_array($2, ',')::int[])
			)
		ORDER BY project.name
	)
	UNION
	(
		SELECT DISTINCT
			project.projectkey, project.name  AS project_name, application.description, application.name,
			false AS favorite,
			'application' AS type
		FROM project
		JOIN application ON application.project_id = project.id
		WHERE project.id IN (
				SELECT project_group.project_id
				FROM project_group
				WHERE
					project_group.group_id = ANY(string_to_array($2, ',')::int[])
					OR
					$3 = ANY(string_to_array($2, ',')::int[])
			)
		ORDER BY project.name
	)
	UNION
	(
		SELECT DISTINCT
			project.projectkey, project.name AS project_name, workflow.description, workflow.name,
			CASE
				WHEN (SELECT workflow_id FROM workflow_favorite WHERE user_id = $1 AND workflow_id = workflow.id) IS NOT NULL THEN true
				ELSE false
			END AS favorite,
			'workflow' AS type
		FROM project
		JOIN workflow ON workflow.project_id = project.id
		WHERE project.id IN (
				SELECT project_group.project_id
				FROM project_group
				WHERE
					project_group.group_id = ANY(string_to_array($2, ',')::int[])
					OR
					$3 = ANY(string_to_array($2, ',')::int[])
			)
		ORDER BY project.name
	)
  `

	rows, err := db.Query(query, userID, gorpmapping.IDsToQueryString(groupIDs), group.SharedInfraGroup.ID)
	if err != nil {
		return data, sdk.WithStack(err)
	}
	defer rows.Close()

	for rows.Next() {
		var key, projectName, projectDescription, eltType string
		var favorite bool
		var name sql.NullString
		if err := rows.Scan(&key, &projectName, &projectDescription, &name, &favorite, &eltType); err != nil {
			return data, sdk.WithStack(err)
		}

		projData := sdk.NavbarProjectData{
			Key:         key,
			Name:        projectName,
			Description: projectDescription,
			Favorite:    favorite,
			Type:        eltType,
		}

		if name.Valid {
			switch eltType {
			case "workflow":
				projData.WorkflowName = name.String
			case "application":
				projData.ApplicationName = name.String
			}
		}

		data = append(data, projData)
	}

	return data, nil
}
