package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

var servicesCacheByType map[string][]sdk.Service
var sEvent chan serviceEvent

type serviceEvent struct {
	Action  string
	Service sdk.Service
}

func init() {
	sEvent = make(chan serviceEvent, 10)
	servicesCacheByType = make(map[string][]sdk.Service)

	go managerServiceEvent(sEvent)
}

// FindByName a service by its name
func FindByName(db gorp.SqlExecutor, name string) (*sdk.Service, error) {
	query := "SELECT name, type, http_url, last_heartbeat, hash FROM services WHERE name = $1"
	return findOne(db, query, name)
}

// FindByHash a service by its hash
func FindByHash(db gorp.SqlExecutor, hash string) (*sdk.Service, error) {
	query := "SELECT name, type, http_url, last_heartbeat, hash FROM services WHERE hash = $1"
	return findOne(db, query, hash)
}

// FindByType services by type
func FindByType(db gorp.SqlExecutor, t string) ([]sdk.Service, error) {
	if ss, ok := servicesCacheByType[t]; ok {
		return ss, nil
	}

	query := `
	SELECT name, type, http_url, last_heartbeat, hash 
	FROM services 
	WHERE type = $1`
	services, err := findAll(db, query, t)
	if err != nil {
		if err == sdk.ErrNotFound {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "FindByType> Unable to find dead services")
	}

	return services, nil
}

// All returns all registered services
func All(db gorp.SqlExecutor) ([]sdk.Service, error) {
	query := `
	SELECT name, type, http_url, last_heartbeat, hash
	FROM services`
	services, err := findAll(db, query)
	if err != nil {
		if err == sdk.ErrNotFound {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "All> Unable to find dead services")
	}
	return services, nil
}

func findOne(db gorp.SqlExecutor, query string, args ...interface{}) (*sdk.Service, error) {
	sdb := service{}
	if err := db.SelectOne(&sdb, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNotFound
		}
		return nil, sdk.WrapError(err, "findOne> service not found")
	}
	if err := PostGet(db, &sdb); err != nil {
		return nil, sdk.WrapError(err, "findOne> postGet")
	}
	s := sdk.Service(sdb)
	if s.Name == "" {
		return nil, sdk.ErrNotFound
	}
	return &s, nil
}

func findAll(db gorp.SqlExecutor, query string, args ...interface{}) ([]sdk.Service, error) {
	sdbs := []service{}
	if _, err := db.Select(&sdbs, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNotFound
		}
		return nil, sdk.WrapError(err, "findAll> no service found")
	}
	ss := make([]sdk.Service, len(sdbs))
	for i := 0; i < len(sdbs); i++ {
		if err := PostGet(db, &sdbs[i]); err != nil {
			return nil, sdk.WrapError(err, "findAll> postGet")
		}
		ss[i] = sdk.Service(sdbs[i])
	}
	return ss, nil
}

// Insert a service
func Insert(db gorp.SqlExecutor, s *sdk.Service) error {
	sdb := service(*s)
	if err := db.Insert(&sdb); err != nil {
		return sdk.WrapError(err, "Insert> unable to insert service %s", s.Name)
	}
	if err := PostUpdate(db, &sdb); err != nil {
		return sdk.WrapError(err, "Insert> error on PostUpdate")
	}
	*s = sdk.Service(sdb)

	sEvent <- serviceEvent{
		Service: *s,
		Action:  "update",
	}
	return nil
}

// Update a service
func Update(db gorp.SqlExecutor, s *sdk.Service) error {
	sdb := service(*s)
	if n, err := db.Update(&sdb); err != nil {
		return sdk.WrapError(err, "Update> unable to update service %s", s.Name)
	} else if n == 0 {
		return sdk.WrapError(sdk.ErrNotFound, "Update> unable to update service %s", s.Name)
	}
	if err := PostUpdate(db, &sdb); err != nil {
		return sdk.WrapError(err, "Update> error on PostUpdate")
	}
	*s = sdk.Service(sdb)
	sEvent <- serviceEvent{
		Service: *s,
		Action:  "update",
	}
	return nil
}

// Delete a service
func Delete(db gorp.SqlExecutor, s *sdk.Service) error {
	sdb := service(*s)
	if _, err := db.Delete(&sdb); err != nil {
		return sdk.WrapError(err, "Delete> unable to delete service %s", s.Name)
	}
	sEvent <- serviceEvent{
		Service: *s,
		Action:  "remove",
	}
	return nil
}

// PostGet is a dbHook on Select to get json column
func PostGet(db gorp.SqlExecutor, s *service) error {
	query := "SELECT monitoring_status FROM services WHERE name = $1"
	var content []byte
	if err := db.QueryRow(query, s.Name).Scan(&content); err != nil {
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
func PostUpdate(db gorp.SqlExecutor, s *service) error {
	content, err := json.Marshal(s.MonitoringStatus)
	if err != nil {
		return err
	}

	query := "update services set monitoring_status = $1 where name = $2"
	if _, err := db.Exec(query, content, s.Name); err != nil {
		return sdk.WrapError(err, "PostUpdate> err on update sql")
	}
	return nil
}

// FindDeadServices returns services which haven't heart since th duration
func FindDeadServices(db gorp.SqlExecutor, t time.Duration) ([]sdk.Service, error) {
	query := `
	SELECT name, type, http_url, last_heartbeat, hash 
	FROM services 
	WHERE last_heartbeat < $1`
	services, err := findAll(db, query, time.Now().Add(-1*t))
	if err != nil {
		if err == sdk.ErrNotFound {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "FindDeadServices> Unable to find dead services")
	}
	return services, nil
}

func managerServiceEvent(c <-chan serviceEvent) {
	for se := range c {
		switch se.Action {
		case "update":
			updateCache(se.Service)
		case "remove":
			removeFromCache(se.Service)
		}
	}
}

func updateCache(s sdk.Service) {
	ss, ok := servicesCacheByType[s.Type]
	indexToUpdate := -1
	if !ok {
		ss = make([]sdk.Service, 0, 1)
	} else {
		for i, sub := range ss {
			if sub.Name == s.Name {
				indexToUpdate = i
				break
			}
		}
	}
	if indexToUpdate == -1 {
		ss = append(ss, s)
	} else {
		ss[indexToUpdate] = s
	}
	servicesCacheByType[s.Type] = ss
}

func removeFromCache(s sdk.Service) {
	ss, ok := servicesCacheByType[s.Type]
	if !ok || len(ss) == 0 {
		return
	}
	indexToSplit := 0
	for i, sub := range ss {
		if sub.Name == s.Name {
			indexToSplit = i
			break
		}
	}
	ss = append(ss[:indexToSplit], ss[indexToSplit+1:]...)
	servicesCacheByType[s.Type] = ss
}
