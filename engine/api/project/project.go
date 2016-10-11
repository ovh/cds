package project

import (
	"database/sql"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/template"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// funcpar contains the list of options to alter the behavior of a CRUD func
type funcpar struct {
	loadvariables bool
	loadapps      bool
	historylength int
}

// Mod is the type of all functionnal parameters
type Mod func(c *funcpar)

// WithVariables is a functionnal parameter usable in LoadProject
func WithVariables() Mod {
	f := func(c *funcpar) {
		c.loadvariables = true
	}

	return f
}

// WithApplications is a functionnal parameter usable in LoadProject
func WithApplications(historylength int) Mod {
	f := func(c *funcpar) {
		c.loadapps = true
		c.historylength = historylength
	}

	return f
}

// CreateFromWizard  Create a project from the creation wizard
func CreateFromWizard(db *sql.Tx, p *sdk.Project, u *sdk.User) error {

	// INSERT NEW PROJECT
	err := InsertProject(db, p)
	if err != nil {
		log.Warning("CreateFromWizard: Cannot insert project: %s\n", err)
		return err
	}

	// INSERT & CONFIGURE GROUP
	for i := range p.ProjectGroups {
		groupPermission := &p.ProjectGroups[i]

		// Insert group
		groupID, new, err := group.AddGroup(db, &groupPermission.Group)
		if groupID == 0 {
			if err == sdk.ErrInvalidGroupPattern {
				log.Warning("CreateFromWizard: Wrong group name: %s\n", err)
				return err
			}
			log.Warning("CreateFromWizard: Cannot add group: %s\n", err)
			return err
		}
		groupPermission.Group.ID = groupID

		// Add group on project
		err = group.InsertGroupInProject(db, p.ID, groupPermission.Group.ID, groupPermission.Permission)
		if err != nil {
			log.Warning("CreateFromWizard: Cannot add group %s in project %s:  %s\n", groupPermission.Group.Name, p.Name, err)
			return err
		}

		// Add user in the new group as admin
		if new {
			err = group.InsertUserInGroup(db, groupPermission.Group.ID, u.ID, true)
			if err != nil {
				log.Warning("CreateFromWizard: Cannot add user %s in group %s:  %s\n", u.Username, groupPermission.Group.Name, err)
				return err
			}
		}
	}

	for i := range p.Applications {
		// check application name pattern
		regexp := regexp.MustCompile(sdk.NamePattern)
		if !regexp.MatchString(p.Applications[i].Name) {
			log.Warning("CreateFromWizard: Application name %s do not respect pattern %s", p.Applications[i].Name, sdk.NamePattern)
			return sdk.ErrInvalidApplicationPattern
		}

		err = template.ApplyTemplate(db, p, &p.Applications[i])
		if err != nil {
			return err
		}
	}

	return nil
}

// LoadProjectByGroup loads all projects where group has access
func LoadProjectByGroup(db database.Querier, group *sdk.Group) error {
	query := `
		SELECT project.projectKey, project.name, project.last_modified, project_group.role 
		FROM project
	 	JOIN project_group ON project_group.project_id = project.id
	 	WHERE project_group.group_id = $1 
		ORDER BY project.name ASC`

	rows, err := db.Query(query, group.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var projectKey, projectName string
		var perm int
		var lastModified time.Time
		err = rows.Scan(&projectKey, &projectName, &lastModified, &perm)
		if err != nil {
			return err
		}
		group.ProjectGroups = append(group.ProjectGroups, sdk.ProjectGroup{
			Project: sdk.Project{
				Key:          projectKey,
				Name:         projectName,
				LastModified: lastModified.Unix(),
			},
			Permission: perm,
		})
	}
	return nil
}

// LoadProjectAndPipelineByPipelineActionID load project and pipeline by pipeline_action_id
func LoadProjectAndPipelineByPipelineActionID(db database.Querier, pipelineActionID int64) (sdk.Project, sdk.Pipeline, error) {
	query := `SELECT project.id, project.projectKey, project.last_modified, pipeline.id, pipeline.name
		  FROM pipeline_action
		  JOIN pipeline_stage ON pipeline_action.pipeline_stage_id = pipeline_stage.id
		  JOIN pipeline ON pipeline.id = pipeline_stage.pipeline_id
		  JOIN project ON project.id = pipeline.project_id
	          WHERE pipeline_action.id = $1`
	var proj sdk.Project
	var pip sdk.Pipeline
	var lastModified time.Time
	err := db.QueryRow(query, pipelineActionID).Scan(&proj.ID, &proj.Key, &lastModified, &pip.ID, &pip.Name)
	proj.LastModified = lastModified.Unix()
	return proj, pip, err
}

func loadprojectwithvariablesandappsandacitvities(db database.Querier, key string, limit int, user *sdk.User) (*sdk.Project, error) {
	query := `
	WITH load_apps AS (%s), load_vars AS (%s)
	SELECT *
	FROM (
		SELECT
			projid, projname, projlast_modified,
			NULL as varid, NULL as var_name, NULL as var_value, NULL as var_type,
			appid as appid, appname, applast_modified,
			envID, pipeline_id, pbid,
			envName, pipName, type,
			build_number, version, status,
			start, done,
			manual_trigger, triggered_by, parent_pipeline_build_id, vcs_changes_branch, vcs_changes_hash, vcs_changes_author,
			username, pipTriggerFrom, versionTriggerFrom
		FROM load_apps
		LEFT JOIN LATERAL (
			SELECT
				application_id, environment_id as envID, pipeline_id, id as pbid, envName, pipName, type,
				build_number, version, status,
				start, done,
				manual_trigger, triggered_by, parent_pipeline_build_id, vcs_changes_branch, vcs_changes_hash, vcs_changes_author,
				username, pipTriggerFrom, versionTriggerFrom
			  FROM ( (%s)  UNION (%s) ) as pb
			  ORDER BY start DESC
			  LIMIT $2
		) as pbh ON pbh.application_id = appid
	UNION
		SELECT
			projid, projname, projlast_modified,
			varid, var_name, var_value, var_type,
			NULL as appid, NULL as appname, current_timestamp as applast_modified,
			NULL as environment_id, NULL as pipeline_id, NULL as pbid,
			NULL as envName, NULL as pipName, NULL as type,
			NULL as build_number, NULL as version, NULL as status,
			NULL as start, NULL as done,
			NULL as manual_trigger, NULL as triggered_by, NULL as parent_pipeline_build_id, NULL as vcs_changes_branch, NULL as vcs_changes_hash, NULL as vcs_changes_author,
			NULL as username, NULL as pipTriggerFrom, NULL as versionTriggerFrom
		FROM load_vars
	) as p
	ORDER BY appname ASC, varid ASC, start DESC
	`

	var rows *sql.Rows
	var err error
	if user.Admin {
		query = fmt.Sprintf(query,
			application.LoadApplicationsRequestAdmin,
			loadProjectWithVariablesQuery,
			fmt.Sprintf(pipeline.LoadPipelineBuildRequest, "", "pb.application_id = appid AND project.projectkey = $1", "LIMIT $2"),
			fmt.Sprintf(pipeline.LoadPipelineHistoryRequest, "", "ph.application_id = appid AND project.projectkey = $1", "LIMIT $2"),
		)
		rows, err = db.Query(query, key, limit)
	} else {
		// $2 = user.ID in LoadApplicationsRequestNormalUser
		query = fmt.Sprintf(query,
			application.LoadApplicationsRequestNormalUser,
			loadProjectWithVariablesQuery,
			fmt.Sprintf(pipeline.LoadPipelineBuildRequest, "", "pb.application_id = appid AND project.projectkey = $1", "LIMIT $3"),
			fmt.Sprintf(pipeline.LoadPipelineHistoryRequest, "", "ph.application_id = appid AND project.projectkey = $1", "LIMIT $3"),
		)
		rows, err = db.Query(query, key, user.ID, limit)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNoProject
		}
		return nil, err
	}
	defer rows.Close()

	p := &sdk.Project{Key: key}
	var varname, appname, value, typ, envName, pipName, typePip, status, branch, hash, author, username, pipTriggerFrom sql.NullString
	var varid, appid, envID, pipID, pbID, buildNumber, triggeredBy, parentPbID, version, versionTriggerFrom sql.NullInt64
	var start, done pq.NullTime
	var manualTrigger sql.NullBool
	var currentApp int
	var lastModified time.Time
	var appLastModified time.Time
	var apps []sdk.Application
	for rows.Next() {
		err = rows.Scan(&p.ID, &p.Name, &lastModified, &varid, &varname, &value, &typ, &appid, &appname, &appLastModified,
			&envID, &pipID, &pbID,
			&envName, &pipName, &typePip,
			&buildNumber, &version, &status,
			&start, &done,
			&manualTrigger, &triggeredBy, &parentPbID, &branch, &hash, &author,
			&username, &pipTriggerFrom, &versionTriggerFrom)
		if err != nil {
			return nil, err
		}
		p.LastModified = lastModified.Unix()
		if varid.Valid && varname.Valid && typ.Valid {
			par := sdk.Variable{
				ID:   varid.Int64,
				Name: varname.String,
				Type: sdk.VariableTypeFromString(typ.String),
			}

			if sdk.NeedPlaceholder(par.Type) {
				par.Value = sdk.PasswordPlaceholder
			} else if value.Valid {
				par.Value = value.String
			}

			p.Variable = append(p.Variable, par)
		}

		if appid.Valid && appname.Valid {
			// Ah, we switched to another app
			if len(apps) == 0 || appid.Int64 != apps[currentApp].ID {
				if len(apps) != 0 {
					currentApp++
				}

				app := sdk.Application{
					ID:           appid.Int64,
					Name:         appname.String,
					LastModified: appLastModified.Unix(),
				}
				apps = append(apps, app)
			}

			if pbID.Valid && start.Valid && done.Valid && buildNumber.Valid && version.Valid && status.Valid {
				var pb sdk.PipelineBuild
				pb.ID = pbID.Int64

				if envName.Valid && envID.Valid {
					pb.Environment.Name = envName.String
					pb.Environment.ID = envID.Int64
				}
				if pipName.Valid && typePip.Valid && pipID.Valid {
					pb.Pipeline.Name = pipName.String
					pb.Pipeline.Type = sdk.PipelineTypeFromString(typePip.String)
					pb.Pipeline.ID = pipID.Int64
				}

				pb.BuildNumber = buildNumber.Int64
				pb.Version = version.Int64
				pb.Status = sdk.StatusFromString(status.String)
				pb.Start = start.Time
				pb.Done = done.Time

				if manualTrigger.Valid {
					pb.Trigger.ManualTrigger = manualTrigger.Bool
				}

				if branch.Valid {
					pb.Trigger.VCSChangesBranch = branch.String
				}

				if hash.Valid && author.Valid {
					pb.Trigger.VCSChangesHash = hash.String
					pb.Trigger.VCSChangesAuthor = author.String
				}

				if username.Valid && triggeredBy.Valid {
					pb.Trigger.TriggeredBy = &sdk.User{
						Username: username.String,
						ID:       triggeredBy.Int64,
					}
				}

				if pipTriggerFrom.Valid && versionTriggerFrom.Valid && parentPbID.Valid {
					pb.Trigger.ParentPipelineBuild = &sdk.PipelineBuild{
						Pipeline: sdk.Pipeline{
							Name: pipTriggerFrom.String,
						},
						ID:      parentPbID.Int64,
						Version: versionTriggerFrom.Int64,
					}
				}

				apps[currentApp].PipelinesBuild = append(apps[currentApp].PipelinesBuild, pb)
			}
		}
	}

	p.Applications = apps

	return p, nil
}

