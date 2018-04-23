package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

// Repository is the data persistence layer
type Repository struct {
	querier   gorp.SqlExecutor
	db        func(context.Context) *gorp.DbMap
	store     cache.Store
	currentTX *gorp.Transaction
}

// NewRepository returns a fresh repository
func NewRepository(dbFunc func(context.Context) *gorp.DbMap, store cache.Store) *Repository {
	return &Repository{db: dbFunc, store: store}
}

// Querier returns a fresh repository
func Querier(db gorp.SqlExecutor, store cache.Store) *Repository {
	return &Repository{querier: db, store: store}
}

// Tx return the current gorp.SqlExecutor
func (r *Repository) Tx() gorp.SqlExecutor {
	if r.currentTX != nil {
		return r.currentTX
	}
	if r.db != nil {
		return r.db(nil)
	}
	return r.querier
}

// Begin a transaction
func (r *Repository) Begin() error {
	if r.currentTX != nil || r.db == nil {
		return errors.New("Unable to start a new transaction on this repository")
	}
	var err error
	r.currentTX, err = r.db(nil).Begin()
	return err
}

// Commit a transaction
func (r *Repository) Commit() error {
	if r.currentTX == nil || r.db == nil {
		return errors.New("No current transaction")
	}
	err := r.currentTX.Commit()
	r.currentTX = nil
	return err
}

// Rollback the transaction
func (r *Repository) Rollback() error {
	if r.currentTX == nil || r.db == nil {
		return errors.New("No current transaction")
	}
	err := r.currentTX.Rollback()
	r.currentTX = nil
	return err
}

// FindByName a service by its name
func (r *Repository) FindByName(name string) (*sdk.Service, error) {
	query := "SELECT name, type, http_url, last_heartbeat, hash FROM services WHERE name = $1"
	return r.findOne(query, name)
}

// FindByHash a service by its hash
func (r *Repository) FindByHash(hash string) (*sdk.Service, error) {
	query := "SELECT name, type, http_url, last_heartbeat, hash FROM services WHERE hash = $1"
	return r.findOne(query, hash)
}

// FindByType services by type
func (r *Repository) FindByType(t string) ([]sdk.Service, error) {
	query := `
	SELECT name, type, http_url, last_heartbeat, hash 
	FROM services 
	WHERE type = $1`
	services, err := r.findAll(query, t)
	if err != nil {
		if err == sdk.ErrNotFound {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "FindByType> Unable to find dead services")
	}
	return services, nil
}

// All returns all registered services
func (r *Repository) All() ([]sdk.Service, error) {
	query := `
	SELECT name, type, http_url, last_heartbeat, hash
	FROM services`
	services, err := r.findAll(query)
	if err != nil {
		if err == sdk.ErrNotFound {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "All> Unable to find dead services")
	}
	return services, nil
}

func (r *Repository) findOne(query string, args ...interface{}) (*sdk.Service, error) {
	sdb := service{}
	if err := r.Tx().SelectOne(&sdb, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNotFound
		}
		return nil, sdk.WrapError(err, "findOne> service not found")
	}
	if err := r.PostGet(&sdb); err != nil {
		return nil, sdk.WrapError(err, "findOne> postGet")
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
		if err := r.PostGet(&sdbs[i]); err != nil {
			return nil, sdk.WrapError(err, "findAll> postGet")
		}
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
	if err := r.PostUpdate(&sdb); err != nil {
		return sdk.WrapError(err, "Insert> error on PostUpdate")
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
	if err := r.PostUpdate(&sdb); err != nil {
		return sdk.WrapError(err, "Update> error on PostUpdate")
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

// PostGet is a dbHook on Select to get json column
func (r *Repository) PostGet(s *service) error {
	query := "SELECT monitoring_status FROM services WHERE name = $1"
	var content []byte
	if err := r.Tx().QueryRow(query, s.Name).Scan(&content); err != nil {
		return sdk.WrapError(err, "PostGet> error on queryRow")
	}

	if len(content) > 0 {
		m := sdk.MonitoringStatus{}
		if err := json.Unmarshal(content, &m); err != nil {
			return sdk.WrapError(err, "PostGet> error on unmarshal job")
		}
		for i := range m.Lines {
			m.Lines[i].Component = fmt.Sprintf("%s/%s", s.Name, m.Lines[i].Component)
			m.Lines[i].Type = s.Type
		}
		s.MonitoringStatus = m
	}

	return nil
}

// PostUpdate is a DB Hook on PostUpdate to store monitoring_status JSON in DB
func (r *Repository) PostUpdate(s *service) error {
	content, err := json.Marshal(s.MonitoringStatus)
	if err != nil {
		return err
	}

	query := "update services set monitoring_status = $1 where name = $2"
	if _, err := r.Tx().Exec(query, content, s.Name); err != nil {
		return sdk.WrapError(err, "PostUpdate> err on update sql")
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
			return nil, nil
		}
		return nil, sdk.WrapError(err, "FindDeadServices> Unable to find dead services")
	}
	return services, nil
}
