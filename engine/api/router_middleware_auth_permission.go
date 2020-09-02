package api

import (
	"context"
	"strconv"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

// PermCheckFunc defines func call to check permission
type PermCheckFunc func(ctx context.Context, key string, perm int, routeVars map[string]string) error

func permissionFunc(api *API) map[string]PermCheckFunc {
	return map[string]PermCheckFunc{
		"permProjectKey":        api.checkProjectPermissions,
		"permWorkflowName":      api.checkWorkflowPermissions,
		"permGroupName":         api.checkGroupPermissions,
		"permModelName":         api.checkWorkerModelPermissions,
		"permActionName":        api.checkActionPermissions,
		"permActionBuiltinName": api.checkActionBuiltinPermissions,
		"permTemplateSlug":      api.checkTemplateSlugPermissions,
		"permUsernamePublic":    api.checkUserPublicPermissions,
		"permUsername":          api.checkUserPermissions,
		"permConsumerID":        api.checkConsumerPermissions,
		"permSessionID":         api.checkSessionPermissions,
		"permJobID":             api.checkJobIDPermissions,
	}
}

func (api *API) checkPermission(ctx context.Context, routeVar map[string]string, permission int) error {
	for key, value := range routeVar {
		if permFunc, ok := permissionFunc(api)[key]; ok {
			if err := permFunc(ctx, value, permission, routeVar); err != nil {
				return err
			}
		}
	}
	return nil
}

func (api *API) checkJobIDPermissions(ctx context.Context, jobID string, perm int, routeVars map[string]string) error {
	ctx, end := telemetry.Span(ctx, "api.checkJobIDPermissions")
	defer end()

	id, err := strconv.ParseInt(jobID, 10, 64)
	if err != nil {
		log.Error(ctx, "checkJobIDPermissions> Unable to parse job id:%s err:%v", jobID, err)
		return sdk.WrapError(sdk.ErrForbidden, "not authorized for job %s", jobID)
	}

	runNodeJob, err := workflow.LoadNodeJobRun(ctx, api.mustDB(), api.Cache, id)
	if err != nil {
		log.Error(ctx, "checkWorkerPermission> Unable to load job %d err:%v", id, err)
		return sdk.WrapError(sdk.ErrForbidden, "not authorized for job %s", jobID)
	}

	// If the expected permission if >= RX and the consumer is a worker
	// We check that the worker has took this job
	if isWorker := isWorker(ctx); isWorker && perm >= sdk.PermissionReadExecute {
		wk, err := worker.LoadByID(ctx, api.mustDB(), getAPIConsumer(ctx).Worker.ID)
		if err != nil {
			return err
		}

		var ok bool
		k := cache.Key("api:workers", getAPIConsumer(ctx).ID, "perm", jobID)
		if has, _ := api.Cache.Get(k, &ok); ok && has {
			return nil
		}

		if wk.JobRunID != nil && runNodeJob.ID == *wk.JobRunID {
			ok = true
		}
		_ = api.Cache.SetWithTTL(k, ok, 60*60)
		if !ok {
			return sdk.WrapError(sdk.ErrForbidden, "not authorized for job %s", jobID)
		}
		return nil
	}

	// Else we check the exec groups
	if !runNodeJob.ExecGroups.HasOneOf(getAPIConsumer(ctx).GetGroupIDs()...) && !isAdmin(ctx) {
		return sdk.WrapError(sdk.ErrForbidden, "not authorized for job %s", jobID)
	}

	return nil
}

func (api *API) checkProjectPermissions(ctx context.Context, projectKey string, requiredPerm int, routeVars map[string]string) error {
	ctx, end := telemetry.Span(ctx, "api.checkProjectPermissions")
	defer end()

	if _, err := project.Load(ctx, api.mustDB(), projectKey); err != nil {
		return err
	}

	perms, err := permission.LoadProjectMaxLevelPermission(ctx, api.mustDB(), []string{projectKey}, getAPIConsumer(ctx).GetGroupIDs())
	if err != nil {
		return sdk.WrapError(err, "cannot get max project permissions for %s", projectKey)
	}

	callerPermission := perms.Level(projectKey)
	// If the caller based on its group doesn't have enough permission level
	if callerPermission < requiredPerm {
		log.Debug("checkProjectPermissions> callerPermission=%d ", callerPermission)
		// If it's about READ: we have to check if the user is a maintainer or an admin
		if requiredPerm == sdk.PermissionRead {
			if !isMaintainer(ctx) {
				// The caller doesn't enough permission level from its groups and is neither a maintainer nor an admin
				log.Debug("checkProjectPermissions> %s(%s) is not authorized to %s", getAPIConsumer(ctx).Name, getAPIConsumer(ctx).ID, projectKey)
				return sdk.WrapError(sdk.ErrNoProject, "not authorized for project %s", projectKey)
			}
			log.Debug("checkProjectPermissions> %s(%s) access granted to %s because is maintainer", getAPIConsumer(ctx).Name, getAPIConsumer(ctx).ID, projectKey)
			telemetry.Current(ctx, telemetry.Tag(telemetry.TagPermission, "is_maintainer"))
			return nil
		}

		// If it's about Execute of Write: we have to check if the user is an admin
		if !isAdmin(ctx) {
			// The caller doesn't enough permission level from its groups and is not an admin
			log.Debug("checkProjectPermissions> %s(%s) is not authorized to %s", getAPIConsumer(ctx).Name, getAPIConsumer(ctx).ID, projectKey)
			return sdk.WrapError(sdk.ErrForbidden, "not authorized for project %s", projectKey)
		}
		log.Debug("checkProjectPermissions> %s(%s) access granted to %s because is admin", getAPIConsumer(ctx).Name, getAPIConsumer(ctx).ID, projectKey)
		telemetry.Current(ctx, telemetry.Tag(telemetry.TagPermission, "is_admin"))
		return nil
	}
	log.Debug("checkWorkflowPermissions> %s(%s) access granted to %s because has permission (max permission = %d)", getAPIConsumer(ctx).Name, getAPIConsumer(ctx).ID, projectKey, callerPermission)
	telemetry.Current(ctx, telemetry.Tag(telemetry.TagPermission, "is_granted"))
	return nil
}

func (api *API) checkWorkflowPermissions(ctx context.Context, workflowName string, perm int, routeVars map[string]string) error {
	ctx, end := telemetry.Span(ctx, "api.checkWorkflowPermissions")
	defer end()

	projectKey, has := routeVars["permProjectKey"]
	if projectKey == "" {
		projectKey, has = routeVars["key"]
	}
	if !has {
		return sdk.WithStack(sdk.ErrNotFound)
	}

	if workflowName == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid given workflow name")
	}

	exists, err := workflow.Exists(api.mustDB(), projectKey, workflowName)
	if err != nil {
		return err
	}
	if !exists {
		return sdk.WithStack(sdk.ErrNotFound)
	}

	perms, err := permission.LoadWorkflowMaxLevelPermission(ctx, api.mustDB(), projectKey, []string{workflowName}, getAPIConsumer(ctx).GetGroupIDs())
	if err != nil {
		return sdk.NewError(sdk.ErrForbidden, err)
	}

	maxLevelPermission := perms.Level(workflowName)

	if maxLevelPermission < perm { // If the caller based on its group doesn have enough permission level
		// If it's about READ: we have to check if the user is a maintainer or an admin
		if perm < sdk.PermissionReadExecute {
			if !isMaintainer(ctx) {
				// The caller doesn't enough permission level from its groups and is neither a maintainer nor an admin
				log.Debug("checkWorkflowPermissions> %s is not authorized to %s/%s", getAPIConsumer(ctx).ID, projectKey, workflowName)
				return sdk.WrapError(sdk.ErrForbidden, "not authorized for workflow %s/%s", projectKey, workflowName)
			}
			log.Debug("checkWorkflowPermissions> %s access granted to %s/%s because is maintainer", getAPIConsumer(ctx).ID, projectKey, workflowName)
			telemetry.Current(ctx, telemetry.Tag(telemetry.TagPermission, "is_maintainer"))
			return nil
		}

		// If it's about Execute of Write: we have to check if the user is an admin
		if !isAdmin(ctx) {
			// The caller doesn't enough permission level from its groups and is not an admin
			log.Debug("checkWorkflowPermissions> %s is not authorized to %s/%s", getAPIConsumer(ctx).ID, projectKey, workflowName)
			return sdk.WrapError(sdk.ErrForbidden, "not authorized for workflow %s/%s", projectKey, workflowName)
		}
		log.Debug("checkWorkflowPermissions> %s access granted to %s/%s because is admin", getAPIConsumer(ctx).ID, projectKey, workflowName)
		telemetry.Current(ctx, telemetry.Tag(telemetry.TagPermission, "is_admin"))
		return nil

	}
	log.Debug("checkWorkflowPermissions> %s access granted to %s/%s because has permission (max permission = %d)", getAPIConsumer(ctx).ID, projectKey, workflowName, maxLevelPermission)
	telemetry.Current(ctx, telemetry.Tag(telemetry.TagPermission, "is_granted"))
	return nil
}

