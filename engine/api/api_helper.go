package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

// group should have members aggregated and authentified user old user struct should be set.
func isGroupAdmin(ctx context.Context, g *sdk.Group) bool {
	c := getAPIConsumer(ctx)
	if c == nil {
		return false
	}
	return group.IsConsumerGroupAdmin(g, c)
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
	return c.Maintainer()
}

func supportMFA(ctx context.Context) bool {
	m := getAuthDriverManifest(ctx)
	if m == nil {
		return false
	}
	return m.SupportMFA
}

func isAdmin(ctx context.Context) bool {
	c := getAPIConsumer(ctx)
	if c == nil {
		return false
	}
	m := getAuthDriverManifest(ctx)
	if m == nil {
		return false
	}
	var dontNeedMFA = !m.SupportMFA
	return c.Admin() && (dontNeedMFA || isMFA(ctx))
}

func isService(ctx context.Context) bool {
	c := getAPIConsumer(ctx)
	if c == nil || c.AuthConsumerUser == nil {
		return false
	}
	return c.AuthConsumerUser.Service != nil
}

func isWorker(ctx context.Context) bool {
	c := getAPIConsumer(ctx)
	if c == nil || c.AuthConsumerUser == nil {
		return false
	}
	return c.AuthConsumerUser.Worker != nil
}

func isHatchery(ctx context.Context) bool {
	c := getAPIConsumer(ctx)
	if c == nil || c.AuthConsumerUser == nil {
		return false
	}
	return c.AuthConsumerUser.Service != nil && c.AuthConsumerUser.Service.Type == sdk.TypeHatchery
}

func isHatcheryShared(ctx context.Context) bool {
	c := getAPIConsumer(ctx)
	if c == nil || c.AuthConsumerUser == nil {
		return false
	}
	return isHatchery(ctx) && c.AuthConsumerUser.GroupIDs.Contains(group.SharedInfraGroup.ID)
}

func isCDN(ctx context.Context) bool {
	c := getAPIConsumer(ctx)
	if c == nil || c.AuthConsumerUser == nil {
		return false
	}
	return c.AuthConsumerUser.Service != nil && c.AuthConsumerUser.Service.Type == sdk.TypeCDN
}

func isHooks(ctx context.Context) bool {
	c := getAPIConsumer(ctx)
	if c == nil || c.AuthConsumerUser == nil {
		return false
	}
	return c.AuthConsumerUser.Service != nil && c.AuthConsumerUser.Service.Type == sdk.TypeHooks
}

func isMFA(ctx context.Context) bool {
	s := getAuthSession(ctx)
	if s == nil {
		return false
	}
	return s.MFA
}

func trackSudo(ctx context.Context, w http.ResponseWriter) {
	if isAdmin(ctx) && !isService(ctx) && !isWorker(ctx) {
		SetTracker(w, cdslog.Sudo, true)
	}
}

func getAPIConsumer(ctx context.Context) *sdk.AuthConsumer {
	i := ctx.Value(contextConsumer)
	if i == nil {
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

func getAuthSession(ctx context.Context) *sdk.AuthSession {
	i := ctx.Value(contextSession)
	if i == nil {
		return nil
	}
	s, ok := i.(*sdk.AuthSession)
	if !ok {
		log.Debug(ctx, "api.getAuthSession> AuthSession type in context is invalid")
		return nil
	}
	return s
}

func getAuthDriverManifest(ctx context.Context) *sdk.AuthDriverManifest {
	i := ctx.Value(contextDriverManifest)
	if i == nil {
		return nil
	}
	m, ok := i.(*sdk.AuthDriverManifest)
	if !ok {
		log.Debug(ctx, "api.getDriverManifest> AuthDriverManifest type in context is invalid")
		return nil
	}
	return m
}

func getAuthClaims(ctx context.Context) *sdk.AuthSessionJWTClaims {
	i := ctx.Value(contextClaims)
	if i == nil {
		return nil
	}
	c, ok := i.(*sdk.AuthSessionJWTClaims)
	if !ok {
		log.Debug(ctx, "api.getAuthClaims> AuthSessionJWTClaims type in context is invalid")
		return nil
	}
	return c
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