func loadprojectwithvariablesandapps(db database.Querier, key string, user *sdk.User) (*sdk.Project, error) {
	query := `
	WITH load_apps AS (%s), load_vars AS (%s)
	SELECT *
	FROM (
		SELECT
			projid, projname, projlast_modified,
                        NULL as varid, NULL as var_name, NULL as var_value, NULL as var_type,
                        appid as appid, appname
		FROM load_apps
	UNION
		SELECT
			projid, projname,projlast_modified,
			varid, var_name, var_value, var_type,
			NULL as appid, NULL as appname
		FROM load_vars
	) as p
	ORDER BY appname ASC, varid ASC
	`

	var rows *sql.Rows
	var err error
	if user.Admin {
		query = fmt.Sprintf(query,
			application.LoadApplicationsRequestAdmin,
			loadProjectWithVariablesQuery,
		)
		rows, err = db.Query(query, key)
	} else {
		query = fmt.Sprintf(query,
			application.LoadApplicationsRequestNormalUser,
			loadProjectWithVariablesQuery,
		)
		rows, err = db.Query(query, key, user.ID)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNoProject
		}
		return nil, err
	}
	defer rows.Close()

	p := &sdk.Project{Key: key}
	var varname, appname, value, typ sql.NullString
	var varid, appid sql.NullInt64
	var lastModified time.Time
	var currentApp int

	var apps []sdk.Application
	for rows.Next() {
		err = rows.Scan(&p.ID, &p.Name, &lastModified, &varid, &varname, &value, &typ, &appid, &appname)
		if err != nil {
			return nil, err
		}
		p.LastModified = lastModified.Unix()
		if varid.Valid && varname.Valid && typ.Valid {
			par := sdk.Variable{
				ID:   varid.Int64,
				Name: varname.String,
				Type: sdk.VariableTypeFromString(typ.String),
			}

			if sdk.NeedPlaceholder(par.Type) {
				par.Value = sdk.PasswordPlaceholder
			} else if value.Valid {
				par.Value = value.String
			}

			p.Variable = append(p.Variable, par)
		}

		if appid.Valid && appname.Valid {

			// Ah, we switched to another app
			if len(apps) == 0 || appid.Int64 != apps[currentApp].ID {
				if len(apps) != 0 {
					currentApp++
				}

				app := sdk.Application{
					ID:   appid.Int64,
					Name: appname.String,
				}
				apps = append(apps, app)
			}
		}
	}

	p.Applications = apps

	return p, nil
}

