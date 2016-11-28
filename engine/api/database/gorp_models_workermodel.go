package database

import (
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

//PostInsert is a DB Hook on WorkerModel
func (m *WorkerModel) PostInsert(s gorp.SqlExecutor) error {
	btes, err := json.Marshal(m.CreatedBy)
	if err != nil {
		return err
	}

	query := "update worker_model set created_by = $2 where id = $1"
	if _, err := s.Exec(query, m.ID, btes); err != nil {
		return err
	}

	for _, a := range m.Capabilities {
		query := `insert into worker_capability (worker_model_id, type, name, argument) values ($1, $2, $3, $4)`
		if _, err := s.Exec(query, m.ID, a.Type, a.Name, a.Value); err != nil {
			return err
		}
	}

	return nil
}

//PostUpdate is a DB Hook on WorkerModel
func (m *WorkerModel) PostUpdate(s gorp.SqlExecutor) error {
	if err := m.PreDelete(s); err != nil {
		return err
	}
	return m.PostInsert(s)
}

//PreDelete is a DB Hook on WorkerModel
func (m *WorkerModel) PreDelete(s gorp.SqlExecutor) error {
	queryDelete := "delete from worker_capability where worker_model_id = $1"
	if _, err := s.Exec(queryDelete, m.ID); err != nil {
		return err
	}

	return nil
}

//PostSelect load capabilitites and createdBy user
func (m *WorkerModel) PostSelect(s gorp.SqlExecutor) error {
	//Load capabilities
	var capabilities = []struct {
		Name  string `db:"name"`
		Type  string `db:"type"`
		Value string `db:"argument"`
	}{}

	query := "select * from worker_capability where worker_model_id = $1"
	if _, err := s.Select(&capabilities, query, m.ID); err != nil {
		return err
	}

	for _, c := range capabilities {
		m.Capabilities = append(m.Capabilities, sdk.Requirement{
			Name:  c.Name,
			Type:  c.Type,
			Value: c.Value,
		})
	}

	//Load created_by
	m.CreatedBy = sdk.User{}
	str, errSelect := s.SelectNullStr("select created_by from worker_model where id = $1", &m.ID)
	if errSelect != nil {
		return errSelect
	}
	if !str.Valid || str.String == "" {
		return nil
	}

	if err := json.Unmarshal([]byte(str.String), &m.CreatedBy); err != nil {
		return err
	}

	return nil
}
