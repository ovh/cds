package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) postVCSOnProjectHandler() (service.Handler, []service.RbacChecker) {
	handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		pKey := vars["projectKey"]

		// my handler
		log.Info(ctx, "My project: %s", pKey)

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		// TODO

		return nil
	}
	permChecker := []service.RbacChecker{
		rbac.ProjectExist,
		rbac.ProjectManage,
	}
	return handler, permChecker
}
