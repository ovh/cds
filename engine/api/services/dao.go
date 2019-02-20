package services

import (
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// FindByNameAndType a service by its name and type
func FindByNameAndType(db gorp.SqlExecutor, name, stype string) (*sdk.Service, error) {
	query := "SELECT * FROM services WHERE name = $1 and type = $2"
	return findOne(db, query, name, stype)
}

// FindByName a service by its name
func FindByName(db gorp.SqlExecutor, name string) (*sdk.Service, error) {
	query := "SELECT * FROM services WHERE name = $1"
	return findOne(db, query, name)
}

// FindByHash a service by its hash
func FindByHash(db gorp.SqlExecutor, hash string) (*sdk.Service, error) {
	query := "SELECT * FROM services WHERE hash = $1"
	return findOne(db, query, hash)
}

// FindByType services by type
func FindByType(db gorp.SqlExecutor, t string) ([]sdk.Service, error) {
	if ss, ok := internalCache.getFromCache(t); ok {
		return ss, nil
	}
	query := `SELECT * FROM services WHERE type = $1`
	services, err := findAll(db, query, t)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Unable to find dead services")
	}

	return services, nil
}

// All returns all registered services
func All(db gorp.SqlExecutor) ([]sdk.Service, error) {
	query := `SELECT * FROM services`
	services, err := findAll(db, query)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Unable to find all services")
	}
	return services, nil
}

func findOne(db gorp.SqlExecutor, query string, args ...interface{}) (*sdk.Service, error) {
	sdb := service{}
	if err := db.SelectOne(&sdb, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrNotFound)
		}
		return nil, sdk.WrapError(err, "service not found")
	}
	s := sdk.Service(sdb)
	if s.Name == "" {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &s, nil
}

func findAll(db gorp.SqlExecutor, query string, args ...interface{}) ([]sdk.Service, error) {
	sdbs := []service{}
	if _, err := db.Select(&sdbs, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrNotFound)
		}
		return nil, sdk.WithStack(err)
	}
	ss := make([]sdk.Service, 0, len(sdbs))
	for i := 0; i < len(sdbs); i++ {
		ss = append(ss, sdk.Service(sdbs[i]))
	}
	return ss, nil
}

// Insert a service
func Insert(db gorp.SqlExecutor, s *sdk.Service) error {
	sdb := service(*s)
	if err := db.Insert(&sdb); err != nil {
		return sdk.WrapError(err, "unable to insert service %s", s.Name)
	}
	*s = sdk.Service(sdb)
	return nil
}

// Update a service
func Update(db gorp.SqlExecutor, s *sdk.Service) error {
	sdb := service(*s)
	if _, err := db.Update(&sdb); err != nil {
		return sdk.WrapError(err, "unable to update service %s", s.Name)
	}
	return nil
}

// Delete a service
func Delete(db gorp.SqlExecutor, s *sdk.Service) error {
	sdb := service(*s)
	if _, err := db.Delete(&sdb); err != nil {
		return sdk.WrapError(err, "unable to delete service %s", s.Name)
	}
	return nil
}

// FindDeadServices returns services which haven't heart since th duration
func FindDeadServices(db gorp.SqlExecutor, t time.Duration) ([]sdk.Service, error) {
	query := `SELECT * FROM services WHERE last_heartbeat < $1`
	services, err := findAll(db, query, time.Now().Add(-1*t))
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Unable to find dead services")
	}
	return services, nil
}
