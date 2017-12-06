package pipeline

import (
	"encoding/json"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

//Audit constants
const (
	AuditAddJob         = "addJob"
	AuditUpdateJob      = "updateJob"
	AuditDeleteJob      = "deleteJob"
	AuditAddStage       = "addStage"
	AuditUpdateStage    = "updateStage"
	AuditDeleteStage    = "deleteStage"
	AuditMoveStage      = "moveStage"
	AuditUpdatePipeline = "updateStage"
)

// CreateAudit insert current pipeline version on audit table
func CreateAudit(db gorp.SqlExecutor, pip *sdk.Pipeline, action string, u *sdk.User) error {
	pipAudit := &sdk.PipelineAudit{
		PipelineID: pip.ID,
		UserName:   u.Username,
		Versionned: time.Now(),
		Pipeline:   pip,
		Action:     action,
	}
	dbmodel := PipelineAudit(*pipAudit)
	return db.Insert(&dbmodel)
}

// LoadAudit load audit for the given pipeline
func LoadAudit(db gorp.SqlExecutor, key string, pipName string) ([]sdk.PipelineAudit, error) {
	query := `
		SELECT pipeline_audit.* FROM pipeline_audit
		JOIN pipeline ON pipeline.id = pipeline_audit.pipeline_id
		JOIN project ON project.id = pipeline.project_id
		WHERE project.projectkey = $1 AND pipeline.name = $2
		ORDER BY pipeline_audit.id DESC
		LIMIT 100
	`
	var auditGorp []PipelineAudit
	if _, err := db.Select(&auditGorp, query, key, pipName); err != nil {
		return nil, err
	}

	var audits []sdk.PipelineAudit
	for i := range auditGorp {
		if err := auditGorp[i].PostGet(db); err != nil {
			return nil, err
		}
		audits = append(audits, sdk.PipelineAudit(auditGorp[i]))
	}
	return audits, nil
}

// DeleteAudit delete audit related to given pipeline
func DeleteAudit(db gorp.SqlExecutor, pipID int64) error {
	_, err := db.Exec("DELETE FROM pipeline_audit WHERE pipeline_id = $1", pipID)
	return err
}

// PostGet is a dbHook on Select to get json column
func (p *PipelineAudit) PostGet(s gorp.SqlExecutor) error {
	query := "SELECT pipeline FROM pipeline_audit WHERE id = $1"
	var pip []byte
	if err := s.QueryRow(query, p.ID).Scan(&pip); err != nil {
		return sdk.WrapError(err, "PostGet> error on queryRow")
	}

	if err := json.Unmarshal(pip, &p.Pipeline); err != nil {
		return sdk.WrapError(err, "PostGet> error on unmarshal job")
	}
	return nil
}

// PostInsert is a DB Hook on PostInsert to store pipeline JSON in DB
func (p *PipelineAudit) PostInsert(s gorp.SqlExecutor) error {
	pipJSON, errPip := json.Marshal(p.Pipeline)
	if errPip != nil {
		return errPip
	}

	query := "update pipeline_audit set pipeline = $1 where id = $2"
	if _, err := s.Exec(query, pipJSON, p.ID); err != nil {
		return sdk.WrapError(err, "PostInsert> err on update sql")
	}
	return nil
}
