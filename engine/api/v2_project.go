package api

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/rockbears/log"
	"net/http"

	"github.com/ovh/cds/engine/service"
)

func (api *API) addVCSOnProjectHandler() (service.Handler, []service.RbacChecker) {
	handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		pKey := vars["projKey"]

		// my handler
		log.Info(ctx, "My project: %s", pKey)

		return nil
	}
	permChecker := []service.RbacChecker{
		rbac.ProjectExist,
		rbac.ProjectManage,
	}
	return handler, permChecker
}
