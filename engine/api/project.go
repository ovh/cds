package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getProjectsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		withApplications := FormBool(r, "application")
		withWorkflows := FormBool(r, "workflow")
		filterByRepo := r.FormValue("repo")
		withPermissions := r.FormValue("permission")
		withIcon := FormBool(r, "withIcon")

		var u = deprecatedGetUser(ctx)
		requestedUserName := r.Header.Get("X-Cds-Username")

		//A provider can make a call for a specific user
		if getProvider(ctx) != nil && requestedUserName != "" {
			var err error
			//Load the specific user
			u, err = user.LoadUserWithoutAuth(api.mustDB(), requestedUserName)
			if err != nil {
				if sdk.Cause(err) == sql.ErrNoRows {
					return sdk.ErrUserNotFound
				}
				return sdk.WrapError(err, "unable to load user '%s'", requestedUserName)
			}
			if err := loadUserPermissions(api.mustDB(), api.Cache, u); err != nil {
				return sdk.WrapError(err, "unable to load user '%s' permissions", requestedUserName)
			}
		}

		opts := []project.LoadOptionFunc{
			project.LoadOptions.WithPermission,
		}

		if withIcon {
			opts = append(opts, project.LoadOptions.WithIcon)
		}
		if withApplications {
			opts = append(opts, project.LoadOptions.WithApplications)
		}
		if withWorkflows {
			opts = append(opts, project.LoadOptions.WithIntegrations, project.LoadOptions.WithWorkflows)
		}

		if filterByRepo == "" {
			projects, err := project.LoadAll(ctx, api.mustDB(), api.Cache, u, opts...)
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

			return service.WriteJSON(w, projects, http.StatusOK)
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

				//Checks the workflow use one of the applications
			wapps:
				for _, a := range w.Applications {
					for _, b := range apps {
						if a.Name == b.Name {
							ws = append(ws, p.Workflows[i])
							break wapps
						}
					}
				}
			}
			p.Workflows = ws
			return nil
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

		return service.WriteJSON(w, projects, http.StatusOK)
	}
}

func (api *API) updateProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		proj := &sdk.Project{}
		if err := service.UnmarshalBody(r, proj); err != nil {
			return sdk.WrapError(err, "Unmarshall error")
		}

		if proj.Name == "" {
			return sdk.WrapError(sdk.ErrInvalidProjectName, "updateProject> Project name must no be empty")
		}

		// Check Request
		if key != proj.Key {
			return sdk.WrapError(sdk.ErrWrongRequest, "updateProject> bad Project key %s/%s ", key, proj.Key)
		}

		// Check is project exist
		p, errProj := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.WithIcon)
		if errProj != nil {
			return sdk.WrapError(errProj, "updateProject> Cannot load project from db")
		}
		// Update in DB is made given the primary key
		proj.ID = p.ID
		proj.VCSServers = p.VCSServers
		if proj.Icon == "" {
			p.Icon = proj.Icon
		}
		if errUp := project.Update(api.mustDB(), api.Cache, proj, deprecatedGetUser(ctx)); errUp != nil {
			return sdk.WrapError(errUp, "updateProject> Cannot update project %s", key)
		}
		event.PublishUpdateProject(proj, p, deprecatedGetUser(ctx))
		return service.WriteJSON(w, proj, http.StatusOK)
	}
}