const loadProjectWithVariablesQuery = `
	SELECT project.id as projid, project.name as projname, project.last_modified as projlast_modified, project_variable.id as varid, project_variable.var_name,
	project_variable.var_value, project_variable.var_type
	FROM project LEFT JOIN project_variable ON project_variable.project_id = project.id
	WHERE project.projectkey = $1
`

func loadprojectwithvariables(db database.Querier, key string) (*sdk.Project, error) {
	query := loadProjectWithVariablesQuery

	rows, err := db.Query(query, key)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNoProject
		}
		return nil, err
	}
	defer rows.Close()

	p := &sdk.Project{Key: key}
	var name, value, typ sql.NullString
	var id sql.NullInt64
	var lastModified time.Time
	for rows.Next() {
		err = rows.Scan(&p.ID, &p.Name, &lastModified, &id, &name, &value, &typ)
		if err != nil {
			return nil, err
		}
		p.LastModified = lastModified.Unix()
		if id.Valid && name.Valid && typ.Valid {
			par := sdk.Variable{
				ID:   id.Int64,
				Name: name.String,
				Type: sdk.VariableTypeFromString(typ.String),
			}

			if sdk.NeedPlaceholder(par.Type) {
				par.Value = sdk.PasswordPlaceholder
			} else if value.Valid {
				par.Value = value.String
			}

			p.Variable = append(p.Variable, par)
		}
	}

	return p, nil
}

