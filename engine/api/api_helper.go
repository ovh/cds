package api

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// group should have members aggregated and authentified user old user struct should be set.
func isGroupAdmin(ctx context.Context, g *sdk.Group) bool {
	c := getAPIConsumer(ctx)
	if c == nil {
		return false
	}
	member := g.IsMember(c.GetGroupIDs())
	admin := g.IsAdmin(*c.AuthentifiedUser)
	log.Debug("api.isGroupAdmin> member:%t admin:%t", member, admin)
	return member && admin
}

func isGroupMember(ctx context.Context, g *sdk.Group) bool {
	c := getAPIConsumer(ctx)
	if c == nil {
		return false
	}
	return g.IsMember(c.GetGroupIDs()) || g.ID == group.SharedInfraGroup.ID
}

func isMaintainer(ctx context.Context) bool {
	c := getAPIConsumer(ctx)
	if c == nil {
		return false
	}
	maintainer := c.Maintainer()
	admin := c.Admin()
	return maintainer || admin
}

func isAdmin(ctx context.Context) bool {
	c := getAPIConsumer(ctx)
	if c == nil {
		return false
	}
	return c.Admin()
}

func isService(ctx context.Context) bool {
	c := getAPIConsumer(ctx)
	if c == nil {
		return false
	}
	return c.Service != nil
}

func isWorker(ctx context.Context) bool {
	c := getAPIConsumer(ctx)
	if c == nil {
		return false
	}
	return c.Worker != nil
}

func isHatchery(ctx context.Context) bool {
	c := getAPIConsumer(ctx)
	if c == nil {
		return false
	}
	return c.Service != nil && c.Service.Type == sdk.TypeHatchery
}

func isCDN(ctx context.Context) bool {
	c := getAPIConsumer(ctx)
	if c == nil {
		return false
	}
	return c.Service != nil && c.Service.Type == sdk.TypeCDN
}

func getAPIConsumer(c context.Context) *sdk.AuthConsumer {
	i := c.Value(contextAPIConsumer)
	if i == nil {
		log.Debug("api.getAPIConsumer> no auth consumer found in context")
		return nil
	}
	consumer, ok := i.(*sdk.AuthConsumer)
	if !ok {
		return nil
	}
	return consumer
}

func getRemoteTime(c context.Context) time.Time {
	i := c.Value(contextDate)
	if i == nil {
		return time.Now()
	}
	t, ok := i.(time.Time)
	if !ok {
		return time.Now()
	}
	return t
}

func getAuthSession(c context.Context) *sdk.AuthSession {
	i := c.Value(contextSession)
	if i == nil {
		log.Debug("api.getAuthSession> no AuthSession found in context")
		return nil
	}
	u, ok := i.(*sdk.AuthSession)
	if !ok {
		log.Debug("api.getAuthSession> AuthSession type in context is invalid")
		return nil
	}
	return u
}

func (a *API) mustDB() *gorp.DbMap {
	db := a.DBConnectionFactory.GetDBMap(gorpmapping.Mapper)()
	if db == nil {
		panic(fmt.Errorf("Database unavailable"))
	}
	return db
}

func (a *API) mustDBWithCtx(ctx context.Context) *gorp.DbMap {
	db := a.DBConnectionFactory.GetDBMap(gorpmapping.Mapper)()
	db = db.WithContext(ctx).(*gorp.DbMap)
	if db == nil {
		panic(fmt.Errorf("Database unavailable"))
	}

	return db
}
