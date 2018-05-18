package api

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getProjectsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		withApplications := FormBool(r, "application")
		withWorkflows := FormBool(r, "workflow")
		filterByRepo := r.FormValue("repo")
		withPermissions := r.FormValue("permission")

		var u = getUser(ctx)

		//A provider can make a call for a specific user
		if getProvider(ctx) != nil {
			requestedUserName := r.Header.Get("X-Cds-Username")
			var err error
			//Load the specific user
			u, err = user.LoadUserWithoutAuth(api.mustDB(), requestedUserName)
			if err != nil {
				return sdk.WrapError(err, "getProjectsHandler> unable to load user '%s'", requestedUserName)
			}
		}

		opts := []project.LoadOptionFunc{
			project.LoadOptions.WithPermission,
		}
		if withApplications {
			opts = append(opts, project.LoadOptions.WithApplications)
		}

		if withWorkflows {
			opts = append(opts, project.LoadOptions.WithPlatforms, project.LoadOptions.WithWorkflows)
		}

		if filterByRepo == "" {
			projects, err := project.LoadAll(api.mustDB(), api.Cache, u, opts...)
			if err != nil {
				return sdk.WrapError(err, "getProjectsHandler")
			}

			if strings.ToUpper(withPermissions) == "W" {
				res := make([]sdk.Project, 0, len(projects))
				for _, p := range projects {
					if p.Permission >= permission.PermissionReadWriteExecute {
						res = append(res, p)
					}
				}
				projects = res
			}

			return WriteJSON(w, projects, http.StatusOK)
		}

		var filterByRepoFunc = func(db gorp.SqlExecutor, store cache.Store, p *sdk.Project, u *sdk.User) error {
			//Filter the applications by repo
			apps := []sdk.Application{}
			for i := range p.Applications {
				if p.Applications[i].RepositoryFullname == filterByRepo {
					apps = append(apps, p.Applications[i])
				}
			}
			p.Applications = apps
			ws := []sdk.Workflow{}
			//Filter the workflow by applications
			for i := range p.Workflows {
				w, err := workflow.LoadByID(db, store, p, p.Workflows[i].ID, u, workflow.LoadOptions{})
				if err != nil {
					return err
				}

				wapps := w.GetApplications()
				//Checks the workflow use one of the applications
			wapps:
				for _, a := range wapps {
					for _, b := range apps {
						if a.Name == b.Name {
							ws = append(ws, p.Workflows[i])
							break wapps
						}
					}
				}
			}
			p.Workflows = ws
			return WriteJSON(w, nil, http.StatusOK)
		}
		opts = append(opts, &filterByRepoFunc)

		projects, err := project.LoadAllByRepo(api.mustDB(), api.Cache, u, filterByRepo, opts...)
		if err != nil {
			return sdk.WrapError(err, "getProjectsHandler")
		}

		if strings.ToUpper(withPermissions) == "W" {
			res := make([]sdk.Project, 0, len(projects))
			for _, p := range projects {
				if p.Permission >= permission.PermissionReadWriteExecute {
					res = append(res, p)
				}
			}
			projects = res
		}

		return WriteJSON(w, projects, http.StatusOK)
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
		proj.VCSServers = p.VCSServers
		if errUp := project.Update(api.mustDB(), api.Cache, proj, getUser(ctx)); errUp != nil {
			return sdk.WrapError(errUp, "updateProject> Cannot update project %s", key)
		}
		event.PublishUpdateProject(proj, p, getUser(ctx))
		return WriteJSON(w, proj, http.StatusOK)
	}
}

