package api

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getActionsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var as []sdk.Action
		var err error
		if isMaintainer(ctx) {
			as, err = action.LoadAllByTypes(ctx, api.mustDB(),
				[]string{sdk.DefaultAction},
				action.LoadOptions.WithRequirements,
				action.LoadOptions.WithParameters,
				action.LoadOptions.WithGroup,
				action.LoadOptions.WithAudits,
			)
		} else {
			as, err = action.LoadAllTypeDefaultByGroupIDs(ctx, api.mustDB(),
				append(getAPIConsumer(ctx).GetGroupIDs(), group.SharedInfraGroup.ID),
				action.LoadOptions.WithRequirements,
				action.LoadOptions.WithParameters,
				action.LoadOptions.WithGroup,
				action.LoadOptions.WithAudits,
			)
		}
		if err != nil {
			return err
		}

		return service.WriteJSON(w, as, http.StatusOK)
	}
}

func (api *API) getActionsForProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		proj, err := project.Load(ctx, api.mustDB(), key, project.LoadOptions.WithGroups)
		if err != nil {
			return sdk.WrapError(err, "unable to load projet %s", key)
		}

		groupIDs := make([]int64, len(proj.ProjectGroups))
		for i := range proj.ProjectGroups {
			groupIDs[i] = proj.ProjectGroups[i].Group.ID
		}

		as, err := action.LoadAllTypeBuiltInOrPluginOrDefaultForGroupIDs(ctx, api.mustDB(),
			append(groupIDs, group.SharedInfraGroup.ID),
			action.LoadOptions.WithRequirements,
			action.LoadOptions.WithParameters,
			action.LoadOptions.WithGroup,
		)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, as, http.StatusOK)
	}
}

func (api *API) getActionsForGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["permGroupName"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName, group.LoadOptions.WithMembers)
		if err != nil {
			return err
		}

		// and user is part of the group
		if !isGroupMember(ctx, g) && !isMaintainer(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		as, err := action.LoadAllTypeBuiltInOrPluginOrDefaultForGroupIDs(ctx, api.mustDB(),
			[]int64{g.ID, group.SharedInfraGroup.ID},
			action.LoadOptions.WithRequirements,
			action.LoadOptions.WithParameters,
			action.LoadOptions.WithGroup,
		)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, as, http.StatusOK)
	}
}

func (api *API) postActionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var data sdk.Action
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}
		if err := data.IsValidDefault(); err != nil {
			return err
		}

		grp, err := group.LoadByID(ctx, api.mustDB(), *data.GroupID, group.LoadOptions.WithMembers)
		if err != nil {
			return err
		}

		if !isGroupAdmin(ctx, grp) && !isAdmin(ctx) {
			return sdk.WithStack(sdk.ErrInvalidGroupAdmin)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		// check that no action already exists for same group/name
		current, err := action.LoadTypeDefaultByNameAndGroupID(ctx, tx, data.Name, grp.ID)
		if err != nil {
			return err
		}
		if current != nil {
			return sdk.NewErrorFrom(sdk.ErrAlreadyExist, "an action already exists for given name on this group")
		}

		// only default action can be posted or updated
		data.Type = sdk.DefaultAction

		// check that given children exists and can be used
		if err := action.CheckChildrenForGroupIDs(ctx, tx, &data, []int64{group.SharedInfraGroup.ID, grp.ID}); err != nil {
			return err
		}

		// inserts action and components
		if err := action.Insert(tx, &data); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		newAction, err := action.LoadByID(ctx, api.mustDB(), data.ID, action.LoadOptions.Default)
		if err != nil {
			return err
		}

		event.PublishActionAdd(ctx, *newAction, getAPIConsumer(ctx))

		if err := action.LoadOptions.WithAudits(ctx, api.mustDB(), newAction); err != nil {
			return err
		}

		// aggregate extra data for ui
		newAction.Editable = true

		return service.WriteJSON(w, newAction, http.StatusCreated)
	}
}

