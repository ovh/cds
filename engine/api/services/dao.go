package services

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getAll(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]sdk.Service, error) {
	ss := []service{}

	if err := gorpmapping.GetAll(ctx, db, q, &ss); err != nil {
		return nil, sdk.WrapError(err, "cannot get services")
	}

	// Check signature of data, if invalid do not return it
	verifiedServices := make([]sdk.Service, 0, len(ss))
	for i := range ss {
		isValid, err := gorpmapping.CheckSignature(ss[i], ss[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "service.getAll> service %d data corrupted", ss[i].ID)
			continue
		}
		verifiedServices = append(verifiedServices, ss[i].Service)
	}

	return verifiedServices, nil
}

func get(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) (*sdk.Service, error) {
	var s service

	found, err := gorpmapping.Get(ctx, db, q, &s)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get service")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := gorpmapping.CheckSignature(s, s.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "service.get> service %d data corrupted", s.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	return &s.Service, nil
}

// LoadAll returns all services in database.
func LoadAll(ctx context.Context, db gorp.SqlExecutor) ([]sdk.Service, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM service`)
	return getAll(ctx, db, query)
}

// LoadAllWithStatus returns all services in database with status
func LoadAllWithStatus(ctx context.Context, db gorp.SqlExecutor) ([]sdk.Service, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM service`)
	srvs, err := getAll(ctx, db, query)
	if err != nil {
		return nil, err
	}

	ss, err := loadAllServiceStatus(ctx, db)
	if err != nil {
		return nil, err
	}
	for i := range srvs {
		srv := &srvs[i]
		srv.MonitoringStatus = sdk.MonitoringStatus{Now: time.Now()}
		for _, status := range ss {
			if srv.ID == status.ServiceID {
				srv.MonitoringStatus.Lines = append(srv.MonitoringStatus.Lines, status.MonitoringStatus.Lines...)
			}
		}
		srvs[i] = *srv
	}

	return srvs, nil
}

// LoadAllByType returns all services with given type.
func LoadAllByType(ctx context.Context, db gorp.SqlExecutor, typeService string) ([]sdk.Service, error) {
	if ss, ok := internalCache.getFromCache(typeService); ok {
		return ss, nil
	}
	query := gorpmapping.NewQuery(`SELECT * FROM service WHERE type = $1`).Args(typeService)
	return getAll(ctx, db, query)
}

// LoadAllByTypeAndUserID returns all services that users can see with given type.
func LoadAllByTypeAndUserID(ctx context.Context, db gorp.SqlExecutor, typeService string, userID string) ([]sdk.Service, error) {
	query := gorpmapping.NewQuery(`
		SELECT service.*
		FROM service
		JOIN auth_consumer on auth_consumer.id = service.auth_consumer_id
		WHERE service.type = $1 AND auth_consumer.user_id = $2`).Args(typeService, userID)
	return getAll(ctx, db, query)
}

// LoadByConsumerID returns a service by its consumer id.
func LoadByConsumerID(ctx context.Context, db gorp.SqlExecutor, consumerID string) (*sdk.Service, error) {
	query := gorpmapping.NewQuery("SELECT * FROM service WHERE auth_consumer_id = $1").Args(consumerID)
	return get(ctx, db, query)
}

// LoadByNameAndType returns a service by its name and type.
func LoadByNameAndType(ctx context.Context, db gorp.SqlExecutor, name, stype string) (*sdk.Service, error) {
	query := gorpmapping.NewQuery("SELECT * FROM service WHERE name = $1 and type = $2").Args(name, stype)
	return get(ctx, db, query)
}

// LoadByName returns a service by its name.
func LoadByName(ctx context.Context, db gorp.SqlExecutor, name string) (*sdk.Service, error) {
	query := gorpmapping.NewQuery("SELECT * FROM service WHERE name = $1").Args(name)
	return get(ctx, db, query)
}

// LoadByID returns a service by its id.
func LoadByID(ctx context.Context, db gorp.SqlExecutor, id int64) (*sdk.Service, error) {
	query := gorpmapping.NewQuery("SELECT * FROM service WHERE id = $1").Args(id)
	return get(ctx, db, query)
}

// FindDeadServices returns services which haven't heart since th duration
func FindDeadServices(ctx context.Context, db gorp.SqlExecutor, t time.Duration) ([]sdk.Service, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM service WHERE last_heartbeat < $1`).Args(time.Now().Add(-1 * t))
	return getAll(ctx, db, query)
}

// Insert a service in database.
func Insert(ctx context.Context, db gorpmapper.SqlExecutorWithTx, s *sdk.Service, authSessionID string) error {
	sdb := service{Service: *s}
	if err := gorpmapping.InsertAndSign(ctx, db, &sdb); err != nil {
		return err
	}
	*s = sdb.Service
	return upsertStatus(db, s, authSessionID)
}

// Update a service in database.
func Update(ctx context.Context, db gorpmapper.SqlExecutorWithTx, s *sdk.Service, authSessionID string) error {
	sdb := service{Service: *s}
	if err := gorpmapping.UpdateAndSign(ctx, db, &sdb); err != nil {
		return err
	}
	*s = sdb.Service
	return upsertStatus(db, s, authSessionID)
}

// loadAllServiceStatus returns all services status
func loadAllServiceStatus(ctx context.Context, db gorp.SqlExecutor) ([]sdk.ServiceStatus, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM service_status`)
	ss := []sdk.ServiceStatus{}
	if err := gorpmapping.GetAll(ctx, db, query, &ss); err != nil {
		return nil, sdk.WrapError(err, "cannot get services")
	}
	return ss, nil
}

func upsertStatus(db gorpmapper.SqlExecutorWithTx, s *sdk.Service, authSessionID string) error {
	var sessionID *string
	if authSessionID != "" {
		sessionID = &authSessionID
	}
	query := `INSERT INTO service_status(monitoring_status, service_id, auth_session_id) VALUES($1,$2, $3)
	ON CONFLICT (service_id, auth_session_id) DO UPDATE SET monitoring_status = $1, service_id = $2, auth_session_id = $3`
	if _, err := db.Exec(query, s.MonitoringStatus, s.ID, sessionID); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

// Delete a service.
func Delete(db gorp.SqlExecutor, s *sdk.Service) error {
	if s.Type == sdk.TypeHatchery {
		if err := worker.ReleaseAllFromHatchery(db, s.ID); err != nil {
			return err
		}
	}
	sdb := service{Service: *s}
	log.Debug("services.Delete> deleting service %s(%d) from database", s.Name, s.ID)
	if _, err := db.Delete(&sdb); err != nil {
		return sdk.WrapError(err, "unable to delete service %s", s.Name)
	}
	return nil
}
