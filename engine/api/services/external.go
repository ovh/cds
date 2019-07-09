package services

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Pings browses all external services and ping them
func Pings(ctx context.Context, dbFunc func() *gorp.DbMap, ss []sdk.ExternalService) {
	tickPing := time.NewTicker(1 * time.Minute)
	db := dbFunc()
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error("services.Ping> Exiting scheduler.Cleaner: %v", ctx.Err())
				return
			}
		case <-tickPing.C:
			for _, s := range ss {
				tx, err := db.Begin()
				if err != nil {
					log.Warning("services.Ping> Unable to start transaction")
					continue
				}
				if err := ping(tx, s); err != nil {
					log.Error("unable to ping service %s: %v", s.Name, err)
					_ = tx.Rollback()
					continue
				}
				if err := tx.Commit(); err != nil {
					_ = tx.Rollback()
				}
			}
		}
	}
}

func ping(db gorp.SqlExecutor, s sdk.ExternalService) error {
	// Select for update
	serv, err := LoadByNameForUpdateAndSkipLocked(context.Background(), db, s.Name)
	if err != nil {
		return sdk.WithStack(err)
	}

	mon := sdk.MonitoringStatus{
		Now: time.Now(),
		Lines: []sdk.MonitoringStatusLine{
			{
				Type:      s.Type,
				Component: s.Name,
			},
		},
	}
	var pingURL string
	if s.HealthPort == "0" || s.HealthPort == "" {
		pingURL = s.HealthURL
	} else {
		pingURL = fmt.Sprintf("%s:%s", s.HealthURL, s.HealthPort)
	}

	u, err := url.ParseRequestURI(pingURL + s.HealthPath)
	if err != nil {
		return sdk.WithStack(err)
	}

	log.Debug("services.ping> Checking service %s (%v)", s.Name, u.String())

	_, _, code, err := doRequestFromURL(context.Background(), db, "GET", u, nil)
	if err != nil || code >= 400 {
		mon.Lines[0].Status = sdk.MonitoringStatusWarn
		mon.Lines[0].Value = "Health: KO"
	} else {
		mon.Lines[0].Status = sdk.MonitoringStatusOK
		mon.Lines[0].Value = "Health: OK"
	}

	serv.LastHeartbeat = time.Now()
	serv.MonitoringStatus = mon
	if err := Update(db, serv); err != nil {
		log.Warning("services.ping> unable to update monitoring status: %v", err)
		return err
	}
	return nil
}

// InitExternal initializes external services
func InitExternal(ctx context.Context, db *gorp.DbMap, ss []sdk.ExternalService) error {
	for _, s := range ss {
		tx, err := db.Begin()
		if err != nil {
			return sdk.WrapError(err, "Unable to start transaction")
		}

		log.Debug("services.InitExternal> Initializing service %s", s.Name)

		oldSrv, errOldSrv := LoadByNameForUpdateAndSkipLocked(ctx, tx, s.Name)
		if errOldSrv != nil && !sdk.ErrorIs(errOldSrv, sdk.ErrNotFound) {
			_ = tx.Rollback()
			return sdk.WithStack(fmt.Errorf("Unable to find service %s", s.Name))
		}

		if oldSrv == nil {
			s.Service.LastHeartbeat = time.Now()
			s.Service.Config = s.ServiceConfig()
			if err := Insert(tx, &s.Service); err != nil {
				_ = tx.Rollback()
				return sdk.WrapError(err, "Unable to insert external service")
			}
		} else {
			s.Service.ID = oldSrv.ID
			s.Service.LastHeartbeat = oldSrv.LastHeartbeat
			s.Service.MonitoringStatus = oldSrv.MonitoringStatus
			s.Service.Config = s.ServiceConfig()
			if err := Update(tx, &s.Service); err != nil {
				_ = tx.Rollback()
				return sdk.WrapError(err, "Unable to update external service")
			}
		}

		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
		}
	}
	return nil
}