func loadproject(db database.Querier, key string) (*sdk.Project, error) {
	query := `SELECT project.id, project.name, project.last_modified FROM project WHERE project.projectKey = $1`
	var name string
	var id int64
	var lastModified time.Time
	err := db.QueryRow(query, key).Scan(&id, &name, &lastModified)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNoProject
		}
		return nil, err
	}

	// Load project
	p := sdk.NewProject(key)
	p.Name = name
	p.ID = id
	p.LastModified = lastModified.Unix()
	return p, nil
}

// LoadProject loads an project from database
func LoadProject(db database.Querier, key string, user *sdk.User, mods ...Mod) (*sdk.Project, error) {
	var c funcpar
	for _, f := range mods {
		f(&c)
	}

	var p *sdk.Project
	var err error

	if c.loadvariables && c.loadapps {
		if c.historylength > 0 {
			p, err = loadprojectwithvariablesandappsandacitvities(db, key, c.historylength, user)
		} else {
			p, err = loadprojectwithvariablesandapps(db, key, user)
		}

	} else if c.loadvariables {
		p, err = loadprojectwithvariables(db, key)
	} else {
		p, err = loadproject(db, key)
	}

	if err != nil {
		return nil, err
	}

	if c.loadapps {
		for i := range p.Applications {
			pipelines, err := application.GetAllPipelinesByID(db, p.Applications[i].ID)
			if err != nil && err != sdk.ErrNoAttachedPipeline {
				return nil, err
			}
			p.Applications[i].Pipelines = pipelines
		}
	}

	return p, nil
}

// LoadProjectByPipelineID loads an project from pipeline iD
func LoadProjectByPipelineID(db database.Querier, pipelineID int64) (*sdk.Project, error) {
	query := `SELECT project.id, project.name, project.projectKey, project.last_modified
	          FROM project
	          JOIN pipeline ON pipeline.project_id = projecT.id
	          WHERE pipeline.id = $1 `
	var projectData sdk.Project
	var lastModified time.Time
	err := db.QueryRow(query, pipelineID).Scan(&projectData.ID, &projectData.Name, &projectData.Key, &lastModified)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNoProject
		}
		return nil, err
	}
	projectData.LastModified = lastModified.Unix()
	return &projectData, nil
}

