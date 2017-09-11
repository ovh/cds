package api

import (
	"context"
	"database/sql"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getProjectsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		withApplication := FormBool(r, "application")

		var projects []sdk.Project
		var err error

		if withApplication {
			projects, err = project.LoadAll(api.mustDB(), api.Cache, getUser(ctx), project.LoadOptions.WithApplications)
		} else {
			projects, err = project.LoadAll(api.mustDB(), api.Cache, getUser(ctx))
		}
		if err != nil {
			return sdk.WrapError(err, "getProjectsHandler")
		}
		return WriteJSON(w, r, projects, http.StatusOK)
	}
}

func (api *API) updateProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		proj := &sdk.Project{}
		if err := UnmarshalBody(r, proj); err != nil {
			return sdk.WrapError(err, "updateProject> Unmarshall error")
		}

		if proj.Name == "" {
			return sdk.WrapError(sdk.ErrInvalidProjectName, "updateProject> Project name must no be empty")
		}

		// Check Request
		if key != proj.Key {
			return sdk.WrapError(sdk.ErrWrongRequest, "updateProject> bad Project key %s/%s ", key, proj.Key)
		}

		// Check is project exist
		p, errProj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errProj != nil {
			return sdk.WrapError(errProj, "updateProject> Cannot load project from db")
		}
		// Update in DB is made given the primary key
		proj.ID = p.ID
		if errUp := project.Update(api.mustDB(), api.Cache, proj, getUser(ctx)); errUp != nil {
			return sdk.WrapError(errUp, "updateProject> Cannot update project %s", key)
		}

		return WriteJSON(w, r, p, http.StatusOK)
	}
}

func (api *API) getProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		WithVariables := FormBool(r, "withVariables")
		WithApplications := FormBool(r, "withApplications")
		WithApplicationPipelines := FormBool(r, "withApplicationPipelines")
		WithPipelines := FormBool(r, "withPipelines")
		WithEnvironments := FormBool(r, "withEnvironments")
		WithGroups := FormBool(r, "withGroups")
		WithPermission := FormBool(r, "withPermission")
		WithRepositoriesManagers := FormBool(r, "withRepositoriesManagers")
		WithKeys := FormBool(r, "withKeys")

		opts := []project.LoadOptionFunc{}
		if WithVariables {
			opts = append(opts, project.LoadOptions.WithVariables)
		}
		if WithApplications {
			opts = append(opts, project.LoadOptions.WithApplications)
		}
		if WithApplicationPipelines {
			opts = append(opts, project.LoadOptions.WithApplicationPipelines)
		}
		if WithPipelines {
			opts = append(opts, project.LoadOptions.WithPipelines)
		}
		if WithEnvironments {
			opts = append(opts, project.LoadOptions.WithEnvironments)
		}
		if WithGroups {
			opts = append(opts, project.LoadOptions.WithGroups)
		}
		if WithPermission {
			opts = append(opts, project.LoadOptions.WithPermission)
		}
		if WithRepositoriesManagers {
			opts = append(opts, project.LoadOptions.WithRepositoriesManagers)
		}
		if WithKeys {
			opts = append(opts, project.LoadOptions.WithKeys)
		}

		p, errProj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), opts...)
		if errProj != nil {
			return sdk.WrapError(errProj, "getProjectHandler (%s)", key)
		}

		return WriteJSON(w, r, p, http.StatusOK)
	}
}

