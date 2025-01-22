package api

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func (api *API) getVCSByIdentifier(ctx context.Context, projectKey string, vcsIdentifier string, opts ...gorpmapping.GetOptionFunc) (*sdk.VCSProject, error) {
	ctx, next := telemetry.Span(ctx, "api.getVCSByIdentifier")
	defer next()
	var vcsProject *sdk.VCSProject
	var err error
	if sdk.IsValidUUID(vcsIdentifier) {
		vcsProject, err = vcs.LoadVCSByIDAndProjectKey(ctx, api.mustDB(), projectKey, vcsIdentifier, opts...)
	} else {
		vcsProject, err = vcs.LoadVCSByProject(ctx, api.mustDB(), projectKey, vcsIdentifier, opts...)
	}
	if err != nil {
		return nil, err
	}
	return vcsProject, nil
}

func (api *API) postVCSProjectHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			project, err := project.Load(ctx, tx, pKey, project.LoadOptions.WithKeys)
			if err != nil {
				return sdk.WithStack(err)
			}

			var vcsProject sdk.VCSProject
			if err := service.UnmarshalRequest(ctx, req, &vcsProject); err != nil {
				return err
			}

			if err := vcsProject.Lint(*project); err != nil {
				return err
			}

			vcsProject.ProjectID = project.ID
			vcsProject.CreatedBy = getUserConsumer(ctx).GetUsername()

			if err := vcs.Insert(ctx, tx, &vcsProject); err != nil {
				return err
			}

			vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, pKey, vcsProject.Name)
			if err != nil {
				return err
			}

			if _, err := vcsClient.Repos(ctx); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			event_v2.PublishVCSEvent(ctx, api.Cache, sdk.EventVCSCreated, pKey, vcsProject, *u.AuthConsumerUser.AuthentifiedUser)

			return service.WriteMarshal(w, req, vcsProject, http.StatusCreated)
		}
}

func (api *API) putVCSProjectHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey, project.LoadOptions.WithKeys)
			if err != nil {
				return err
			}

			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}

			vcsOld, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			var vcsProject sdk.VCSProject
			if err := service.UnmarshalRequest(ctx, req, &vcsProject); err != nil {
				return err
			}

			vcsProject.ID = vcsOld.ID
			vcsProject.Created = vcsOld.Created
			vcsProject.CreatedBy = vcsOld.CreatedBy
			vcsProject.ProjectID = vcsOld.ProjectID

			if err := vcsProject.Lint(*proj); err != nil {
				return err
			}

			if err := vcs.Update(ctx, tx, &vcsProject); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			event_v2.PublishVCSEvent(ctx, api.Cache, sdk.EventVCSUpdated, proj.Key, vcsProject, *u.AuthConsumerUser.AuthentifiedUser)

			return service.WriteMarshal(w, req, vcsProject, http.StatusCreated)
		}
}

func (api *API) deleteVCSProjectHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, _ http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			project, err := project.Load(ctx, tx, pKey)
			if err != nil {
				return sdk.WithStack(err)
			}

			if err := vcs.Delete(tx, project.ID, vcsProject.Name); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			event_v2.PublishVCSEvent(ctx, api.Cache, sdk.EventVCSDeleted, project.Key, *vcsProject, *u.AuthConsumerUser.AuthentifiedUser)

			return nil
		}
}

// getVCSProjectAllHandler returns list of vcs of one project key
func (api *API) getVCSProjectAllHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			pKey := vars["projectKey"]

			vcsProjects, err := vcs.LoadAllVCSByProject(ctx, api.mustDB(), pKey)
			if err != nil {
				return sdk.WrapError(err, "unable to load vcs server on project %s", pKey)
			}

			return service.WriteJSON(w, vcsProjects, http.StatusOK)
		}
}

func (api *API) getVCSProjectHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			pKey := vars["projectKey"]

			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
			if err != nil {
				return err
			}

			vcsClear, err := vcs.LoadVCSByIDAndProjectKey(ctx, api.mustDB(), pKey, vcsProject.ID, gorpmapping.GetOptions.WithDecryption)
			if err != nil {
				return err
			}
			vcsProject.Auth.Username = vcsClear.Auth.Username
			vcsProject.Auth.SSHKeyName = vcsClear.Auth.SSHKeyName
			vcsProject.Auth.GPGKeyName = vcsClear.Auth.GPGKeyName
			vcsProject.Auth.EmailAddress = vcsClear.Auth.EmailAddress
			vcsProject.Auth.SSHUsername = vcsClear.Auth.SSHUsername
			vcsProject.Auth.SSHPort = vcsClear.Auth.SSHPort
			return service.WriteMarshal(w, r, vcsProject, http.StatusOK)
		}
}

func (api *API) GetVCSPGKeyHandler() ([]service.RbacChecker, service.Handler) {
	return nil,
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			keyID := vars["gpgKeyID"]

			k, err := project.LoadKeyByLongKeyID(ctx, api.mustDB(), keyID)
			if err != nil {
				return err
			}

			p, err := project.LoadByID(api.mustDB(), k.ProjectID)
			if err != nil {
				return err
			}

			allvcs, err := vcs.LoadAllVCSByProject(ctx, api.mustDB(), p.Key, gorpmapper.GetOptions.WithDecryption)
			if err != nil {
				return err
			}

			var selectedVCS []sdk.VCSProject
			for _, v := range allvcs {
				v.Auth.Token = "" // we are sure we don't need this
				v.Auth.SSHPrivateKey = ""

				log.Debug(ctx, "%s = %s", v.Auth.GPGKeyName, k.Name)
				if v.Auth.GPGKeyName == k.Name {
					selectedVCS = append(selectedVCS, v)
				}
			}

			if len(selectedVCS) == 0 {
				return sdk.WithStack(sdk.ErrNotFound)
			}

			var results []sdk.VCSUserGPGKey
			for _, vcs := range selectedVCS {
				results = append(results, sdk.VCSUserGPGKey{
					ProjectKey:     p.Key,
					VCSProjectName: vcs.Name,
					Username:       vcs.Auth.Username,
					KeyName:        vcs.Auth.GPGKeyName,
					KeyID:          k.LongKeyID,
					PublicKey:      k.Public,
				})
			}

			return service.WriteMarshal(w, r, results, http.StatusOK)
		}
}