func (api *API) checkGroupPermissions(ctx context.Context, groupName string, permissionValue int, routeVars map[string]string) error {
	if groupName == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid given group name")
	}

	// check that group exists
	g, err := group.LoadByName(ctx, api.mustDB(), groupName, group.LoadOptions.WithMembers)
	if err != nil {
		return sdk.WrapError(err, "cannot get group for name %s", groupName)
	}

	log.Debug("api.checkGroupPermissions> group %d has members %v", g.ID, g.Members)

	if permissionValue > sdk.PermissionRead { // Only group administror or CDS administrator can update a group or its dependencies
		if !isGroupAdmin(ctx, g) && !isAdmin(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}
	} else {
		if !isGroupMember(ctx, g) && !isMaintainer(ctx) { // Only group member or CDS maintainer can get a group or its dependencies
			return sdk.WithStack(sdk.ErrForbidden)
		}
	}

	return nil
}

func (api *API) checkWorkerModelPermissions(ctx context.Context, modelName string, perm int, routeVars map[string]string) error {
	if modelName == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid worker model name")
	}

	g, err := group.LoadByName(ctx, api.mustDB(), routeVars["permGroupName"])
	if err != nil {
		return err
	}

	if _, err := workermodel.LoadByNameAndGroupID(ctx, api.mustDB(), modelName, g.ID, workermodel.LoadOptions.Default); err != nil {
		return err
	}

	return nil
}