// Exist checks whether a project exists or not
func Exist(db database.Querier, projectKey string) (bool, error) {
	query := `SELECT COUNT(id) FROM project WHERE project.projectKey = $1`

	var nb int64
	err := db.QueryRow(query, projectKey).Scan(&nb)
	if err != nil {
		return false, err
	}
	if nb != 0 {
		return true, nil
	}
	return false, nil
}

// LoadAllProjects load all projects from database
func LoadAllProjects(db database.Querier) ([]*sdk.Project, error) {
	projects := []*sdk.Project{}

	var query string
	var err error
	var rows *sql.Rows

	query = `SELECT project.id, project.projectKey, project.name, project.last_modified
			  FROM project
			  ORDER by project.name, project.projectkey ASC`
	rows, err = db.Query(query)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var key, name string
		var lastModified time.Time
		rows.Scan(&id, &key, &name, &lastModified)
		p := sdk.NewProject(key)
		p.Name = name
		p.ID = id
		p.LastModified = lastModified.Unix()
		projects = append(projects, p)
	}
	return projects, nil
}

// LoadProjects load all projects from database
func LoadProjects(db database.Querier, user *sdk.User) ([]*sdk.Project, error) {
	if user.Admin {
		return LoadAllProjects(db)
	}

	projects := []*sdk.Project{}

	query := `SELECT distinct(project.id), project.projectKey,project.name, project.last_modified
			  FROM project
			  JOIN project_group ON project.id = project_group.project_id
			  JOIN group_user ON project_group.group_id = group_user.group_id
			  WHERE group_user.user_id = $1
			  ORDER by project.name, project.projectkey ASC`
	rows, err := db.Query(query, user.ID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var key, name string
		var lastModified time.Time
		rows.Scan(&id, &key, &name, &lastModified)
		p := sdk.NewProject(key)
		p.Name = name
		p.ID = id
		p.LastModified = lastModified.Unix()
		projects = append(projects, p)
	}
	return projects, nil
}

// InsertProject insert given project into given database
func InsertProject(db database.QueryExecuter, p *sdk.Project) error {
	if p.Name == "" {
		return sdk.ErrInvalidName
	}
	query := `INSERT INTO project (projectKey, name) VALUES($1,$2) RETURNING id`
	err := db.QueryRow(query, p.Key, p.Name).Scan(&p.ID)
	return err
}

// UpdateProjectDB set new project name in database
func UpdateProjectDB(db database.Querier, projectKey, projectName string) (time.Time, error) {
	var lastModified time.Time
	query := `UPDATE project SET name=$1, last_modified=current_timestamp WHERE projectKey=$2 RETURNING last_modified`
	err := db.QueryRow(query, projectName, projectKey).Scan(&lastModified)
	return lastModified, err
}

// DeleteProject removes given project from database (project and project_group table)
// DeleteProject also removes all pipelines inside project (pipeline and pipeline_group table).
func DeleteProject(db database.QueryExecuter, key string) error {
	var projectID int64
	query := `SELECT id FROM project WHERE projectKey = $1`
	err := db.QueryRow(query, key).Scan(&projectID)
	if err != nil {
		return err
	}

	err = group.DeleteGroupProjectByProject(db, projectID)
	if err != nil {
		return err
	}

	err = DeleteAllVariableFromProject(db, projectID)
	if err != nil {
		return err
	}

	err = environment.DeleteAllEnvironment(db, projectID)
	if err != nil {
		return err
	}

	query = `DELETE FROM project WHERE project.id = $1`
	_, err = db.Exec(query, projectID)
	if err != nil {
		return err
	}

	return nil
}

