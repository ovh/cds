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
		select distinct project.projectkey, project.name, applications.names, workflows.names
		from project
		left outer join (
			select project_id, string_agg(name, ',') as "names"
			from application
			group by project_id
		) as "applications"  on project.id = applications.project_id
		left outer join (
			select project_id, string_agg(name, ',') as "names"
			from workflow
			group by project_id
		) as "workflows"  on project.id = workflows.project_id
		order by project.name`
	} else {
		query = `
		select distinct project.projectkey, project.name, applications.names, workflows.names
		from project
		left outer join (
			select project_id, string_agg(name, ',') as "names"
			from application
			group by project_id
		) as "applications"  on project.id = applications.project_id
		left outer join (
			select project_id, string_agg(name, ',') as "names"
			from workflow
			group by project_id
		) as "workflows"  on project.id = workflows.project_id
		where project.id IN (
			SELECT project_group.project_id
			FROM project_group
			WHERE
				project_group.group_id = ANY(string_to_array($1, ',')::int[])
				OR
				$2 = ANY(string_to_array($1, ',')::int[])
		)
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

	fmt.Println("Narvar: ", query, args)
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

		fmt.Println(key, name, apps, workflows)

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