func (api *API) checkActionPermissions(ctx context.Context, actionName string, perm int, routeVars map[string]string) error {
	if actionName == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid action name")
	}

	g, err := group.LoadByName(ctx, api.mustDB(), routeVars["permGroupName"])
	if err != nil {
		return err
	}

	a, err := action.LoadTypeDefaultByNameAndGroupID(ctx, api.mustDB(), actionName, g.ID)
	if err != nil {
		return err
	}
	if a == nil {
		return sdk.WithStack(sdk.ErrNotFound)
	}

	return nil
}

func (api *API) checkActionBuiltinPermissions(ctx context.Context, actionName string, perm int, routeVars map[string]string) error {
	if actionName == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid given action name")
	}

	a, err := action.LoadByTypesAndName(ctx, api.mustDB(), []string{sdk.BuiltinAction, sdk.PluginAction}, actionName)
	if err != nil {
		return err
	}
	if a == nil {
		return sdk.WithStack(sdk.ErrNoAction)
	}

	return nil
}

func (api *API) checkTemplateSlugPermissions(ctx context.Context, templateSlug string, permissionValue int, routeVars map[string]string) error {
	if templateSlug == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid workflow template slug")
	}

	g, err := group.LoadByName(ctx, api.mustDB(), routeVars["permGroupName"])
	if err != nil {
		return err
	}

	wt, err := workflowtemplate.LoadBySlugAndGroupID(ctx, api.mustDB(), templateSlug, g.ID)
	if err != nil {
		return err
	}
	if wt == nil {
		return sdk.WithStack(sdk.ErrNotFound)
	}

	return nil
}

// checkUserPublicPermissions give user R to everyone, RW to itself and RW to admin.
func (api *API) checkUserPublicPermissions(ctx context.Context, username string, permissionValue int, routeVars map[string]string) error {
	if username == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid username")
	}

	consumer := getAPIConsumer(ctx)

	var u *sdk.AuthentifiedUser
	var err error

	// Load user from database, returns an error if not exists
	if username == "me" {
		u, err = user.LoadByID(ctx, api.mustDB(), consumer.AuthentifiedUserID)
	} else {
		u, err = user.LoadByUsername(ctx, api.mustDB(), username)
	}
	if err != nil {
		return sdk.NewErrorWithStack(err, sdk.WrapError(sdk.ErrForbidden, "not authorized for user %s", username))
	}

	// Valid if the current consumer match given username
	if consumer.AuthentifiedUserID == u.ID {
		log.Debug("checkUserPermissions> %s read/write access granted to %s because itself", getAPIConsumer(ctx).ID, u.ID)
		return nil
	}

	// Everyone can read public user data
	if permissionValue == sdk.PermissionRead {
		log.Debug("checkUserPermissions> %s read access granted to %s on public user data", getAPIConsumer(ctx).ID, u.ID)
		return nil
	}

	// If the current user is an admin
	if isAdmin(ctx) {
		log.Debug("checkUserPermissions> %s read/write access granted to %s because is admin", getAPIConsumer(ctx).ID, u.ID)
		return nil
	}

	log.Debug("checkUserPermissions> %s is not authorized to %s", getAPIConsumer(ctx).ID, u.ID)
	return sdk.WrapError(sdk.ErrForbidden, "not authorized for user %s", username)
}