func (api *API) getProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		withVariables := FormBool(r, "withVariables")
		withApplications := FormBool(r, "withApplications")
		withApplicationNames := FormBool(r, "withApplicationNames")
		withPipelines := FormBool(r, "withPipelines")
		withPipelineNames := FormBool(r, "withPipelineNames")
		withEnvironments := FormBool(r, "withEnvironments")
		withEnvironmentNames := FormBool(r, "withEnvironmentNames")
		withGroups := FormBool(r, "withGroups")
		withPermission := FormBool(r, "withPermission")
		withKeys := FormBool(r, "withKeys")
		withWorkflows := FormBool(r, "withWorkflows")
		withWorkflowNames := FormBool(r, "withWorkflowNames")
		withIntegrations := FormBool(r, "withIntegrations")
		withFeatures := FormBool(r, "withFeatures")
		withIcon := FormBool(r, "withIcon")
		withLabels := FormBool(r, "withLabels")

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
		if withPipelines {
			opts = append(opts, project.LoadOptions.WithPipelines)
		}
		if withPipelineNames {
			opts = append(opts, project.LoadOptions.WithPipelineNames)
		}
		if withEnvironments {
			opts = append(opts, project.LoadOptions.WithEnvironments)
		}
		if withEnvironmentNames {
			opts = append(opts, project.LoadOptions.WithEnvironmentNames)
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
		if withIntegrations {
			opts = append(opts, project.LoadOptions.WithIntegrations)
		}
		if withFeatures {
			opts = append(opts, project.LoadOptions.WithFeatures)
		}
		if withIcon {
			opts = append(opts, project.LoadOptions.WithIcon)
		}
		if withLabels {
			opts = append(opts, project.LoadOptions.WithLabels)
		}

		p, errProj := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx), opts...)
		if errProj != nil {
			return sdk.WrapError(errProj, "getProjectHandler (%s)", key)
		}

		p.URLs.APIURL = api.Config.URL.API + api.Router.GetRoute("GET", api.getProjectHandler, map[string]string{"permProjectKey": key})
		p.URLs.UIURL = api.Config.URL.UI + "/project/" + key

		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) putProjectLabelsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		db := api.mustDB()

		var labels []sdk.Label
		if err := service.UnmarshalBody(r, &labels); err != nil {
			return sdk.WrapError(err, "Unmarshall error")
		}

		// Check is project exist
		proj, errProj := project.Load(db, api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.WithLabels)
		if errProj != nil {
			return sdk.WrapError(errProj, "putProjectLabelsHandler> Cannot load project from db")
		}

		var labelsToUpdate, labelsToAdd []sdk.Label
		for _, lblUpdated := range labels {
			var lblFound bool
			for _, lbl := range proj.Labels {
				if lbl.ID == lblUpdated.ID {
					lblFound = true
				}
			}
			lblUpdated.ProjectID = proj.ID
			if lblFound {
				labelsToUpdate = append(labelsToUpdate, lblUpdated)
			} else {
				labelsToAdd = append(labelsToAdd, lblUpdated)
			}
		}

		var labelsToDelete []sdk.Label
		for _, lbl := range proj.Labels {
			var lblFound bool
			for _, lblUpdated := range labels {
				if lbl.ID == lblUpdated.ID {
					lblFound = true
				}
			}
			if !lblFound {
				lbl.ProjectID = proj.ID
				labelsToDelete = append(labelsToDelete, lbl)
			}
		}

		tx, errTx := db.Begin()
		if errTx != nil {
			return sdk.WrapError(errTx, "putProjectLabelsHandler> Cannot create transaction")
		}
		defer tx.Rollback() //nolint

		for _, lblToDelete := range labelsToDelete {
			if err := project.DeleteLabel(tx, lblToDelete.ID); err != nil {
				return sdk.WrapError(err, "cannot delete label %s with id %d", lblToDelete.Name, lblToDelete.ID)
			}
		}
		for _, lblToUpdate := range labelsToUpdate {
			if err := project.UpdateLabel(tx, &lblToUpdate); err != nil {
				return sdk.WrapError(err, "cannot update label %s with id %d", lblToUpdate.Name, lblToUpdate.ID)
			}
		}
		for _, lblToAdd := range labelsToAdd {
			if err := project.InsertLabel(tx, &lblToAdd); err != nil {
				return sdk.WrapError(err, "cannot add label %s with id %d", lblToAdd.Name, lblToAdd.ID)
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		p, errP := project.Load(db, api.Cache, key, deprecatedGetUser(ctx),
			project.LoadOptions.WithLabels, project.LoadOptions.WithWorkflowNames, project.LoadOptions.WithVariables,
			project.LoadOptions.WithFavorites, project.LoadOptions.WithKeys, project.LoadOptions.WithPermission, project.LoadOptions.WithIntegrations)
		if errP != nil {
			return sdk.WrapError(errP, "putProjectLabelsHandler> Cannot load project updated from db")
		}
		event.PublishUpdateProject(p, proj, deprecatedGetUser(ctx))

		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) addProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Unmarshal data
		p := &sdk.Project{}
		if err := service.UnmarshalBody(r, p); err != nil {
			return sdk.WrapError(err, "Unable to unmarshal body")
		}

		// check projectKey pattern
		if rgxp := regexp.MustCompile(sdk.ProjectKeyPattern); !rgxp.MatchString(p.Key) {
			return sdk.WrapError(sdk.ErrInvalidProjectKey, "addProjectHandler> Project key %s do not respect pattern %s", p.Key, sdk.ProjectKeyPattern)
		}

		//check project Name
		if p.Name == "" {
			return sdk.WrapError(sdk.ErrInvalidProjectName, "addProjectHandler> Project name must no be empty")
		}

		// Check that project does not already exists
		exist, errExist := project.Exist(api.mustDB(), p.Key)
		if errExist != nil {
			return sdk.WrapError(errExist, "Cannot check if project %s exist", p.Key)
		}

		if exist {
			return sdk.WrapError(sdk.ErrConflict, "Project %s already exists", p.Key)
		}

		var groupAttached bool
		for i := range p.ProjectGroups {
			groupPermission := &p.ProjectGroups[i]
			if strings.TrimSpace(groupPermission.Group.Name) == "" {
				continue
			}
			// the default group could not be selected on ui 'Project Add'
			if groupPermission.Group.ID != 0 && !group.IsDefaultGroupID(groupPermission.Group.ID) {
				groupAttached = true
				continue
			}
			if groupPermission.Group.Name != "" && !group.IsDefaultGroupName(groupPermission.Group.Name) {
				groupAttached = true
			}
		}

		if !groupAttached {
			// check if new auto group does not already exists
			if _, errl := group.LoadGroup(api.mustDB(), p.Name); errl != nil {
				if sdk.ErrorIs(errl, sdk.ErrGroupNotFound) {
					// group name does not exists, add it on project
					permG := sdk.GroupPermission{
						Group:      sdk.Group{Name: strings.Replace(p.Name, " ", "", -1)},
						Permission: permission.PermissionReadWriteExecute,
					}
					p.ProjectGroups = append(p.ProjectGroups, permG)
				} else {
					return sdk.WrapError(errl, "Cannot check if group already exists")
				}
			} else {
				return sdk.WrapError(sdk.ErrGroupPresent, "Group %s already exists", p.Name)
			}
		}

		//Create a project within a transaction
		tx, errBegin := api.mustDB().Begin()
		defer tx.Rollback()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "Cannot start transaction")
		}

		if err := project.Insert(tx, api.Cache, p, deprecatedGetUser(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot insert project")
		}

		// Add group
		for i := range p.ProjectGroups {
			groupPermission := &p.ProjectGroups[i]

			// Insert group
			groupID, newGroup, errGroup := group.AddGroup(tx, &groupPermission.Group)
			if groupID == 0 {
				return errGroup
			}
			groupPermission.Group.ID = groupID

			if group.IsDefaultGroupID(groupID) {
				groupPermission.Permission = permission.PermissionRead
			}

			// Add group on project
			if err := group.InsertGroupInProject(tx, p.ID, groupPermission.Group.ID, groupPermission.Permission); err != nil {
				return sdk.WrapError(err, "Cannot add group %s in project %s", groupPermission.Group.Name, p.Name)
			}

			// Add user in group
			if newGroup {
				if err := group.InsertUserInGroup(tx, groupPermission.Group.ID, deprecatedGetUser(ctx).ID, true); err != nil {
					return sdk.WrapError(err, "Cannot add user %s in group %s", deprecatedGetUser(ctx).Username, groupPermission.Group.Name)
				}
			}
		}

		for _, v := range p.Variable {
			if errVar := project.InsertVariable(tx, p, &v, deprecatedGetUser(ctx)); errVar != nil {
				return sdk.WrapError(errVar, "addProjectHandler> Cannot add variable %s in project %s", v.Name, p.Name)
			}
		}

		var sshExists, gpgExists bool
		for _, k := range p.Keys {
			switch k.Type {
			case sdk.KeyTypeSSH:
				sshExists = true
			case sdk.KeyTypePGP:
				gpgExists = true
			}
		}

		if !sshExists {
			p.Keys = append(p.Keys, sdk.ProjectKey{Key: sdk.Key{
				Type: sdk.KeyTypeSSH,
				Name: fmt.Sprintf("proj-%s-%s", sdk.KeyTypeSSH, strings.ToLower(p.Key))},
			})
		}
		if !gpgExists {
			p.Keys = append(p.Keys, sdk.ProjectKey{Key: sdk.Key{
				Type: sdk.KeyTypePGP,
				Name: fmt.Sprintf("proj-%s-%s", sdk.KeyTypePGP, strings.ToLower(p.Key))},
			})
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
				return sdk.WrapError(errK, "addProjectHandler> Cannot add key %s in project %s", k.Name, p.Name)
			}
		}

		integrationModels, errP := integration.LoadModels(tx)
		if errP != nil {
			return sdk.WrapError(errP, "addProjectHandler> Cannot load integration models")
		}

		for i := range integrationModels {
			pf := &integrationModels[i]
			if err := propagatePublicIntegrationModelOnProject(tx, api.Cache, *pf, *p, deprecatedGetUser(ctx)); err != nil {
				return sdk.WrapError(err, "propagatePublicIntegrationModelOnProject error")
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}
		p.Permission = permission.PermissionReadWriteExecute

		event.PublishAddProject(p, deprecatedGetUser(ctx))

		proj, errL := project.Load(api.mustDB(), api.Cache, p.Key, deprecatedGetUser(ctx), project.LoadOptions.WithLabels, project.LoadOptions.WithWorkflowNames,
			project.LoadOptions.WithFavorites, project.LoadOptions.WithKeys, project.LoadOptions.WithPermission,
			project.LoadOptions.WithIntegrations, project.LoadOptions.WithVariables)
		if errL != nil {
			return sdk.WrapError(errL, "Cannot load project %s", p.Key)
		}

		return service.WriteJSON(w, *proj, http.StatusCreated)
	}
}

func (api *API) deleteProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		p, errProj := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.WithPipelines, project.LoadOptions.WithApplications)
		if errProj != nil {
			if !sdk.ErrorIs(errProj, sdk.ErrNoProject) {
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
			return sdk.WrapError(err, "cannot delete project %s", key)
		}
		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishDeleteProject(p, deprecatedGetUser(ctx))

		log.Info("Project %s deleted.", p.Name)

		return nil
	}
}
