package ui

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/go-gorp/gorp"
	
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
)

// NavbarData is the struct returned by getUINavbarHandler
type NavbarData struct {
	Projects []NavbarProjectData `json:"projects"`
}

// NavbarProjectData is the sub struct returned by getUINavbarHandler
type NavbarProjectData struct {
	Key              string   `json:"key"`
	Name             string   `json:"name"`
	ApplicationNames []string `json:"application_names,omitempty"`
	WorkflowNames    []string `json:"workflow_names,omitempty"`
}

// LoadNavbarData returns just the needed data for the ui navbar
func LoadNavbarData(db gorp.SqlExecutor, store cache.Store, u *sdk.User) (data NavbarData, err error) {
	var query string
	var args []interface{}
	// Admin can gets all project
	// Users can gets only their projects

	if u == nil || u.Admin {
		query = `
		select project.projectkey, project.name, string_agg(application.name, ','), string_agg(workflow.name, ',')
		from project
		left outer join application on project.id = application.project_id
		left outer join workflow on project.id = workflow.project_id
		and project.id = workflow.project_id
		group by project.projectkey, project.name
		order by project.name`
	} else {
		query = `
		select project.projectkey, project.name, string_agg(application.name, ','), string_agg(workflow.name, ',')
		from project
		left outer join application on project.id = application.project_id
		left outer join workflow on project.id = workflow.project_id
		and project.id IN (
			SELECT project_group.project_id
			FROM project_group
			WHERE
				project_group.group_id = ANY(string_to_array($1, ',')::int[])
				OR
				$2 = ANY(string_to_array($1, ',')::int[])
		)
		group by project.projectkey, project.name
		order by project.name`

		var groupID string
		for i, g := range u.Groups {
			if i == 0 {
				groupID = fmt.Sprintf("%d", g.ID)
			} else {
				groupID += "," + fmt.Sprintf("%d", g.ID)
			}
		}
		args = []interface{}{groupID, group.SharedInfraGroup.ID}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return data, err
	}
	defer rows.Close()

	for rows.Next() {
		var key, name string
		var apps, workflows sql.NullString
		if err := rows.Scan(&key, &name, &apps, &workflows); err != nil {
			return data, err
		}

		var appNames = []string{}
		if apps.Valid {
			appNames = strings.Split(apps.String, ",")
			sort.Strings(appNames)
		}

		var workflowNames = []string{}
		if workflows.Valid {
			workflowNames = strings.Split(workflows.String, ",")
			sort.Strings(workflowNames)
		}

		data.Projects = append(data.Projects,
			NavbarProjectData{
				Key:              key,
				Name:             name,
				ApplicationNames: appNames,
				WorkflowNames:    workflowNames,
			},
		)
	}

	return data, nil
}
