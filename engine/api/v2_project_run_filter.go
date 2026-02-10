package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getProjectRunFiltersHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			projectKey := vars["projectKey"]

			filters, err := project.LoadRunFiltersByProjectKey(ctx, api.mustDB(), projectKey)
			if err != nil {
				return err
			}

			return service.WriteJSON(w, filters, http.StatusOK)
		}
}

func (api *API) postProjectRunFilterHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			projectKey := vars["projectKey"]

			var filter sdk.ProjectRunFilter
			if err := service.UnmarshalBody(r, &filter); err != nil {
				return err
			}

			filter.ProjectKey = projectKey // force depuis l'URL

			if err := filter.Check(); err != nil {
				return err
			}

			// Calculer le prochain order (max + 1)
			existingFilters, err := project.LoadRunFiltersByProjectKey(ctx, api.mustDB(), projectKey)
			if err != nil {
				return err
			}
			maxOrder := int64(-1)
			for _, f := range existingFilters {
				if f.Order > maxOrder {
					maxOrder = f.Order
				}
			}
			filter.Order = maxOrder + 1

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback()

			if err := project.InsertRunFilter(ctx, tx, &filter); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			return service.WriteJSON(w, filter, http.StatusCreated)
		}
}

func (api *API) putProjectRunFilterHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			projectKey := vars["projectKey"]
			filterName := vars["filterName"]

			var filterUpdate sdk.ProjectRunFilter
			if err := service.UnmarshalBody(r, &filterUpdate); err != nil {
				return err
			}

			// Charger le filtre existant
			existingFilter, err := project.LoadRunFilterByNameAndProjectKey(ctx, api.mustDB(), projectKey, filterName)
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback()

			// Actuellement, seul le champ order est modifiable
			if err := project.UpdateRunFilterOrder(ctx, tx, projectKey, existingFilter.Name, filterUpdate.Order); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			// Recharger le filtre mis à jour
			updatedFilter, err := project.LoadRunFilterByNameAndProjectKey(ctx, api.mustDB(), projectKey, existingFilter.Name)
			if err != nil {
				return err
			}

			return service.WriteJSON(w, updatedFilter, http.StatusOK)
		}
}

func (api *API) deleteProjectRunFilterHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			projectKey := vars["projectKey"]
			filterName := vars["filterName"]

			// Vérifier que le filtre existe
			filter, err := project.LoadRunFilterByNameAndProjectKey(ctx, api.mustDB(), projectKey, filterName)
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback()

			if err := project.DeleteRunFilter(tx, projectKey, filter.ID); err != nil {
				return err
			}

			// Recalculer les ordres des filtres restants
			if err := project.RecomputeRunFilterOrder(ctx, tx, projectKey); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			return service.WriteJSON(w, nil, http.StatusNoContent)
		}
}
