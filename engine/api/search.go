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

func parseSearchQuery(values url.Values) (string, uint, uint) {
	var query string
	var offset, limit uint = 0, 10

	for k, v := range values {
		switch k {
		case "query":
			query = v[0]
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

	return query, offset, limit
}

func (api *API) getSearchHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			query, offset, limit := parseSearchQuery(r.URL.Query())

			if !isAdmin(ctx) {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			var pKeys []string
			if isAdmin(ctx) {
				// For admin
				ps, err := project.LoadAll(ctx, api.mustDB(), api.Cache)
				if err != nil {
					return err
				}
				for i := range ps {
					pKeys = append(pKeys, ps[i].Key)
				}
			} else {
				// Normal user
				var err error
				pKeys, err = rbac.LoadAllProjectKeysAllowed(ctx, api.mustDB(), sdk.ProjectRoleRead, u.AuthConsumerUser.AuthentifiedUserID)
				if err != nil {
					return err
				}
			}

			count, err := search.CountAll(ctx, api.mustDB(), search.SearchFilters{
				Projects: pKeys,
				Query:    query,
			})
			if err != nil {
				return err
			}

			res, err := search.SearchAll(ctx, api.mustDB(), search.SearchFilters{
				Projects: pKeys,
				Query:    query,
			}, offset, limit)
			if err != nil {
				return err
			}

			w.Header().Add("X-Total-Count", fmt.Sprintf("%d", count))

			return service.WriteJSON(w, res, http.StatusOK)
		}
}