func (api *API) getActionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["permGroupName"]
		actionName := vars["permActionName"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName, group.LoadOptions.WithMembers)
		if err != nil {
			return err
		}

		a, err := action.LoadTypeDefaultByNameAndGroupID(ctx, api.mustDB(), actionName, g.ID,
			action.LoadOptions.Default,
			action.LoadOptions.WithAudits,
		)
		if err != nil {
			return err
		}
		if a == nil {
			return sdk.WithStack(sdk.ErrNoAction)
		}

		if isGroupAdmin(ctx, g) || isAdmin(ctx) {
			a.Editable = true
		}

		return service.WriteJSON(w, a, http.StatusOK)
	}
}

func (api *API) putActionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["permGroupName"]
		actionName := vars["permActionName"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName)
		if err != nil {
			return err
		}

		old, err := action.LoadTypeDefaultByNameAndGroupID(ctx, api.mustDB(), actionName, g.ID, action.LoadOptions.Default)
		if err != nil {
			return err
		}
		if old == nil {
			return sdk.WithStack(sdk.ErrNoAction)
		}

		var data sdk.Action
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}
		if err := data.IsValidDefault(); err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot begin transaction")
		}
		defer tx.Rollback() // nolint

		grp, err := group.LoadByID(ctx, tx, *data.GroupID, group.LoadOptions.WithMembers)
		if err != nil {
			return err
		}

		if *old.GroupID != *data.GroupID || old.Name != data.Name {
			if !isGroupAdmin(ctx, grp) && !isAdmin(ctx) {
				return sdk.WithStack(sdk.ErrInvalidGroupAdmin)
			}

			// check that no action already exists for same group/name
			current, err := action.LoadTypeDefaultByNameAndGroupID(ctx, tx, data.Name, grp.ID)
			if err != nil {
				return err
			}
			if current != nil {
				return sdk.NewErrorFrom(sdk.ErrAlreadyExist, "an action already exists for given name on this group")
			}
		}

		// only default action can be posted or updated
		data.ID = old.ID
		data.Type = sdk.DefaultAction

		// check that given children exists and can be used, and no loop exists
		if err := action.CheckChildrenForGroupIDsWithLoop(ctx, tx, &data, []int64{group.SharedInfraGroup.ID, grp.ID}); err != nil {
			return err
		}

		if err = action.Update(tx, &data); err != nil {
			return sdk.WrapError(err, "cannot update action")
		}

		if err = tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		newAction, err := action.LoadByID(ctx, api.mustDB(), data.ID, action.LoadOptions.Default)
		if err != nil {
			return err
		}

		event.PublishActionUpdate(ctx, *old, *newAction, getAPIConsumer(ctx))

		if err := action.LoadOptions.WithAudits(ctx, api.mustDB(), newAction); err != nil {
			return err
		}

		// aggregate extra data for ui
		newAction.Editable = true

		return service.WriteJSON(w, newAction, http.StatusOK)
	}
}

func (api *API) deleteActionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["permGroupName"]
		actionName := vars["permActionName"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName)
		if err != nil {
			return err
		}

		a, err := action.LoadTypeDefaultByNameAndGroupID(ctx, api.mustDB(), actionName, g.ID)
		if err != nil {
			return err
		}
		if a == nil {
			return sdk.WithStack(sdk.ErrNoAction)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		used, err := action.Used(tx, a.ID)
		if err != nil {
			return err
		}
		if used {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "cannot delete action %s is used in other actions or pipelines", a.Name)
		}

		if err := action.Delete(tx, a); err != nil {
			return sdk.WrapError(err, "cannot delete action %s", a.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return nil
	}
}

