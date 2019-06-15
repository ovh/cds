package services

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
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
			log.Error("service.getAll> service %d data corrupted", ss[i].ID)
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
		return nil, sdk.WithStack(sdk.ErrNotFound) // TODO return no error
	}

	isValid, err := gorpmapping.CheckSignature(s, s.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error("service.get> service %d data corrupted", s.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound) // TODO return no error
	}

	// TODO why is this code needed ?
	if s.Name == "" {
		return nil, sdk.WithStack(sdk.ErrNotFound) // TODO return no error
	}

	return &s.Service, nil
}

// GetAll returns all services in database.
func GetAll(ctx context.Context, db gorp.SqlExecutor) ([]sdk.Service, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM services`)
	return getAll(ctx, db, query)
}

// GetAllByType returns all services with given type.
func GetAllByType(ctx context.Context, db gorp.SqlExecutor, stype string) ([]sdk.Service, error) {
	if ss, ok := internalCache.getFromCache(stype); ok {
		return ss, nil
	}
	query := gorpmapping.NewQuery(`SELECT * FROM services WHERE type = $1`).Args(stype)
	return getAll(ctx, db, query)
}

// GetByConsumerID returns a service by its consumer id.
func GetByConsumerID(ctx context.Context, db gorp.SqlExecutor, consumerID string) (*sdk.Service, error) {
	query := gorpmapping.NewQuery("SELECT * FROM services WHERE auth_consumer_id = $1").Args(consumerID)
	return get(ctx, db, query)
}

// GetByNameAndType returns a service by its name and type.
func GetByNameAndType(ctx context.Context, db gorp.SqlExecutor, name, stype string) (*sdk.Service, error) {
	query := gorpmapping.NewQuery("SELECT * FROM services WHERE name = $1 and type = $2").Args(name, stype)
	return get(ctx, db, query)
}

// GetByName returns a service by its name.
func GetByName(ctx context.Context, db gorp.SqlExecutor, name string) (*sdk.Service, error) {
	query := gorpmapping.NewQuery("SELECT * FROM services WHERE name = $1").Args(name)
	return get(ctx, db, query)
}

// GetByID returns a service by its id.
func GetByID(ctx context.Context, db gorp.SqlExecutor, id int64) (*sdk.Service, error) {
	query := gorpmapping.NewQuery("SELECT * FROM services WHERE id = $1").Args(id)
	return get(ctx, db, query)
}

// FindDeadServices returns services which haven't heart since th duration
func FindDeadServices(ctx context.Context, db gorp.SqlExecutor, t time.Duration) ([]sdk.Service, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM services WHERE last_heartbeat < $1`).Args(time.Now().Add(-1 * t))
	return getAll(ctx, db, query)
}

// Insert a service in database.
func Insert(db gorp.SqlExecutor, s *sdk.Service) error {
	sdb := service{Service: *s}
	if err := gorpmapping.InsertAndSign(db, &sdb); err != nil {
		return err
	}
	*s = sdb.Service
	return nil
}

// Update a service in database.
func Update(db gorp.SqlExecutor, s *sdk.Service) error {
	sdb := service{Service: *s}
	if err := gorpmapping.UpdatetAndSign(db, &sdb); err != nil {
		return err
	}
	*s = sdb.Service
	return nil
}

// Delete a service.
func Delete(db gorp.SqlExecutor, s *sdk.Service) error {
	sdb := service{Service: *s}
	if _, err := db.Delete(&sdb); err != nil {
		return sdk.WrapError(err, "unable to delete service %s", s.Name)
	}
	return nil
}