// checkUserPermissions give user RW to itself, R to maintainer and RW to admin.
func (api *API) checkUserPermissions(ctx context.Context, username string, permissionValue int, routeVars map[string]string) error {
	if username == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid username")
	}

	consumer := getAPIConsumer(ctx)

	var u *sdk.AuthentifiedUser
	var err error

	// Load user from database, returns an error if not exists
	if username == "me" {
		u, err = user.LoadByID(ctx, api.mustDB(), consumer.AuthentifiedUserID)
	} else {
		u, err = user.LoadByUsername(ctx, api.mustDB(), username)
	}
	if err != nil {
		return sdk.NewErrorWithStack(err, sdk.WrapError(sdk.ErrForbidden, "not authorized for user %s", username))
	}

	// Valid if the current consumer match given username
	if consumer.AuthentifiedUserID == u.ID {
		log.Debug("checkUserPermissions> %s read/write access granted to %s because itself", getAPIConsumer(ctx).ID, u.ID)
		return nil
	}

	// If the current user is a maintainer and we want to read a user
	if permissionValue == sdk.PermissionRead && isMaintainer(ctx) {
		log.Debug("checkUserPermissions> %s read access granted to %s because is maintainer", getAPIConsumer(ctx).ID, u.ID)
		return nil
	}

	// If the current user is an admin, gives RW on the user
	if isAdmin(ctx) {
		log.Debug("checkUserPermissions> %s read/write access granted to %s because is admin", getAPIConsumer(ctx).ID, u.ID)
		return nil
	}

	log.Debug("checkUserPermissions> %s is not authorized to %s", getAPIConsumer(ctx).ID, u.ID)
	return sdk.WrapError(sdk.ErrForbidden, "not authorized for user %s", username)
}

func (api *API) checkConsumerPermissions(ctx context.Context, consumerID string, permissionValue int, routeVars map[string]string) error {
	if consumerID == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given consumer id")
	}

	authConsumer := getAPIConsumer(ctx)
	consumer, err := authentication.LoadConsumerByID(ctx, api.mustDB(), consumerID)
	if err != nil {
		return sdk.NewErrorWithStack(err, sdk.WrapError(sdk.ErrForbidden, "not authorized for consumer %s", consumerID))
	}

	// If current consumer's authentified user match given one
	if consumer.AuthentifiedUserID == authConsumer.AuthentifiedUserID {
		log.Debug("checkConsumerPermissions> %s access granted to %s because is owner", authConsumer.ID, consumer.ID)
		return nil
	}

	// If the current user is a maintainer and we want to read a consumer
	if permissionValue == sdk.PermissionRead && isMaintainer(ctx) {
		log.Debug("checkConsumerPermissions> %s read access granted to %s because is maintainer", authConsumer.ID, consumer.ID)
		return nil
	}

	// If the current user is an admin, gives RW on the consumer
	if isAdmin(ctx) {
		log.Debug("checkConsumerPermissions> %s read/write access granted to %s because is admin", authConsumer.ID, consumer.ID)
		return nil
	}

	log.Debug("checkConsumerPermissions> %s is not authorized to %s", authConsumer.ID, consumer.ID)
	return sdk.WrapError(sdk.ErrForbidden, "not authorized for consumer %s", consumerID)
}

func (api *API) checkSessionPermissions(ctx context.Context, sessionID string, permissionValue int, routeVars map[string]string) error {
	if sessionID == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given session id")
	}

	authConsumer := getAPIConsumer(ctx)
	session, err := authentication.LoadSessionByID(ctx, api.mustDB(), sessionID)
	if err != nil {
		return sdk.NewErrorWithStack(err, sdk.WrapError(sdk.ErrForbidden, "not authorized for session %s", sessionID))
	}
	consumer, err := authentication.LoadConsumerByID(ctx, api.mustDB(), session.ConsumerID)
	if err != nil {
		return sdk.NewErrorWithStack(err, sdk.WrapError(sdk.ErrForbidden, "not authorized for session %s", sessionID))
	}

	// If current consumer's authentified user match session's consumer
	if consumer.AuthentifiedUserID == authConsumer.AuthentifiedUserID {
		log.Debug("checkSessionPermissions> %s access granted to %s because is owner", authConsumer.ID, session.ID)
		return nil
	}

	// If the current user is a maintainer and we want to read a session
	if permissionValue == sdk.PermissionRead && isMaintainer(ctx) {
		log.Debug("checkSessionPermissions> %s read access granted to %s because is maintainer", authConsumer.ID, session.ID)
		return nil
	}

	// If the current user is an admin, gives RW on the session
	if isAdmin(ctx) {
		log.Debug("checkSessionPermissions> %s read/write access granted to %s because is admin", authConsumer.ID, session.ID)
		return nil
	}

	log.Debug("checkSessionPermissions> %s is not authorized to %s", authConsumer.ID, session.ID)
	return sdk.WrapError(sdk.ErrForbidden, "not authorized for session %s", sessionID)
}
