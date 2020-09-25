package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// ExternalService represents an external service
type ExternalService struct {
	sdk.Service `json:"-"`
	HealthURL   string `json:"health_url"`
	HealthPort  string `json:"health_port"`
	HealthPath  string `json:"health_path"`
	Port        string `json:"port"`
	URL         string `json:"url"`
	Path        string `json:"path"`
}

// ServiceConfig return sthe serviceConfig for the current ExternalService
func (e ExternalService) ServiceConfig() sdk.ServiceConfig {
	b, _ := json.Marshal(e)
	var cfg sdk.ServiceConfig
	json.Unmarshal(b, &cfg) // nolint
	return cfg
}

// Pings browses all external services and ping them
func Pings(ctx context.Context, dbFunc func() *gorp.DbMap, ss []ExternalService) {
	tickPing := time.NewTicker(1 * time.Minute)
	db := dbFunc()
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "services.Ping> Exiting scheduler.Cleaner: %v", ctx.Err())
				return
			}
		case <-tickPing.C:
			for _, s := range ss {
				tx, err := db.Begin()
				if err != nil {
					log.Warning(ctx, "services.Ping> Unable to start transaction")
					continue
				}
				if err := ping(ctx, tx, s); err != nil {
					log.Error(ctx, "unable to ping service %s: %v", s.Name, err)
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

func ping(ctx context.Context, db gorpmapper.SqlExecutorWithTx, s ExternalService) error {
	// Select for update
	srv, err := LoadByName(context.Background(), db, s.Name)
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

	srv.LastHeartbeat = time.Now()
	srv.MonitoringStatus = mon
	if err := Update(ctx, db, srv); err != nil {
		log.Warning(ctx, "services.ping> unable to update external service: %v", err)
		return err
	}
	if err := UpsertStatus(db, *srv, ""); err != nil {
		log.Warning(ctx, "services.ping> unable to update monitoring status: %v", err)
		return err
	}
	return nil
}

// InitExternal initializes external services
func InitExternal(ctx context.Context, db *gorp.DbMap, ss []ExternalService) error {
	for _, s := range ss {
		if err := initExternal(ctx, db, s); err != nil {
			return err
		}
	}
	return nil
}

func initExternal(ctx context.Context, db *gorp.DbMap, s ExternalService) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "unable to start transaction")
	}
	defer tx.Rollback() // nolint

	log.Debug("services.InitExternal> Initializing service %s", s.Name)

	old, err := LoadByName(ctx, tx, s.Name)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return sdk.WrapError(err, "unable to find service %s", s.Name)
	}
	exists := old != nil

	if exists && old.ConsumerID != nil {
		return sdk.WithStack(fmt.Errorf("can't save an external service as one no external service already exists for given name %s", s.Name))
	}

	if exists {
		s.Service.ID = old.ID
		s.Service.LastHeartbeat = old.LastHeartbeat
		s.Service.MonitoringStatus = old.MonitoringStatus
		s.Service.Config = s.ServiceConfig()
		if err := Update(ctx, tx, &s.Service); err != nil {
			return sdk.WrapError(err, "unable to update external service")
		}
	} else {
		s.Service.LastHeartbeat = time.Now()
		s.Service.Config = s.ServiceConfig()
		if err := Insert(ctx, tx, &s.Service); err != nil {
			return sdk.WrapError(err, "unable to insert external service")
		}
	}

	if err := UpsertStatus(tx, s.Service, ""); err != nil {
		return sdk.WrapError(err, "unable to insert or update monitoring status for external service")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	return nil
}
