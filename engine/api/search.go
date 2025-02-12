package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/search"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func parseSearchQuery(values url.Values) (search.SearchFilters, uint, uint) {
	var filters search.SearchFilters
	var offset, limit uint = 0, 10

	for k, v := range values {
		switch k {
		case "project":
			filters.Projects = v
		case "type":
			filters.Types = v
		case "query":
			filters.Query = v[0]
		case "offset":
			value, _ := strconv.ParseUint(v[0], 10, 0)
			offset = uint(value)
		case "limit":
			value, _ := strconv.ParseUint(v[0], 10, 0)
			limit = uint(value)
		}
	}

	if limit > 100 {
		limit = 100
	}

	return filters, offset, limit
}

func (api *API) getSearchHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			filters, offset, limit := parseSearchQuery(r.URL.Query())

			var pKeys []string
			var err error
			if isMaintainer(ctx) {
				pKeys, err = project.LoadAllProjectKeys(ctx, api.mustDB(), api.Cache)
				if err != nil {
					return err
				}
			} else {
				pKeys, err = rbac.LoadAllProjectKeysAllowed(ctx, api.mustDB(), sdk.ProjectRoleRead, u.AuthConsumerUser.AuthentifiedUserID)
				if err != nil {
					return err
				}
			}

			var pKeysFiltered []string
			if len(filters.Projects) > 0 {
				for i := range filters.Projects {
					var found bool
					for j := range pKeys {
						if filters.Projects[i] == pKeys[j] {
							found = true
							break
						}
					}
					if found {
						pKeysFiltered = append(pKeysFiltered, filters.Projects[i])
					}
				}
			} else {
				pKeysFiltered = pKeys
			}

			count, err := search.CountAll(ctx, api.mustDB(), search.SearchFilters{
				Projects: pKeysFiltered,
				Types:    filters.Types,
				Query:    filters.Query,
			})
			if err != nil {
				return err
			}

			res, err := search.SearchAll(ctx, api.mustDB(), search.SearchFilters{
				Projects: pKeysFiltered,
				Types:    filters.Types,
				Query:    filters.Query,
			}, offset, limit)
			if err != nil {
				return err
			}

			w.Header().Add("X-Total-Count", fmt.Sprintf("%d", count))

			return service.WriteJSON(w, res, http.StatusOK)
		}
}

func (api *API) getSearchFiltersHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			var pKeys []string
			var err error
			if isMaintainer(ctx) {
				pKeys, err = project.LoadAllProjectKeys(ctx, api.mustDB(), api.Cache)
				if err != nil {
					return err
				}
			} else {
				pKeys, err = rbac.LoadAllProjectKeysAllowed(ctx, api.mustDB(), sdk.ProjectRoleRead, u.AuthConsumerUser.AuthentifiedUserID)
				if err != nil {
					return err
				}
			}

			filters := []sdk.SearchFilter{
				{
					Key:     "project",
					Options: pKeys,
					Example: "KEY",
				},
				{
					Key: "type",
					Options: []string{
						string(sdk.ProjectSearchResultType),
						string(sdk.WorkflowSearchResultType),
						string(sdk.WorkflowLegacySearchResultType),
					},
					Example: "project, workflow, etc.",
				},
			}

			return service.WriteJSON(w, filters, http.StatusOK)
		}
}
