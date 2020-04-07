package workermodel

import (
	"database/sql"
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

func init() {
	gorpmapping.Register(gorpmapping.New(WorkerModel{}, "worker_model", true, "id"))
	gorpmapping.Register(gorpmapping.New(workerModelPattern{}, "worker_model_pattern", true, "id"))
}

// WorkerModel is a gorp wrapper around sdk.Model.
type WorkerModel sdk.Model

//PostInsert is a DB Hook on WorkerModel
func (m *WorkerModel) PostInsert(s gorp.SqlExecutor) error {
	btes, err := json.Marshal(m.Author)
	if err != nil {
		return err
	}

	var modelBtes []byte
	switch m.Type {
	case sdk.Docker:
		m.ModelDocker.Envs = MergeModelEnvsWithDefaultEnvs(m.ModelDocker.Envs)
		var err error
		if m.ModelDocker.Private {
			if m.ModelDocker.Password != "" {
				m.ModelDocker.Password, err = secret.EncryptValue(m.ModelDocker.Password)
				if err != nil {
					return sdk.WrapError(err, "cannot encrypt docker password")
				}
			}
		} else {
			m.ModelDocker.Username = ""
			m.ModelDocker.Password = ""
			m.ModelDocker.Registry = ""
		}
		if modelBtes, err = json.Marshal(m.ModelDocker); err != nil {
			return err
		}
	default:
		var err error
		if modelBtes, err = json.Marshal(m.ModelVirtualMachine); err != nil {
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
		return sdk.WithStack(err)
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
	var createdBy, model, registeredOS, registeredArch, lastSpawnErr, lastSpawnErrLogs sql.NullString
	if err := s.QueryRow(`
    SELECT
      created_by, model, registered_os, registered_arch, last_spawn_err, last_spawn_err_log
    FROM worker_model
    WHERE id = $1
  `, m.ID).Scan(&createdBy, &model, &registeredOS,
		&registeredArch, &lastSpawnErr, &lastSpawnErrLogs); err != nil {
		return sdk.WrapError(err, "unable to load created_by, model, registered_os, registered_arch")
	}

	if registeredOS.Valid {
		m.RegisteredOS = registeredOS.String
	}

	if registeredArch.Valid {
		m.RegisteredArch = registeredArch.String
	}

	if lastSpawnErr.Valid {
		m.LastSpawnErr = lastSpawnErr.String
	}

	if lastSpawnErrLogs.Valid {
		m.LastSpawnErrLogs = &lastSpawnErrLogs.String
	}

	switch m.Type {
	case sdk.Docker:
		if err := gorpmapping.JSONNullString(model, &m.ModelDocker); err != nil {
			return sdk.WrapError(err, "cannot unmarshall for docker model")
		}
	default:
		if err := gorpmapping.JSONNullString(model, &m.ModelVirtualMachine); err != nil {
			return sdk.WrapError(err, "cannot unmarshall for vm model")
		}
	}

	if err := gorpmapping.JSONNullString(createdBy, &m.Author); err != nil {
		return sdk.WithStack(err)
	}

	if m.GroupID == group.SharedInfraGroup.ID {
		m.IsOfficial = true
	}

	return nil
}

// workerModelPattern is a gorp wrapper around sdk.ModelPattern
type workerModelPattern sdk.ModelPattern

//PostGet load capabilitites and createdBy user
func (wmp *workerModelPattern) PostGet(s gorp.SqlExecutor) error {
	modelStr, err := s.SelectNullStr("SELECT model FROM worker_model_pattern WHERE id = $1", wmp.ID)
	if err != nil {
		return sdk.WrapError(err, "Cannot load model for pattern %d", wmp.ID)
	}

	if err := gorpmapping.JSONNullString(modelStr, &wmp.Model); err != nil {
		return sdk.WrapError(err, "Cannot unmarshal json from model pattern")
	}

	return nil
}

//PostInsert is a DB Hook on workerModelPattern
func (wmp *workerModelPattern) PostInsert(s gorp.SqlExecutor) error {
	modelBtes, err := json.Marshal(wmp.Model)
	if err != nil {
		return sdk.WrapError(err, "Cannot marshal model")
	}

	query := "update worker_model_pattern set model = $1 where id = $2"
	if _, err := s.Exec(query, modelBtes, wmp.ID); err != nil {
		return sdk.WrapError(err, "Cannot update model")
	}

	return nil
}

//PostUpdate is a DB Hook on workerModelPattern
func (wmp *workerModelPattern) PostUpdate(s gorp.SqlExecutor) error {
	return wmp.PostInsert(s)
}