func (api *API) addProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Unmarshal data
		p := &sdk.Project{}
		if err := UnmarshalBody(r, p); err != nil {
			return sdk.WrapError(err, "AddProject> Unable to unmarshal body")
		}

		// check projectKey pattern
		if rgxp := regexp.MustCompile(sdk.ProjectKeyPattern); !rgxp.MatchString(p.Key) {
			return sdk.WrapError(sdk.ErrInvalidProjectKey, "AddProject> Project key %s do not respect pattern %s")
		}

		//check project Name
		if p.Name == "" {
			return sdk.WrapError(sdk.ErrInvalidProjectName, "AddProject> Project name must no be empty")
		}

		// Check that project does not already exists
		exist, errExist := project.Exist(api.mustDB(), p.Key)
		if errExist != nil {
			return sdk.WrapError(errExist, "AddProject>  Cannot check if project %s exist", p.Key)
		}

		if exist {
			return sdk.WrapError(sdk.ErrConflict, "AddProject> Project %s already exists", p.Key)
		}

		var groupAttached bool
		for i := range p.ProjectGroups {
			groupPermission := &p.ProjectGroups[i]
			if strings.TrimSpace(groupPermission.Group.Name) == "" {
				continue
			}
			groupAttached = true
		}
		if !groupAttached {
			// check if new auto group does not already exists
			if _, errl := group.LoadGroup(api.mustDB(), p.Name); errl != nil {
				if errl == sdk.ErrGroupNotFound {
					// group name does not exists, add it on project
					permG := sdk.GroupPermission{
						Group:      sdk.Group{Name: p.Name},
						Permission: permission.PermissionReadWriteExecute,
					}
					p.ProjectGroups = append(p.ProjectGroups, permG)
				} else {
					return sdk.WrapError(errl, "AddProject> Cannot check if group already exists")
				}
			} else {
				return sdk.WrapError(sdk.ErrGroupPresent, "AddProject> Group %s already exists", p.Name)
			}
		}

		//Create a project within a transaction
		tx, errBegin := api.mustDB().Begin()
		defer tx.Rollback()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "AddProject> Cannot start tx")
		}

		if err := project.Insert(tx, api.Cache, p, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "AddProject> Cannot insert project")
		}

		// Add group
		for i := range p.ProjectGroups {
			groupPermission := &p.ProjectGroups[i]

			// Insert group
			groupID, new, errGroup := group.AddGroup(tx, &groupPermission.Group)
			if groupID == 0 {
				return errGroup
			}
			groupPermission.Group.ID = groupID

			// Add group on project
			if err := group.InsertGroupInProject(tx, p.ID, groupPermission.Group.ID, groupPermission.Permission); err != nil {
				return sdk.WrapError(err, "addProject> Cannot add group %s in project %s", groupPermission.Group.Name, p.Name)
			}

			// Add user in group
			if new {
				if err := group.InsertUserInGroup(tx, groupPermission.Group.ID, getUser(ctx).ID, true); err != nil {
					return sdk.WrapError(err, "addProject> Cannot add user %s in group %s", getUser(ctx).Username, groupPermission.Group.Name)
				}
			}
		}

		for _, v := range p.Variable {
			var errVar error
			switch v.Type {
			case sdk.KeyVariable:
				errVar = project.AddKeyPair(tx, p, v.Name, getUser(ctx))
			default:
				errVar = project.InsertVariable(tx, p, &v, getUser(ctx))
			}
			if errVar != nil {
				return sdk.WrapError(errVar, "addProject> Cannot add variable %s in project %s", v.Name, p.Name)
			}
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p); err != nil {
			return sdk.WrapError(err, "addProject> Cannot update last modified")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addProject> Cannot commit transaction")
		}

		return WriteJSON(w, r, p, http.StatusCreated)
	}
}

func (api *API) deleteProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		p, errProj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithPipelines, project.LoadOptions.WithApplications)
		if errProj != nil {
			if errProj != sdk.ErrNoProject {
				return sdk.WrapError(errProj, "deleteProject> load project '%s' from db", key)
			}
			return sdk.WrapError(errProj, "deleteProject> cannot load project %s", key)
		}

		if len(p.Pipelines) > 0 {
			return sdk.WrapError(sdk.ErrProjectHasPipeline, "deleteProject> Project '%s' still used by %d pipelines", key, len(p.Pipelines))
		}

		if len(p.Applications) > 0 {
			return sdk.WrapError(sdk.ErrProjectHasApplication, "deleteProject> Project '%s' still used by %d applications", key, len(p.Applications))
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "deleteProject> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := project.Delete(tx, api.Cache, p.Key); err != nil {
			return sdk.WrapError(err, "deleteProject> cannot delete project %s", key)
		}
		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteProject> Cannot commit transaction")
		}
		log.Info("Project %s deleted.", p.Name)

		return nil
	}
}

func (api *API) getUserLastUpdatesHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		sinceHeader := r.Header.Get("If-Modified-Since")
		since := time.Unix(0, 0)
		if sinceHeader != "" {
			since, _ = time.Parse(time.RFC1123, sinceHeader)
		}

		lastUpdates, errUp := project.LastUpdates(api.mustDB(), api.Cache, getUser(ctx), since)
		if errUp != nil {
			if errUp == sql.ErrNoRows {
				w.WriteHeader(http.StatusNotModified)
				return nil
			}
			return errUp
		}
		if len(lastUpdates) == 0 {
			w.WriteHeader(http.StatusNotModified)
			return nil
		}

		return WriteJSON(w, r, lastUpdates, http.StatusOK)
	}
}
