package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getServiceHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		typeService := vars["type"]

		var servicesConf []sdk.ServiceConfiguration
		for _, s := range api.Config.Services {
			if s.Type == typeService {
				servicesConf = append(servicesConf, s)
			}
		}
		if len(servicesConf) != 0 {
			return service.WriteJSON(w, servicesConf, http.StatusOK)
		}

		// Try to load from DB
		var srvs []sdk.Service
		var err error
		if isAdmin(ctx) || isMaintainer(ctx) {
			srvs, err = services.LoadAllByType(ctx, api.mustDB(), typeService)
		} else {
			c := getAPIConsumer(ctx)
			srvs, err = services.LoadAllByTypeAndUserID(ctx, api.mustDB(), typeService, c.AuthentifiedUserID)
		}
		if err != nil {
			return err
		}
		for _, s := range srvs {
			servicesConf = append(servicesConf, sdk.ServiceConfiguration{
				URL:       s.HTTPURL,
				Name:      s.Name,
				ID:        s.ID,
				PublicKey: base64.StdEncoding.EncodeToString(s.PublicKey),
				Type:      s.Type,
			})
		}
		if len(servicesConf) == 0 {
			return sdk.WrapError(sdk.ErrNotFound, "service %s not found", typeService)
		}
		return service.WriteJSON(w, servicesConf, http.StatusOK)
	}
}

func (api *API) postServiceRegisterHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		consumer := getAPIConsumer(ctx)

		var data sdk.Service
		if err := service.UnmarshalBody(r, &data); err != nil {
			return sdk.WithStack(err)
		}
		data.LastHeartbeat = time.Now()

		if data.Name == "" {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing service name")
		}

		// Service that are not hatcheries should be started as an admin
		if data.Type != sdk.TypeHatchery && !isAdmin(ctx) {
			return sdk.WrapError(sdk.ErrForbidden, "cannot register service of type %s for consumer %s", data.Type, consumer.ID)
		}

		// Insert or update the service
		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		// Try to find the service, and keep; else generate a new one
		srv, err := services.LoadByConsumerID(ctx, tx, consumer.ID)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		exists := srv != nil

		if exists && srv.Type != data.Type {
			return sdk.WrapError(sdk.ErrForbidden, "cannot register service %s of type %s for consumer %s while existing service type is different", data.Name, data.Type, consumer.ID)
		}

		// Update or create the service

		var sessionID string
		if a := getAuthSession(ctx); a != nil {
			sessionID = a.ID
		}
		if exists {
			srv.Update(data)
			if err := services.Update(ctx, tx, srv); err != nil {
				return err
			}
			log.Debug("postServiceRegisterHandler> update existing service %s(%d) registered for consumer %s", srv.Name, srv.ID, *srv.ConsumerID)
		} else {
			srv = &data
			srv.ConsumerID = &consumer.ID

			if err := services.Insert(ctx, tx, srv); err != nil {
				return sdk.WithStack(err)
			}
			log.Debug("postServiceRegisterHandler> insert new service %s(%d) registered for consumer %s", srv.Name, srv.ID, *srv.ConsumerID)
		}

		if err := services.UpsertStatus(tx, *srv, sessionID); err != nil {
			return sdk.WithStack(err)
		}

		if len(srv.PublicKey) > 0 {
			log.Debug("postServiceRegisterHandler> service %s registered with public key: %s", srv.Name, string(srv.PublicKey))
		}

		// For hatchery service we need to check if there are workers that are not attached to an existing hatchery
		// If some worker's parent consumer match current hatchery consumer we will attach this worker to the new hatchery.
		if srv.Type == sdk.TypeHatchery {
			if err := worker.ReAttachAllToHatchery(ctx, tx, *srv); err != nil {
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		srv.Uptodate = data.Version == sdk.VERSION

		return service.WriteJSON(w, srv, http.StatusOK)
	}
}

func (api *API) postServiceHearbeatHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if ok := isService(ctx); !ok {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		s, err := services.LoadByID(ctx, api.mustDB(), getAPIConsumer(ctx).Service.ID)
		if err != nil {
			return err
		}

		var mon sdk.MonitoringStatus
		if err := service.UnmarshalBody(r, &mon); err != nil {
			return err
		}

		// Update status to warn if service version != api version
		for i := range mon.Lines {
			if mon.Lines[i].Component == "Version" {
				if sdk.VERSION != mon.Lines[i].Value {
					mon.Lines[i].Status = sdk.MonitoringStatusWarn
				} else {
					mon.Lines[i].Status = sdk.MonitoringStatusOK
				}
				break
			}
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		s.LastHeartbeat = time.Now()
		s.MonitoringStatus = mon

		var sessionID string
		if a := getAuthSession(ctx); a != nil {
			sessionID = a.ID
		}
		if err := services.Update(ctx, tx, s); err != nil {
			return err
		}

		if err := services.UpsertStatus(tx, *s, sessionID); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return nil
	}
}

func (api *API) serviceAPIHeartbeat(ctx context.Context) {
	tick := time.NewTicker(30 * time.Second).C

	// first call
	api.serviceAPIHeartbeatUpdate(ctx, api.mustDB())

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting serviceAPIHeartbeat: %v", ctx.Err())
				return
			}
		case <-tick:
			api.serviceAPIHeartbeatUpdate(ctx, api.mustDB())
		}
	}
}

func (api *API) serviceAPIHeartbeatUpdate(ctx context.Context, db *gorp.DbMap) {
	tx, err := db.Begin()
	if err != nil {
		log.Error(ctx, "serviceAPIHeartbeat> error on repo.Begin:%v", err)
		return
	}
	defer tx.Rollback() // nolint

	var srvConfig sdk.ServiceConfig
	b, _ := json.Marshal(api.Config)
	json.Unmarshal(b, &srvConfig) // nolint

	srv := &sdk.Service{
		CanonicalService: sdk.CanonicalService{
			Name:   event.GetCDSName(),
			Type:   sdk.TypeAPI,
			Config: srvConfig,
		},
		MonitoringStatus: *api.Status(ctx),
		LastHeartbeat:    time.Now(),
	}

	old, err := services.LoadByName(ctx, tx, srv.Name)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		log.Error(ctx, "serviceAPIHeartbeat> Unable to find service by name: %v", err)
		return
	}
	exists := old != nil

	if exists && old.ConsumerID != nil {
		log.Error(ctx, "serviceAPIHeartbeat> Can't save an api service as one service already exists for given name %s", srv.Name)
		return
	}

	var authSessionID string
	if a := getAuthSession(ctx); a != nil {
		authSessionID = a.ID
	}
	if exists {
		srv.ID = old.ID
		if err := services.Update(ctx, tx, srv); err != nil {
			log.Error(ctx, "serviceAPIHeartbeat> Unable to update service %s: %v", srv.Name, err)
			return
		}
	} else {
		if err := services.Insert(ctx, tx, srv); err != nil {
			log.Error(ctx, "serviceAPIHeartbeat> Unable to insert service %s: %v", srv.Name, err)
			return
		}
	}

	if err := services.UpsertStatus(tx, *srv, authSessionID); err != nil {
		log.Error(ctx, "serviceAPIHeartbeat> Unable to insert or update monitoring status %s: %v", srv.Name, err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Error(ctx, "serviceAPIHeartbeat> error tx commit: %v", err)
		return
	}
}
