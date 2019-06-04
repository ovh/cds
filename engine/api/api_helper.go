package api

import (
	"context"
	"fmt"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk/log"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
)

func isGroupAdmin(ctx context.Context, g *sdk.Group) bool {
	u := getAPIConsumer(ctx)
	return g.IsMember(u) && g.IsAdmin(u.OnBehalfOf)
}

func isGroupMember(ctx context.Context, g *sdk.Group) bool {
	u := getAPIConsumer(ctx)
	return g.IsMember(u)
}

func isMaintainer(ctx context.Context) bool {
	u := getAPIConsumer(ctx)
	return u.Maintainer() || u.Admin()
}

func isAdmin(ctx context.Context) bool {
	u := getAPIConsumer(ctx)
	return u.Admin()
}

func getAPIConsumer(c context.Context) *sdk.APIConsumer {
	i := c.Value(contextAPIConsumer)
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
	i := c.Value(contextScope)
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
	i := c.Value(contextJWT)
	if i == nil {
		return nil
	}
	u, ok := i.(*sdk.AccessToken)
	if !ok {
		return nil
	}
	return u
}

func JWTRaw(c context.Context) string {
	i := c.Value(contextJWTRaw)
	if i == nil {
		return ""
	}
	u, ok := i.(string)
	if !ok {
		return ""
	}
	return u
}

func getProvider(c context.Context) *string {
	i := c.Value(contextProvider)
	if i == nil {
		return nil
	}
	u, ok := i.(string)
	if !ok {
		return nil
	}
	return &u
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

func (a *API) isWorker(ctx context.Context) (*sdk.Worker, bool) {
	db := a.mustDBWithCtx(ctx)
	t := JWT(ctx)
	if t == nil {
		return nil, false
	}
	w, err := worker.LoadByAccessTokenID(ctx, db, t.ID)
	if err != nil {
		log.Error("unable to get worker from token %s: %v", t.ID, err)
		return nil, false
	}
	if w == nil {
		return nil, false
	}
	return w, true
}

func (a *API) isHatchery(ctx context.Context) (*sdk.Service, bool) {
	db := a.mustDBWithCtx(ctx)
	t := JWT(ctx)
	if t == nil {
		return nil, false
	}
	s, err := services.FindByTokenID(db, t.ID)
	if err != nil {
		log.Error("unable to get hatchery from token %s: %v", t.ID, err)
		return nil, false
	}
	if s.Type != services.TypeHatchery {
		return nil, false
	}
	return s, true
}
