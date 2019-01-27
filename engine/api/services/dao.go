package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
		s := &sdbs[i]
		if err := s.PostGet(db); err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil, sdk.WrapError(err, "postGet on srv id:%d name:%s type:%s lastHeartbeat:%v", s.ID, s.Name, s.Type, s.LastHeartbeat)
		}
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

// PostGet is a dbHook on Select to get json column
func (s *service) PostGet(db gorp.SqlExecutor) error {
	query := "SELECT monitoring_status, config FROM services WHERE name = $1"
	var monitoringStatus, config []byte
	if err := db.QueryRow(query, s.Name).Scan(&monitoringStatus, &config); err != nil {
		return sdk.WrapError(err, "error on queryRow where name:%s id:%d type:%s", s.Name, s.ID, s.Type)
	}

	if len(monitoringStatus) > 0 {
		m := sdk.MonitoringStatus{}
		if err := json.Unmarshal(monitoringStatus, &m); err != nil {
			return sdk.WrapError(err, "error on unmarshal monitoringStatus service")
		}
		for i := range m.Lines {
			m.Lines[i].Component = fmt.Sprintf("%s/%s", s.Name, m.Lines[i].Component)
			m.Lines[i].Type = s.Type
		}
		s.MonitoringStatus = m
	}
	if len(config) > 0 {
		if err := json.Unmarshal(config, &s.Config); err != nil {
			return sdk.WrapError(err, "error on unmarshal config service")
		}
	}
	return nil
}

// PostInsert is a DB Hook
func (s *service) PostInsert(db gorp.SqlExecutor) error {
	return s.PostUpdate(db)
}

// PostUpdate is a DB Hook on PostUpdate to store monitoring_status JSON in DB
func (s *service) PostUpdate(db gorp.SqlExecutor) error {
	content, err := json.Marshal(s.MonitoringStatus)
	if err != nil {
		return err
	}
	config, errc := json.Marshal(s.Config)
	if errc != nil {
		return errc
	}

	query := "update services set monitoring_status = $1, config = $2 where name = $3"
	if _, err := db.Exec(query, content, config, s.Name); err != nil {
		return sdk.WrapError(err, "err on update sql service")
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
