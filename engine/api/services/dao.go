package services

import (
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/sdk"
)

// Repository is the data persistence layer
type Repository struct {
	sync      *sync.Mutex
	db        *database.DBConnectionFactory
	store     cache.Store
	currentTX *gorp.Transaction
}

// NewRepository returns a fresh repository
func NewRepository(db *database.DBConnectionFactory, store cache.Store) *Repository {
	return &Repository{db: db, store: store, sync: new(sync.Mutex)}
}

func (r *Repository) Tx() gorp.SqlExecutor {
	r.sync.Lock()
	defer r.sync.Unlock()
	if r.currentTX != nil {
		return r.currentTX
	}
	return r.db.GetDBMap()
}

func (r *Repository) Begin() error {
	r.sync.Lock()
	defer r.sync.Unlock()
	if r.currentTX != nil {
		return errors.New("Unable to start a new transaction on this repository")
	}
	var err error
	r.currentTX, err = r.db.GetDBMap().Begin()
	return err
}

func (r *Repository) Commit() error {
	r.sync.Lock()
	defer r.sync.Unlock()
	if r.currentTX == nil {
		return errors.New("No current transaction")
	}
	err := r.currentTX.Commit()
	r.currentTX = nil
	return err
}

func (r *Repository) Rollback() error {
	r.sync.Lock()
	defer r.sync.Unlock()
	if r.currentTX == nil {
		return errors.New("No current transaction")
	}
	err := r.currentTX.Rollback()
	r.currentTX = nil
	return err
}

// Find a service by its name
func (r *Repository) Find(name string) (*sdk.Service, error) {
	query := "SELECT name, type, http_url, last_heartbeat, hash FROM services WHERE name = $1"
	sdb := service{}
	if err := r.Tx().SelectOne(&sdb, query, name); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNotFound
		}
		return nil, sdk.WrapError(err, "Find> service %s not found", name)
	}
	s := sdk.Service(sdb)
	if s.Name == "" {
		return nil, sdk.ErrNotFound
	}
	return &s, nil
}

func (r *Repository) findAll(query string, args ...interface{}) ([]sdk.Service, error) {
	sdbs := []service{}
	if _, err := r.Tx().Select(&sdbs, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNotFound
		}
		return nil, sdk.WrapError(err, "findAll> no service found")
	}
	ss := make([]sdk.Service, len(sdbs))
	for i := 0; i < len(sdbs); i++ {
		ss[i] = sdk.Service(sdbs[i])
	}
	return ss, nil
}

// Insert a service
func (r *Repository) Insert(s *sdk.Service) error {
	sdb := service(*s)
	if err := r.Tx().Insert(&sdb); err != nil {
		return sdk.WrapError(err, "Insert> unable to insert service %s", s.Name)
	}
	*s = sdk.Service(sdb)
	return nil
}

// Update a service
func (r *Repository) Update(s *sdk.Service) error {
	sdb := service(*s)
	if n, err := r.Tx().Update(&sdb); err != nil {
		return sdk.WrapError(err, "Update> unable to update service %s", s.Name)
	} else if n == 0 {
		return sdk.WrapError(sdk.ErrNotFound, "Update> unable to update service %s", s.Name)
	}
	*s = sdk.Service(sdb)
	return nil
}

// Delete a service
func (r *Repository) Delete(s *sdk.Service) error {
	sdb := service(*s)
	if _, err := r.Tx().Delete(&sdb); err != nil {
		return sdk.WrapError(err, "Delete> unable to delete service %s", s.Name)
	}
	return nil
}

// FindDeadServices returns services which haven't heart since th duration
func (r *Repository) FindDeadServices(t time.Duration) ([]sdk.Service, error) {
	query := `
	SELECT name, type, http_url, last_heartbeat, hash 
	FROM services 
	WHERE last_heartbeat < $1`
	services, err := r.findAll(query, time.Now().Add(-1*t))
	if err != nil {
		if err == sdk.ErrNotFound {
			return services, nil
		}
		return nil, sdk.WrapError(err, "FindDeadServices> Unable to find dead services")
	}
	return services, nil
}
