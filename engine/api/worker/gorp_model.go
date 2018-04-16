package worker

import (
	"database/sql"
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
)

// WorkerModel is a gorp wrapper around sdk.Model
type WorkerModel sdk.Model

// WorkerModelPattern is a gorp wrapper around sdk.ModelPattern
type WorkerModelPattern sdk.ModelPattern

//PostInsert is a DB Hook on WorkerModel
func (m *WorkerModel) PostInsert(s gorp.SqlExecutor) error {
	m.CreatedBy.Groups = nil
	m.CreatedBy.Permissions = sdk.UserPermissions{}
	m.CreatedBy.Auth = sdk.Auth{}
	btes, err := json.Marshal(m.CreatedBy)
	if err != nil {
		return err
	}

	var modelBtes []byte
	switch m.Type {
	case sdk.Docker:
		var err error
		modelBtes, err = json.Marshal(m.ModelDocker)
		if err != nil {
			return err
		}
	default:
		var err error
		modelBtes, err = json.Marshal(m.ModelVirtualMachine)
		if err != nil {
			return err
		}
	}

	query := "update worker_model set created_by = $2, model = $3 where id = $1"
	if _, err := s.Exec(query, m.ID, btes, modelBtes); err != nil {
		return err
	}

	for _, a := range m.RegisteredCapabilities {
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

	m.RegisteredCapabilities = make(sdk.RequirementList, len(capabilities))
	for i, c := range capabilities {
		m.RegisteredCapabilities[i] = sdk.Requirement{
			Name:  c.Name,
			Type:  c.Type,
			Value: c.Value,
		}
	}

	//Load created_by
	m.CreatedBy = sdk.User{}
	var createdBy, model, registeredOS, registeredArch sql.NullString
	err := s.QueryRow("select created_by, model, registered_os, registered_arch from worker_model where id = $1", m.ID).Scan(&createdBy, &model, &registeredOS, &registeredArch)
	if err != nil {
		return err
	}

	if registeredOS.Valid {
		m.RegisteredOS = registeredOS.String
	}

	if registeredArch.Valid {
		m.RegisteredArch = registeredArch.String
	}

	switch m.Type {
	case sdk.Docker:
		if err := gorpmapping.JSONNullString(model, &m.ModelDocker); err != nil {
			return sdk.WrapError(err, "PostSelect> cannot unmarshall for docker model")
		}
	default:
		if err := gorpmapping.JSONNullString(model, &m.ModelVirtualMachine); err != nil {
			return sdk.WrapError(err, "PostSelect> cannot unmarshall for vm model")
		}
	}

	if err := gorpmapping.JSONNullString(createdBy, &m.CreatedBy); err != nil {
		return err
	}

	m.CreatedBy.Groups = nil
	m.CreatedBy.Permissions = sdk.UserPermissions{}
	m.CreatedBy.Auth = sdk.Auth{}

	if m.GroupID == group.SharedInfraGroup.ID {
		m.IsOfficial = true
	}

	return nil
}

//PostSelect load capabilitites and createdBy user
func (wmp *WorkerModelPattern) PostSelect(s gorp.SqlExecutor) error {
	modelStr, err := s.SelectNullStr("SELECT model FROM worker_model_pattern WHERE id = $1", wmp.ID)
	if err != nil {
		return sdk.WrapError(err, "PostSelect> Cannot load model for pattern %d", wmp.ID)
	}

	if err := gorpmapping.JSONNullString(modelStr, &wmp.Model); err != nil {
		return sdk.WrapError(err, "PostSelect> Cannot unmarshal json from model pattern")
	}

	return nil
}

//PostInsert is a DB Hook on WorkerModelPattern
func (wmp *WorkerModelPattern) PostInsert(s gorp.SqlExecutor) error {
	modelBtes, err := json.Marshal(wmp.Model)
	if err != nil {
		return sdk.WrapError(err, "PostInsert> Cannot marshal model")
	}

	query := "update worker_model_pattern set model = $1 where id = $2"
	if _, err := s.Exec(query, modelBtes, wmp.ID); err != nil {
		return sdk.WrapError(err, "PostInsert> Cannot update model")
	}

	return nil
}

//PostUpdate is a DB Hook on WorkerModelPattern
func (wmp *WorkerModelPattern) PostUpdate(s gorp.SqlExecutor) error {
	return wmp.PostInsert(s)
}

func init() {
	gorpmapping.Register(gorpmapping.New(WorkerModel{}, "worker_model", true, "id"))
	gorpmapping.Register(gorpmapping.New(WorkerModelPattern{}, "worker_model_pattern", true, "id"))
}