//LastUpdates returns projects and application last update
func LastUpdates(db database.Querier, user *sdk.User, since time.Time) ([]sdk.ProjectLastUpdates, error) {
	query := `
		SELECT 	project.projectkey, project.last_modified, apps.name, apps.last_modified, pipelines.name, pipelines.last_modified
		FROM 	project	
		JOIN    project_group ON project_group.project_id = project.id
		JOIN    group_user ON project_group.group_id = group_user.group_id
		LEFT OUTER JOIN (
			SELECT 	application.project_id, application.name, application.last_modified
			FROM 	application, application_group, group_user
			WHERE   application.id = application_group.application_id
			AND 	application_group.group_id = group_user.group_id
			AND 	group_user.user_id = $1
			AND		application.last_modified >= $2
		) apps ON apps.project_id = project.id
		LEFT OUTER JOIN (
			SELECT 	pipeline.project_id, pipeline.name, pipeline.last_modified
			FROM 	pipeline, pipeline_group, group_user
			WHERE   pipeline.id = pipeline_group.pipeline_id
			AND 	pipeline_group.group_id = group_user.group_id
			AND 	group_user.user_id = $1
			AND		pipeline.last_modified >= $2
		) pipelines ON pipelines.project_id = project.id
		WHERE 	group_user.user_id = $1
		AND 	project.last_modified >= $2
		ORDER	by project.projectkey asc
	`
	rows, err := db.Query(query, user.ID, since)
	if err != nil {
		return []sdk.ProjectLastUpdates{}, err
	}
	defer rows.Close()

	res := []sdk.ProjectLastUpdates{}

	mapRes := map[string]*sdk.ProjectLastUpdates{}

	chanProj := make(chan struct {
		key          string
		lastModified time.Time
	})
	chanApp := make(chan struct {
		key          string
		name         string
		lastModified time.Time
	})
	chanPip := make(chan struct {
		key          string
		name         string
		lastModified time.Time
	})

	wg := &sync.WaitGroup{}
	quit := make(chan int)
	go func() {
		for {
			select {
			case <-quit:
				return
			case proj := <-chanProj:
				r := mapRes[proj.key]
				if r == nil {
					mapRes[proj.key] = &sdk.ProjectLastUpdates{
						Key:          proj.key,
						LastModified: proj.lastModified.Unix(),
					}
					r = mapRes[proj.key]
				}
				r.LastModified = proj.lastModified.Unix()
				wg.Done()
			case app := <-chanApp:
				r := mapRes[app.key]
				if r == nil {
					mapRes[app.key] = &sdk.ProjectLastUpdates{
						Key: app.key,
					}
					r = mapRes[app.key]
				}
				var appExists bool
				for _, a := range r.Applications {
					if app.name == a.Name {
						appExists = true
						break
					}
				}
				if !appExists {
					r.Applications = append(r.Applications, struct {
						Name         string `json:"name"`
						LastModified int64  `json:"last_modified"`
					}{
						Name:         app.name,
						LastModified: app.lastModified.Unix(),
					})
				}
				wg.Done()
			case pip := <-chanPip:
				r := mapRes[pip.key]
				if r == nil {
					mapRes[pip.key] = &sdk.ProjectLastUpdates{
						Key: pip.key,
					}
					r = mapRes[pip.key]
				}
				var pipExists bool
				for _, p := range r.Pipelines {
					if pip.name == p.Name {
						pipExists = true
						break
					}
				}
				if !pipExists {
					r.Pipelines = append(r.Pipelines, struct {
						Name         string `json:"name"`
						LastModified int64  `json:"last_modified"`
					}{
						Name:         pip.name,
						LastModified: pip.lastModified.Unix(),
					})
				}
				wg.Done()
			}
		}
	}()

	for rows.Next() {
		var projectKey, appName, pipName sql.NullString
		var projectLastModified, appLastModified, pipLastModified pq.NullTime
		err := rows.Scan(&projectKey, &projectLastModified, &appName, &appLastModified, &pipName, &pipLastModified)
		if err != nil {
			log.Warning("LastUpdates> Error scanning values: %s", err)
			continue
		}

		wg.Add(1)
		chanProj <- struct {
			key          string
			lastModified time.Time
		}{
			projectKey.String,
			projectLastModified.Time,
		}

		if appName.Valid && appLastModified.Valid {
			wg.Add(1)
			chanApp <- struct {
				key          string
				name         string
				lastModified time.Time
			}{
				projectKey.String,
				appName.String,
				appLastModified.Time,
			}
		}

		if pipName.Valid && pipLastModified.Valid {
			wg.Add(1)
			chanPip <- struct {
				key          string
				name         string
				lastModified time.Time
			}{
				projectKey.String,
				pipName.String,
				pipLastModified.Time,
			}
		}
	}

	wg.Wait()
	quit <- 1
	close(chanProj)
	close(chanApp)
	close(chanPip)
	for _, v := range mapRes {
		res = append(res, *v)
	}

	return res, nil
}