func (api *API) getActionAuditHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["permGroupName"]
		actionName := vars["permActionName"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName)
		if err != nil {
			return err
		}

		a, err := action.LoadTypeDefaultByNameAndGroupID(ctx, api.mustDB(), actionName, g.ID)
		if err != nil {
			return err
		}
		if a == nil {
			return sdk.WithStack(sdk.ErrNoAction)
		}

		as, err := action.GetAuditsByActionID(api.mustDB(), a.ID)
		if err != nil {
			return err
		}

		// convert all audits to export entities yaml
		converted := make([]sdk.AuditAction, 0, len(as))
		for i := range as {
			clone := as[i]
			clone.DataType = "yaml"

			if clone.DataBefore != "" {
				var before sdk.Action
				if err := json.Unmarshal([]byte(clone.DataBefore), &before); err != nil {
					log.Error(ctx, "%+v", sdk.WrapError(err, "cannot parse action audit"))
					continue
				}

				ea := exportentities.NewAction(before)
				buf, err := yaml.Marshal(ea)
				if err != nil {
					log.Error(ctx, "%+v", sdk.WrapError(err, "cannot parse action audit"))
					continue
				}

				clone.DataBefore = string(buf)
			}

			if clone.DataAfter != "" {
				var after sdk.Action
				if err := json.Unmarshal([]byte(clone.DataAfter), &after); err != nil {
					log.Error(ctx, "%+v", sdk.WrapError(err, "cannot parse action audit"))
					continue
				}

				ea := exportentities.NewAction(after)
				buf, err := yaml.Marshal(ea)
				if err != nil {
					log.Error(ctx, "%+v", sdk.WrapError(err, "cannot parse action audit"))
					continue
				}

				clone.DataAfter = string(buf)
			}

			converted = append(converted, clone)
		}

		return service.WriteJSON(w, converted, http.StatusOK)
	}
}

func (api *API) postActionAuditRollbackHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["permGroupName"]
		actionName := vars["permActionName"]

		auditID, err := requestVarInt(r, "auditID")
		if err != nil {
			return err
		}

		grp, err := group.LoadByName(ctx, api.mustDB(), groupName)
		if err != nil {
			return err
		}

		old, err := action.LoadTypeDefaultByNameAndGroupID(ctx, api.mustDB(), actionName, grp.ID, action.LoadOptions.Default)
		if err != nil {
			return err
		}
		if old == nil {
			return sdk.WithStack(sdk.ErrNoAction)
		}

		aa, err := action.GetAuditByActionIDAndID(ctx, api.mustDB(), old.ID, auditID)
		if err != nil {
			return err
		}
		if aa == nil {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		var before sdk.Action
		if err := json.Unmarshal([]byte(aa.DataBefore), &before); err != nil {
			return sdk.WrapError(err, "cannot parse action audit")
		}

		ea := exportentities.NewAction(before)

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		// set group id on given action, if no group given use shared.infra fo backward compatibility
		// current user should be admin if the group
		var newGrp *sdk.Group
		if ea.Group == sdk.SharedInfraGroupName || ea.Group == "" {
			newGrp = group.SharedInfraGroup
		} else if ea.Group == grp.Name {
			newGrp = grp
		} else {
			newGrp, err = group.LoadByName(ctx, tx, ea.Group, group.LoadOptions.WithMembers)
			if err != nil {
				return err
			}
		}

		if grp.ID != newGrp.ID || old.Name != ea.Name {
			if !isGroupAdmin(ctx, grp) && !isAdmin(ctx) {
				return sdk.WithStack(sdk.ErrInvalidGroupAdmin)
			}

			// check that no action already exists for same group/name
			current, err := action.LoadTypeDefaultByNameAndGroupID(ctx, tx, ea.Name, newGrp.ID)
			if err != nil {
				return err
			}
			if current != nil {
				return sdk.NewErrorFrom(sdk.ErrAlreadyExist, "an action already exists for given name on this group")
			}
		}

		data, err := ea.GetAction()
		if err != nil {
			return err
		}

		data.GroupID = &newGrp.ID

		// set action id for children based on action name and group name
		// if no group name given for child, first search an action for shared.infra for backward compatibility
		// else search a builtin or plugin action
		for i := range data.Actions {
			a, err := action.RetrieveForGroupAndName(ctx, tx, data.Actions[i].Group, data.Actions[i].Name)
			if err != nil {
				return err
			}
			data.Actions[i].ID = a.ID
		}

		// check data validity
		if err := data.IsValidDefault(); err != nil {
			return err
		}

		data.ID = old.ID

		// check that given children exists and can be used, and no loop exists
		if err := action.CheckChildrenForGroupIDsWithLoop(ctx, tx, &data, []int64{group.SharedInfraGroup.ID, newGrp.ID}); err != nil {
			return err
		}

		if err = action.Update(tx, &data); err != nil {
			return sdk.WrapError(err, "cannot update action")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		newAction, err := action.LoadByID(ctx, api.mustDB(), data.ID, action.LoadOptions.Default)
		if err != nil {
			return err
		}

		event.PublishActionUpdate(ctx, *old, *newAction, getAPIConsumer(ctx))

		if err := action.LoadOptions.WithAudits(ctx, api.mustDB(), newAction); err != nil {
			return err
		}

		// aggregate extra data for ui
		newAction.Editable = true

		return service.WriteJSON(w, newAction, http.StatusOK)
	}
}

