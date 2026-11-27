package api

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

// getAdminOutdatedPGPKeysEmailHandler lists, project by project, the PGP keys, default or not and disabled
// included, whose emails differ from the configured template; any error stops the listing.
func (api *API) getAdminOutdatedPGPKeysEmailHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := api.checkGPGKeyEmailTemplate(); err != nil {
			return err
		}
		projs, err := api.loadAllProjectsSortedByKey(ctx)
		if err != nil {
			return err
		}
		res := []sdk.ProjectKeyEmailStatus{}
		for _, p := range projs {
			ctx := context.WithValue(ctx, cdslog.Project, p.Key)
			projectKeys, err := project.LoadAllKeys(ctx, api.mustDB(), p.ID)
			if err != nil {
				return err
			}
			slices.SortFunc(projectKeys, func(a, b sdk.ProjectKey) int { return cmp.Compare(a.Name, b.Name) })
			for _, k := range projectKeys {
				if k.Type != sdk.KeyTypePGP {
					continue
				}
				status, outdated, err := api.pgpKeyEmailStatus(ctx, p.Key, k)
				if err != nil {
					return err
				}
				if outdated {
					res = append(res, status)
				}
			}
		}
		return service.WriteJSON(w, res, http.StatusOK)
	}
}

// postAdminUpdatePGPKeysEmailHandler sets the template email on the default PGP key of each project, one
// transaction per key; a failing project is reported and does not stop the others.
func (api *API) postAdminUpdatePGPKeysEmailHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := api.checkGPGKeyEmailTemplate(); err != nil {
			return err
		}
		projs, err := api.loadAllProjectsSortedByKey(ctx)
		if err != nil {
			return err
		}
		report := newProjectKeyEmailUpdateReport()
		for _, p := range projs {
			ctx := context.WithValue(ctx, cdslog.Project, p.Key)
			defaultName := sdk.GenerateProjectDefaultKeyName(p.Key, sdk.KeyTypePGP)
			fail := func(err error) {
				log.Error(ctx, "unable to update email of key %s: %v", defaultName, err)
				report.Failed = append(report.Failed, sdk.ProjectKeyEmailUpdateFailure{ProjectKey: p.Key, KeyName: defaultName, Error: err.Error()})
			}

			k, err := project.LoadKeyByName(ctx, api.mustDB(), p.ID, defaultName)
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				fail(err)
				continue
			}
			if k == nil || k.Type != sdk.KeyTypePGP {
				log.Warn(ctx, "project %s has no default PGP key %s", p.Key, defaultName)
				report.MissingDefaultKey = append(report.MissingDefaultKey, p.Key)
				continue
			}
			status, outdated, err := api.pgpKeyEmailStatus(ctx, p.Key, *k)
			if err != nil {
				fail(err)
				continue
			}
			if !outdated {
				continue
			}
			updated, err := api.updatePGPKeyEmail(ctx, k.ID, status)
			switch {
			case sdk.ErrorIs(err, sdk.ErrNotFound):
				report.Skipped++
			case err != nil:
				fail(err)
			case updated:
				report.Updated = append(report.Updated, status)
			default:
				report.Skipped++
			}
		}
		return service.WriteJSON(w, report, http.StatusOK)
	}
}

// postAdminUpdateProjectPGPKeyEmailHandler sets the template email on one PGP key of a project, default or not.
func (api *API) postAdminUpdateProjectPGPKeyEmailHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		pKey := vars["projectKey"]
		keyName := vars["keyName"]

		if err := api.checkGPGKeyEmailTemplate(); err != nil {
			return err
		}
		proj, err := project.Load(ctx, api.mustDB(), pKey)
		if err != nil {
			return err
		}
		key, err := project.LoadKeyByName(ctx, api.mustDB(), proj.ID, keyName)
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return sdk.NewErrorFrom(sdk.ErrNotFound, "key %s not found in project %s", keyName, pKey)
		}
		if err != nil {
			return err
		}
		if key.Type != sdk.KeyTypePGP {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "key %s is not a PGP key", keyName)
		}

		ctx = context.WithValue(ctx, cdslog.Project, proj.Key)
		status, _, err := api.pgpKeyEmailStatus(ctx, proj.Key, *key)
		if err != nil {
			return err
		}
		updated, err := api.updatePGPKeyEmail(ctx, key.ID, status)
		if err != nil {
			return err
		}
		report := newProjectKeyEmailUpdateReport()
		if updated {
			report.Updated = append(report.Updated, status)
		} else {
			report.Skipped = 1
		}
		return service.WriteJSON(w, report, http.StatusOK)
	}
}

