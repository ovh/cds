package project

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/keys"
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

// LoadProjectByGroup loads all projects where group has access
func LoadProjectByGroup(db gorp.SqlExecutor, group *sdk.Group) error {
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
				LastModified: lastModified,
			},
			Permission: perm,
		})
	}
	return nil
}

// LoadProjectByPipelineActionID load project and pipeline by pipeline_action_id
func LoadProjectByPipelineActionID(db gorp.SqlExecutor, pipelineActionID int64) (sdk.Project, error) {
	query := `SELECT project.id, project.projectKey, project.last_modified
		  FROM pipeline_action
		  JOIN pipeline_stage ON pipeline_action.pipeline_stage_id = pipeline_stage.id
		  JOIN pipeline ON pipeline.id = pipeline_stage.pipeline_id
		  JOIN project ON project.id = pipeline.project_id
	          WHERE pipeline_action.id = $1`
	var proj sdk.Project
	var lastModified time.Time
	err := db.QueryRow(query, pipelineActionID).Scan(&proj.ID, &proj.Key, &lastModified)
	proj.LastModified = lastModified
	return proj, err
}

func loadprojectwithvariablesandapps(db gorp.SqlExecutor, key string, user *sdk.User) (*sdk.Project, error) {
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
		p.LastModified = lastModified
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

func loadprojectwithvariables(db gorp.SqlExecutor, key string) (*sdk.Project, error) {
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
		p.LastModified = lastModified
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

func loadproject(db gorp.SqlExecutor, key string) (*sdk.Project, error) {
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
	p.LastModified = lastModified
	return p, nil
}

// Load loads an project from database
func Load(db gorp.SqlExecutor, key string, user *sdk.User, mods ...Mod) (*sdk.Project, error) {
	var c funcpar
	for _, f := range mods {
		f(&c)
	}

	var p *sdk.Project
	var err error

	if c.loadvariables && c.loadapps {
		p, err = loadprojectwithvariablesandapps(db, key, user)
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
func LoadProjectByPipelineID(db gorp.SqlExecutor, pipelineID int64) (*sdk.Project, error) {
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
	projectData.LastModified = lastModified
	return &projectData, nil
}

// InsertProject insert given project into given database
func InsertProject(db gorp.SqlExecutor, p *sdk.Project) error {
	if p.Name == "" {
		return sdk.ErrInvalidName
	}
	query := `INSERT INTO project (projectKey, name) VALUES($1,$2) RETURNING id`
	err := db.QueryRow(query, p.Key, p.Name).Scan(&p.ID)
	return err
}

// UpdateProjectDB set new project name in database
func UpdateProjectDB(db gorp.SqlExecutor, projectKey, projectName string) (time.Time, error) {
	var lastModified time.Time
	query := `UPDATE project SET name=$1, last_modified=current_timestamp WHERE projectKey=$2 RETURNING last_modified`
	err := db.QueryRow(query, projectName, projectKey).Scan(&lastModified)
	return lastModified, err
}

//LastUpdates returns projects and application last update
func LastUpdates(db gorp.SqlExecutor, user *sdk.User, since time.Time) ([]sdk.ProjectLastUpdates, error) {
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

// AddKeyPairToProject generate a ssh key pair and add them as project variables
func AddKeyPairToProject(db gorp.SqlExecutor, proj *sdk.Project, keyname string) error {

	pub, priv, errGenerate := keys.Generatekeypair(keyname)
	if errGenerate != nil {
		return errGenerate
	}

	v := sdk.Variable{
		Name:  keyname,
		Type:  sdk.KeyVariable,
		Value: priv,
	}

	if err := InsertVariableInProject(db, proj, v); err != nil {
		return err
	}

	p := sdk.Variable{
		Name:  keyname + ".pub",
		Type:  sdk.TextVariable,
		Value: pub,
	}

	return InsertVariableInProject(db, proj, p)
}