func (api *API) getActionUsageHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["permGroupName"]
		actionName := vars["permActionName"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName)
		if err != nil {
			return err
		}

		a, err := action.LoadTypeDefaultByNameAndGroupID(ctx, api.mustDB(), actionName, g.ID)
		if err != nil {
			return err
		}
		if a == nil {
			return sdk.WithStack(sdk.ErrNoAction)
		}

		u, err := getActionUsage(ctx, api.mustDB(), api.Cache, a)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, u, http.StatusOK)
	}
}

func (api *API) getActionExportHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["permGroupName"]
		actionName := vars["permActionName"]

		format := FormString(r, "format")
		if format == "" {
			format = "yaml"
		}
		f, err := exportentities.GetFormat(format)
		if err != nil {
			return err
		}

		g, err := group.LoadByName(ctx, api.mustDB(), groupName)
		if err != nil {
			return err
		}

		a, err := action.LoadTypeDefaultByNameAndGroupID(ctx, api.mustDB(), actionName, g.ID, action.LoadOptions.Default)
		if err != nil {
			return err
		}
		if a == nil {
			return sdk.WithStack(sdk.ErrNoAction)
		}

		if err := action.Export(*a, f, w); err != nil {
			return err
		}

		w.Header().Add("Content-Type", f.ContentType())
		return nil
	}
}

// importActionHandler insert OR update an existing action.
func (api *API) importActionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return err
		}
		defer r.Body.Close()

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(body)
		}
		format, err := exportentities.GetFormatFromContentType(contentType)
		if err != nil {
			return err
		}

		var ea exportentities.Action
		if err := exportentities.Unmarshal(body, format, &ea); err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		// set group id on given action, if no group given use shared.infra fo backward compatibility
		// current user should be admin if the group
		var grp *sdk.Group
		if ea.Group == sdk.SharedInfraGroupName || ea.Group == "" {
			grp = group.SharedInfraGroup
		} else {
			grp, err = group.LoadByName(ctx, tx, ea.Group, group.LoadOptions.WithMembers)
			if err != nil {
				return err
			}
		}

		if !isGroupAdmin(ctx, grp) && !isAdmin(ctx) {
			return sdk.WithStack(sdk.ErrInvalidGroupAdmin)
		}

		data, err := ea.GetAction()
		if err != nil {
			return err
		}

		data.GroupID = &grp.ID

		// set action id for children based on action name and group name
		// if no group name given for child, first search an action for shared.infra for backward compatibility
		// else search a builtin or plugin action
		for i := range data.Actions {
			a, err := action.RetrieveForGroupAndName(ctx, tx, data.Actions[i].Group, data.Actions[i].Name)
			if err != nil {
				return err
			}
			data.Actions[i].ID = a.ID
		}

		// check data validity
		if err := data.IsValidDefault(); err != nil {
			return err
		}

		// check if action exists in database
		old, err := action.LoadTypeDefaultByNameAndGroupID(ctx, tx, data.Name, grp.ID, action.LoadOptions.Default)
		if err != nil {
			return err
		}
		exists := old != nil

		// update or insert depending action if action exists
		if exists {
			data.ID = old.ID

			// check that given children exists and can be used, and no loop exists
			if err := action.CheckChildrenForGroupIDsWithLoop(ctx, tx, &data, []int64{group.SharedInfraGroup.ID, grp.ID}); err != nil {
				return err
			}

			if err = action.Update(tx, &data); err != nil {
				return sdk.WrapError(err, "cannot update action")
			}
		} else {
			// check that given children exists and can be used
			if err := action.CheckChildrenForGroupIDs(ctx, tx, &data, []int64{group.SharedInfraGroup.ID, grp.ID}); err != nil {
				return err
			}

			// inserts action and components
			if err := action.Insert(tx, &data); err != nil {
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		newAction, err := action.LoadByID(ctx, api.mustDB(), data.ID, action.LoadOptions.Default)
		if err != nil {
			return err
		}

		if exists {
			event.PublishActionUpdate(ctx, *old, *newAction, getAPIConsumer(ctx))
		} else {
			event.PublishActionAdd(ctx, *newAction, getAPIConsumer(ctx))
		}

		if err := action.LoadOptions.WithAudits(ctx, api.mustDB(), newAction); err != nil {
			return err
		}

		// aggregate extra data for ui
		newAction.Editable = true

		code := http.StatusCreated
		if exists {
			code = http.StatusOK
		}
		return service.WriteJSON(w, newAction, code)
	}
}

