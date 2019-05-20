package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/sdk"
)

func isGroupAdmin(ctx context.Context, g *sdk.Group) bool {
	u := getAPIConsumer(ctx)
	return g.IsMember(g, u)
}

func isGroupMember(ctx context.Context, g *sdk.Group) bool {
	u := getAPIConsumer(ctx)
	return g.IsMember(g, u)
}

func isMaintainer(ctx context.Context) bool {
	u := getAPIConsumer(ctx)
	return u.Maintainer()
}

func isAdmin(ctx context.Context) bool {
	u := getAPIConsumer(ctx)
	return u.Admin()
}

func getAPIConsumer(c context.Context) *sdk.APIConsumer {
	i := c.Value(auth.ContextAPIConsumer)
	if i == nil {
		return nil
	}
	u, ok := i.(*sdk.APIConsumer)
	if !ok {
		return nil
	}
	return u
}

func getHandlerScope(c context.Context) HandlerScope {
	i := c.Value(auth.ContextScope)
	if i == nil {
		return nil
	}
	u, ok := i.(HandlerScope)
	if !ok {
		return nil
	}
	return u
}

func JWT(c context.Context) *sdk.AccessToken {
	i := c.Value(auth.ContextJWT)
	if i == nil {
		return nil
	}
	u, ok := i.(*sdk.AccessToken)
	if !ok {
		return nil
	}
	return u
}

func getProvider(c context.Context) *string {
	i := c.Value(auth.ContextProvider)
	if i == nil {
		return nil
	}
	u, ok := i.(string)
	if !ok {
		return nil
	}
	return &u
}

func getAgent(r *http.Request) string {
	return r.Header.Get("User-Agent")
}

func isServiceOrWorker(r *http.Request) bool {
	switch getAgent(r) {
	case sdk.ServiceAgent:
		return true
	case sdk.WorkerAgent:
		return true
	default:
		return false
	}
}

func getWorker(c context.Context) *sdk.Worker {
	i := c.Value(auth.ContextWorker)
	if i == nil {
		return nil
	}
	u, ok := i.(*sdk.Worker)
	if !ok {
		return nil
	}
	return u
}

func getHatchery(c context.Context) *sdk.Service {
	i := c.Value(auth.ContextHatchery)
	if i == nil {
		return nil
	}
	u, ok := i.(*sdk.Service)
	if !ok {
		return nil
	}
	return u
}

func getService(c context.Context) *sdk.Service {
	i := c.Value(auth.ContextService)
	if i == nil {
		return nil
	}
	u, ok := i.(*sdk.Service)
	if !ok {
		return nil
	}
	return u
}

func (a *API) mustDB() *gorp.DbMap {
	db := a.DBConnectionFactory.GetDBMap()
	if db == nil {
		panic(fmt.Errorf("Database unavailable"))
	}
	return db
}

func (a *API) mustDBWithCtx(ctx context.Context) *gorp.DbMap {
	db := a.DBConnectionFactory.GetDBMap()
	db = db.WithContext(ctx).(*gorp.DbMap)
	if db == nil {
		panic(fmt.Errorf("Database unavailable"))
	}

	return db
}