func newProjectKeyEmailUpdateReport() sdk.ProjectKeyEmailUpdateReport {
	return sdk.ProjectKeyEmailUpdateReport{
		Updated:           []sdk.ProjectKeyEmailStatus{},
		Failed:            []sdk.ProjectKeyEmailUpdateFailure{},
		MissingDefaultKey: []string{},
	}
}

// checkGPGKeyEmailTemplate refuses to rewrite keys without a configured template: the default one is a localhost address.
func (api *API) checkGPGKeyEmailTemplate() error {
	if api.Config.Project.GPGKeyEmailAddressTemplate == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "no GPG key email address template configured on the API")
	}
	return nil
}

func (api *API) loadAllProjectsSortedByKey(ctx context.Context) ([]sdk.Project, error) {
	projs, err := project.LoadAll(ctx, api.mustDB(), api.Cache)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(projs, func(a, b sdk.Project) int { return cmp.Compare(a.Key, b.Key) })
	return projs, nil
}

// pgpKeyEmailStatus compares the emails of a public PGP key with the template one; an unreadable public key
// counts as outdated, its rewrite starting from the private key fixes it.
func (api *API) pgpKeyEmailStatus(ctx context.Context, projectKey string, k sdk.ProjectKey) (sdk.ProjectKeyEmailStatus, bool, error) {
	expected, err := api.gpgKeyEmailAddress(ctx, projectKey, k.Name)
	if err != nil {
		return sdk.ProjectKeyEmailStatus{}, false, err
	}
	status := sdk.ProjectKeyEmailStatus{
		ProjectKey:    projectKey,
		KeyName:       k.Name,
		LongKeyID:     k.LongKeyID,
		Default:       k.Name == sdk.GenerateProjectDefaultKeyName(projectKey, sdk.KeyTypePGP),
		Disabled:      k.Disabled,
		Emails:        []string{},
		ExpectedEmail: expected,
	}
	emails, err := keys.PGPKeyEmails(k.Public)
	if err != nil {
		log.Warn(ctx, "unable to read the public part of key %s: %v", k.Name, err)
		return status, true, nil
	}
	status.Emails = emails
	return status, len(emails) != 1 || emails[0] != expected, nil
}

// updatePGPKeyEmail locks the key and rewrites it with the expected email; it returns false when the key already has it.
func (api *API) updatePGPKeyEmail(ctx context.Context, id int64, status sdk.ProjectKeyEmailStatus) (bool, error) {
	tx, err := api.mustDBWithCtx(ctx).Begin()
	if err != nil {
		return false, sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	k, err := project.LoadAndLockKeyByID(ctx, tx, id)
	if err != nil {
		return false, err
	}
	updated, changed, err := keys.ReplacePGPKeyEmail(k.Private, status.ExpectedEmail)
	if err != nil || !changed {
		return false, err
	}
	// Keys stored before long key IDs existed have an empty one: the short ID is always set.
	if updated.KeyID != k.KeyID || (k.LongKeyID != "" && updated.LongKeyID != k.LongKeyID) {
		return false, sdk.WithStack(fmt.Errorf("private key %s does not match the stored key ID %s", updated.LongKeyID, k.KeyID))
	}
	k.Public = updated.Public
	k.Private = updated.Private
	if err := project.UpdateKey(tx, k); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, sdk.WithStack(err)
	}
	log.Info(ctx, "key %s [%s]: emails %v replaced by %s", k.Name, updated.LongKeyID, status.Emails, status.ExpectedEmail)
	return true, nil
}