func (api *API) getActionsRequirements() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		rs, err := action.GetRequirementsDistinctBinary(api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "cannot load action requirements")
		}

		return service.WriteJSON(w, rs, http.StatusOK)
	}
}

func (api *API) getActionsBuiltinHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		as, err := action.LoadAllByTypes(ctx, api.mustDB(), []string{sdk.BuiltinAction, sdk.PluginAction},
			action.LoadOptions.WithRequirements,
			action.LoadOptions.WithParameters,
			action.LoadOptions.WithGroup,
		)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, as, http.StatusOK)
	}
}

func (api *API) getActionBuiltinHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		actionName := vars["permActionBuiltinName"]

		a, err := action.LoadByTypesAndName(ctx, api.mustDB(), []string{sdk.BuiltinAction, sdk.PluginAction}, actionName,
			action.LoadOptions.WithRequirements,
			action.LoadOptions.WithParameters,
			action.LoadOptions.WithGroup,
		)
		if err != nil {
			return err
		}
		if a == nil {
			return sdk.WithStack(sdk.ErrNoAction)
		}

		return service.WriteJSON(w, a, http.StatusOK)
	}
}

func (api *API) getActionBuiltinUsageHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		actionName := vars["permActionBuiltinName"]

		a, err := action.LoadByTypesAndName(ctx, api.mustDB(), []string{sdk.BuiltinAction, sdk.PluginAction}, actionName,
			action.LoadOptions.WithRequirements,
			action.LoadOptions.WithParameters,
			action.LoadOptions.WithGroup,
		)
		if err != nil {
			return err
		}
		if a == nil {
			return sdk.WithStack(sdk.ErrNoAction)
		}

		u, err := getActionUsage(ctx, api.mustDB(), api.Cache, a)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, u, http.StatusOK)
	}
}

func getActionUsage(ctx context.Context, db gorp.SqlExecutor, store cache.Store, a *sdk.Action) (sdk.ActionUsages, error) {
	var usage sdk.ActionUsages
	var err error
	usage.Pipelines, err = action.GetPipelineUsages(db, group.SharedInfraGroup.ID, a.ID)
	if err != nil {
		return usage, err
	}
	usage.Actions, err = action.GetActionUsages(db, group.SharedInfraGroup.ID, a.ID)
	if err != nil {
		return usage, err
	}

	consumer := getAPIConsumer(ctx)

	if !isMaintainer(ctx) {
		// filter usage in pipeline by user's projects
		ps, err := project.LoadAllByGroupIDs(ctx, db, store, consumer.GetGroupIDs())
		if err != nil {
			return usage, err
		}
		mProjectIDs := make(map[int64]struct{}, len(ps))
		for i := range ps {
			mProjectIDs[ps[i].ID] = struct{}{}
		}

		filteredPipelines := make([]sdk.UsagePipeline, 0, len(usage.Pipelines))
		for i := range usage.Pipelines {
			if _, ok := mProjectIDs[usage.Pipelines[i].ProjectID]; ok {
				filteredPipelines = append(filteredPipelines, usage.Pipelines[i])
			}
		}
		usage.Pipelines = filteredPipelines

		// filter usage in action by user's groups
		groupIDs := consumer.GetGroupIDs()
		mGroupIDs := make(map[int64]struct{}, len(groupIDs))
		for i := range groupIDs {
			mGroupIDs[groupIDs[i]] = struct{}{}
		}

		filteredActions := make([]sdk.UsageAction, 0, len(usage.Actions))
		for i := range usage.Actions {
			if _, ok := mGroupIDs[usage.Actions[i].GroupID]; ok {
				filteredActions = append(filteredActions, usage.Actions[i])
			}
		}
		usage.Actions = filteredActions
	}

	return usage, nil
}