func (api *API) getProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		withVariables := FormBool(r, "withVariables")
		withApplications := FormBool(r, "withApplications")
		withApplicationNames := FormBool(r, "withApplicationNames")
		withApplicationPipelines := FormBool(r, "withApplicationPipelines")
		withPipelines := FormBool(r, "withPipelines")
		withPipelineNames := FormBool(r, "withPipelineNames")
		withEnvironments := FormBool(r, "withEnvironments")
		withGroups := FormBool(r, "withGroups")
		withPermission := FormBool(r, "withPermission")
		withKeys := FormBool(r, "withKeys")
		withWorkflows := FormBool(r, "withWorkflows")
		withWorkflowNames := FormBool(r, "withWorkflowNames")
		withPlatforms := FormBool(r, "withPlatforms")
		withFeatures := FormBool(r, "withFeatures")

		opts := []project.LoadOptionFunc{
			project.LoadOptions.WithFavorites,
		}
		if withVariables {
			opts = append(opts, project.LoadOptions.WithVariables)
		}
		if withApplications {
			opts = append(opts, project.LoadOptions.WithApplications)
		}
		if withApplicationNames {
			opts = append(opts, project.LoadOptions.WithApplicationNames)
		}
		if withApplicationPipelines {
			opts = append(opts, project.LoadOptions.WithApplicationPipelines)
		}
		if withPipelines {
			opts = append(opts, project.LoadOptions.WithPipelines)
		}
		if withPipelineNames {
			opts = append(opts, project.LoadOptions.WithPipelineNames)
		}
		if withEnvironments {
			opts = append(opts, project.LoadOptions.WithEnvironments)
		}
		if withGroups {
			opts = append(opts, project.LoadOptions.WithGroups)
		}
		if withPermission {
			opts = append(opts, project.LoadOptions.WithPermission)
		}
		if withKeys {
			opts = append(opts, project.LoadOptions.WithKeys)
		}
		if withWorkflows {
			opts = append(opts, project.LoadOptions.WithWorkflows)
		}
		if withWorkflowNames {
			opts = append(opts, project.LoadOptions.WithWorkflowNames)
		}
		if withPlatforms {
			opts = append(opts, project.LoadOptions.WithPlatforms)
		}
		if withFeatures {
			opts = append(opts, project.LoadOptions.WithFeatures)
		}

		p, errProj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), opts...)
		if errProj != nil {
			return sdk.WrapError(errProj, "getProjectHandler (%s)", key)
		}

		return WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) addProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Unmarshal data
		p := &sdk.Project{}
		if err := UnmarshalBody(r, p); err != nil {
			return sdk.WrapError(err, "addProjectHandler> Unable to unmarshal body")
		}

		// check projectKey pattern
		if rgxp := regexp.MustCompile(sdk.ProjectKeyPattern); !rgxp.MatchString(p.Key) {
			return sdk.WrapError(sdk.ErrInvalidProjectKey, "addProjectHandler> Project key %s do not respect pattern %s")
		}

		//check project Name
		if p.Name == "" {
			return sdk.WrapError(sdk.ErrInvalidProjectName, "addProjectHandler> Project name must no be empty")
		}

		// Check that project does not already exists
		exist, errExist := project.Exist(api.mustDB(), p.Key)
		if errExist != nil {
			return sdk.WrapError(errExist, "addProjectHandler>  Cannot check if project %s exist", p.Key)
		}

		if exist {
			return sdk.WrapError(sdk.ErrConflict, "addProjectHandler> Project %s already exists", p.Key)
		}

		var groupAttached bool
		for i := range p.ProjectGroups {
			groupPermission := &p.ProjectGroups[i]
			if strings.TrimSpace(groupPermission.Group.Name) == "" {
				continue
			}
			// the default group could not be selected on ui 'Project Add'
			if !group.IsDefaultGroupID(groupPermission.Group.ID) {
				groupAttached = true
			}
		}
		if !groupAttached {
			// check if new auto group does not already exists
			if _, errl := group.LoadGroup(api.mustDB(), p.Name); errl != nil {
				if errl == sdk.ErrGroupNotFound {
					// group name does not exists, add it on project
					permG := sdk.GroupPermission{
						Group:      sdk.Group{Name: strings.Replace(p.Name, " ", "", -1)},
						Permission: permission.PermissionReadWriteExecute,
					}
					p.ProjectGroups = append(p.ProjectGroups, permG)
				} else {
					return sdk.WrapError(errl, "addProjectHandler> Cannot check if group already exists")
				}
			} else {
				return sdk.WrapError(sdk.ErrGroupPresent, "addProjectHandler> Group %s already exists", p.Name)
			}
		}

		//Create a project within a transaction
		tx, errBegin := api.mustDB().Begin()
		defer tx.Rollback()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "addProjectHandler> Cannot start tx")
		}

		if err := project.Insert(tx, api.Cache, p, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "addProjectHandler> Cannot insert project")
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

			if group.IsDefaultGroupID(groupID) {
				groupPermission.Permission = permission.PermissionRead
			}

			// Add group on project
			if err := group.InsertGroupInProject(tx, p.ID, groupPermission.Group.ID, groupPermission.Permission); err != nil {
				return sdk.WrapError(err, "addProjectHandler> Cannot add group %s in project %s", groupPermission.Group.Name, p.Name)
			}

			// Add user in group
			if new {
				if err := group.InsertUserInGroup(tx, groupPermission.Group.ID, getUser(ctx).ID, true); err != nil {
					return sdk.WrapError(err, "addProjectHandler> Cannot add user %s in group %s", getUser(ctx).Username, groupPermission.Group.Name)
				}
			}
		}

		for _, v := range p.Variable {
			if errVar := project.InsertVariable(tx, p, &v, getUser(ctx)); errVar != nil {
				return sdk.WrapError(errVar, "addProjectHandler> Cannot add variable %s in project %s", v.Name, p.Name)
			}
		}

		for _, k := range p.Keys {
			k.ProjectID = p.ID
			switch k.Type {
			case sdk.KeyTypeSSH:
				keyTemp, errK := keys.GenerateSSHKey(k.Name)
				if errK != nil {
					return sdk.WrapError(errK, "addProjectHandler> Cannot generate ssh key for project %s", p.Name)
				}
				k.Key = keyTemp
			case sdk.KeyTypePGP:
				keyTemp, errK := keys.GeneratePGPKeyPair(k.Name)
				if errK != nil {
					return sdk.WrapError(errK, "addProjectHandler> Cannot generate pgp key for project %s", p.Name)
				}
				k.Key = keyTemp
			}
			if errK := project.InsertKey(tx, &k); errK != nil {
				return sdk.WrapError(errK, "addProjectHandler> Cannot add key %s in project %s", k.Name)
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addProjectHandler> Cannot commit transaction")
		}

		event.PublishAddProject(p, getUser(ctx))

		return WriteJSON(w, p, http.StatusCreated)
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

		event.PublishDeleteProject(p, getUser(ctx))

		log.Info("Project %s deleted.", p.Name)

		return WriteJSON(w, nil, http.StatusOK)
	}
}
