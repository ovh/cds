package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/ovh/cds/sdk/slug"

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

func (api *API) getProjectsHandler_FilterByRepo(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	withPermissions := r.FormValue("permission")
	filterByRepo := r.FormValue("repo")

	var projects sdk.Projects
	var err error
	var filterByRepoFunc = func(db gorp.SqlExecutor, store cache.Store, p *sdk.Project) error {
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
			w, err := workflow.LoadByID(ctx, db, store, p, p.Workflows[i].ID, workflow.LoadOptions{})
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

	opts := []project.LoadOptionFunc{
		project.LoadOptions.WithPermission,
	}
	opts = append(opts, filterByRepoFunc)

	if isMaintainer(ctx) || isAdmin(ctx) {
		projects, err = project.LoadAllByRepo(ctx, api.mustDB(), api.Cache, filterByRepo, opts...)
		if err != nil {
			return err
		}
	} else {
		projects, err = project.LoadAllByRepoAndGroupIDs(ctx, api.mustDB(), api.Cache, getAPIConsumer(ctx).GetGroupIDs(), filterByRepo, opts...)
		if err != nil {
			return err
		}
	}

	pKeys := projects.Keys()
	perms, err := permission.LoadProjectMaxLevelPermission(ctx, api.mustDB(), pKeys, getAPIConsumer(ctx).GetGroupIDs())
	if err != nil {
		return err
	}
	for i := range projects {
		if isAdmin(ctx) {
			projects[i].Permissions = sdk.Permissions{Readable: true, Writable: true, Executable: true}
			continue
		}
		projects[i].Permissions = perms[projects[i].Key]
		if isMaintainer(ctx) {
			projects[i].Permissions.Readable = true
		}
	}

	if strings.ToUpper(withPermissions) == "W" {
		res := make([]sdk.Project, 0, len(projects))
		for _, p := range projects {
			if p.Permissions.Writable {
				res = append(res, p)
			}
		}
		projects = res
	}

	return service.WriteJSON(w, projects, http.StatusOK)
}

func (api *API) getProjectsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		withPermissions := r.FormValue("permission")
		filterByRepo := r.FormValue("repo")
		if filterByRepo != "" {
			return api.getProjectsHandler_FilterByRepo(ctx, w, r)
		}

		withApplications := FormBool(r, "application")
		withWorkflows := FormBool(r, "workflow")
		withIcon := FormBool(r, "withIcon")

		requestedUserName := r.Header.Get("X-Cds-Username")
		var requestedUser *sdk.AuthentifiedUser
		if requestedUserName != "" {
			var err error
			requestedUser, err = user.LoadByUsername(ctx, api.mustDB(), requestedUserName)
			if err != nil {
				if sdk.Cause(err) == sql.ErrNoRows {
					return sdk.ErrUserNotFound
				}
				return sdk.WrapError(err, "unable to load user '%s'", requestedUserName)
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

		var projects sdk.Projects
		var err error
		switch {
		case isMaintainer(ctx) && requestedUser == nil:
			projects, err = project.LoadAll(ctx, api.mustDB(), api.Cache, opts...)
		case isMaintainer(ctx) && requestedUser != nil:
			groups, errG := group.LoadAllByUserID(context.TODO(), api.mustDB(), requestedUser.ID)
			if errG != nil {
				return sdk.WrapError(errG, "unable to load user '%s' groups", requestedUserName)
			}
			requestedUser.Groups = groups
			log.Debug("load all projects for user %s", requestedUser.Fullname)
			projects, err = project.LoadAllByGroupIDs(ctx, api.mustDB(), api.Cache, requestedUser.GetGroupIDs(), opts...)
		default:
			projects, err = project.LoadAllByGroupIDs(ctx, api.mustDB(), api.Cache, getAPIConsumer(ctx).GetGroupIDs(), opts...)
		}
		if err != nil {
			return err
		}

		var groupIDs []int64
		var admin bool
		var maintainer bool
		if requestedUser == nil {
			groupIDs = getAPIConsumer(ctx).GetGroupIDs()
			admin = isAdmin(ctx)
			maintainer = isMaintainer(ctx)
		} else {
			groupIDs = requestedUser.GetGroupIDs()
			admin = requestedUser.Ring == sdk.UserRingAdmin
			maintainer = requestedUser.Ring == sdk.UserRingMaintainer
		}

		pKeys := projects.Keys()
		perms, err := permission.LoadProjectMaxLevelPermission(ctx, api.mustDB(), pKeys, groupIDs)
		if err != nil {
			return err
		}
		for i := range projects {
			if admin {
				projects[i].Permissions = sdk.Permissions{Readable: true, Writable: true, Executable: true}
				continue
			}
			projects[i].Permissions = perms[projects[i].Key]
			if maintainer {
				projects[i].Permissions.Readable = true
			}
		}

		if strings.ToUpper(withPermissions) == "W" {
			res := make([]sdk.Project, 0, len(projects))
			for _, p := range projects {
				if p.Permissions.Writable {
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
		p, errProj := project.Load(api.mustDB(), api.Cache, key, project.LoadOptions.WithIcon)
		if errProj != nil {
			return sdk.WrapError(errProj, "updateProject> Cannot load project from db")
		}
		// Update in DB is made given the primary key
		proj.ID = p.ID
		proj.VCSServers = p.VCSServers
		if proj.Icon == "" {
			p.Icon = proj.Icon
		}
		if errUp := project.Update(api.mustDB(), api.Cache, proj); errUp != nil {
			return sdk.WrapError(errUp, "updateProject> Cannot update project %s", key)
		}
		event.PublishUpdateProject(ctx, proj, p, getAPIConsumer(ctx))

		proj.Permissions.Readable = true
		proj.Permissions.Writable = true

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
			project.LoadOptions.WithFavorites(getAPIConsumer(ctx).AuthentifiedUser.ID),
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

		p, errProj := project.Load(api.mustDB(), api.Cache, key, opts...)
		if errProj != nil {
			return sdk.WrapError(errProj, "getProjectHandler (%s)", key)
		}

		p.URLs.APIURL = api.Config.URL.API + api.Router.GetRoute("GET", api.getProjectHandler, map[string]string{"permProjectKey": key})
		p.URLs.UIURL = api.Config.URL.UI + "/project/" + key

		if isAdmin(ctx) {
			p.Permissions = sdk.Permissions{Readable: true, Writable: true, Executable: true}
		} else {
			permissions, err := permission.LoadProjectMaxLevelPermission(ctx, api.mustDB(), []string{p.Key}, getAPIConsumer(ctx).GetGroupIDs())
			if err != nil {
				return err
			}
			p.Permissions = permissions.Permissions(p.Key)
			if isMaintainer(ctx) {
				p.Permissions.Readable = true
			}
		}

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
		proj, err := project.Load(db, api.Cache, key, project.LoadOptions.WithLabels)
		if err != nil {
			return err
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

		p, errP := project.Load(db, api.Cache, key,
			project.LoadOptions.WithLabels, project.LoadOptions.WithWorkflowNames, project.LoadOptions.WithVariables,
			project.LoadOptions.WithFavorites(getAPIConsumer(ctx).AuthentifiedUser.ID),
			project.LoadOptions.WithKeys, project.LoadOptions.WithPermission, project.LoadOptions.WithIntegrations)
		if errP != nil {
			return sdk.WrapError(errP, "putProjectLabelsHandler> Cannot load project updated from db")
		}

		p.Permissions.Readable = true
		p.Permissions.Writable = true

		event.PublishUpdateProject(ctx, p, proj, getAPIConsumer(ctx))

		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) postProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		consumer := getAPIConsumer(ctx)

		var p sdk.Project
		if err := service.UnmarshalBody(r, &p); err != nil {
			return sdk.WrapError(err, "unable to unmarshal body")
		}

		// Check key pattern
		if rgxp := regexp.MustCompile(sdk.ProjectKeyPattern); !rgxp.MatchString(p.Key) {
			return sdk.WrapError(sdk.ErrInvalidProjectKey, "project key %s do not respect pattern %s", p.Key, sdk.ProjectKeyPattern)
		}

		// Check project name
		if p.Name == "" {
			return sdk.WrapError(sdk.ErrInvalidProjectName, "project name must no be empty")
		}

		//Create a project within a transaction
		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		// Check that project does not already exists
		exist, errExist := project.Exist(tx, p.Key)
		if errExist != nil {
			return sdk.WrapError(errExist, "cannot check if project %s exist", p.Key)
		}
		if exist {
			return sdk.WrapError(sdk.ErrConflict, "project %s already exists", p.Key)
		}

		if err := project.Insert(tx, api.Cache, &p); err != nil {
			return sdk.WrapError(err, "cannot insert project")
		}

		// Check that given project groups are valid
		var groupIDs []int64
		for _, gp := range p.ProjectGroups {
			var grp *sdk.Group
			var err error
			if gp.Group.ID != 0 {
				grp, err = group.LoadByID(ctx, tx, gp.Group.ID, group.LoadOptions.WithMembers)
			} else {
				grp, err = group.LoadByName(ctx, tx, gp.Group.Name, group.LoadOptions.WithMembers)
			}
			if err != nil {
				return err
			}

			// the default group could not be selected
			if group.IsDefaultGroupID(grp.ID) {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot use default group to create project")
			}

			// consumer should be group member to add it on a project
			if !isGroupMember(ctx, grp) && !isAdmin(ctx) {
				return sdk.WithStack(sdk.ErrInvalidGroupMember)
			}

			groupIDs = append(groupIDs, grp.ID)
		}

		// If no groups were given, try to create a new one with project name
		if len(groupIDs) == 0 {
			groupSlug := slug.Convert(p.Name)
			existingGroop, err := group.LoadByName(ctx, tx, groupSlug)
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return err
			}
			if existingGroop != nil {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot create a new group %s for given project name", groupSlug)
			}

			newGroup := sdk.Group{Name: groupSlug}
			if err := group.Create(ctx, tx, &newGroup, consumer.AuthentifiedUser.ID); err != nil {
				return err
			}

			groupIDs = []int64{newGroup.ID}
		}

		// Insert all links between project and group
		for _, groupID := range groupIDs {
			if err := group.InsertLinkGroupProject(ctx, tx, &group.LinkGroupProject{
				GroupID:   groupID,
				ProjectID: p.ID,
				Role:      sdk.PermissionReadWriteExecute,
			}); err != nil {
				return sdk.WrapError(err, "cannot add group %d in project %s", groupID, p.Name)
			}
		}

		for _, v := range p.Variable {
			if errVar := project.InsertVariable(tx, &p, &v, consumer); errVar != nil {
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

		integrationModels, err := integration.LoadModels(tx)
		if err != nil {
			return sdk.WrapError(err, "cannot load integration models")
		}

		for i := range integrationModels {
			pf := &integrationModels[i]
			if err := propagatePublicIntegrationModelOnProject(ctx, tx, api.Cache, *pf, p, consumer); err != nil {
				return sdk.WithStack(err)
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		event.PublishAddProject(ctx, &p, consumer)

		proj, err := project.Load(api.mustDB(), api.Cache, p.Key,
			project.LoadOptions.WithLabels,
			project.LoadOptions.WithWorkflowNames,
			project.LoadOptions.WithFavorites(consumer.AuthentifiedUser.ID),
			project.LoadOptions.WithKeys,
			project.LoadOptions.WithPermission,
			project.LoadOptions.WithIntegrations,
			project.LoadOptions.WithVariables,
		)
		if err != nil {
			return sdk.WrapError(err, "cannot load project %s", p.Key)
		}
		proj.Permissions.Readable = true
		proj.Permissions.Writable = true

		return service.WriteJSON(w, *proj, http.StatusCreated)
	}
}

func (api *API) deleteProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		p, errProj := project.Load(api.mustDB(), api.Cache, key, project.LoadOptions.WithPipelines, project.LoadOptions.WithApplications)
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
		defer tx.Rollback() // nolint

		if err := project.Delete(tx, api.Cache, p.Key); err != nil {
			return sdk.WrapError(err, "cannot delete project %s", key)
		}
		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishDeleteProject(ctx, p, getAPIConsumer(ctx))

		log.Info(ctx, "Project %s deleted.", p.Name)

		return nil
	}
}
