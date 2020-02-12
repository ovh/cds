package pipeline

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
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
	AuditUpdatePipeline = "updatePipeline"
)

// CreateAudit insert current pipeline version on audit table
func CreateAudit(db gorp.SqlExecutor, pip *sdk.Pipeline, action string, u sdk.Identifiable) error {
	pipAudit := &sdk.PipelineAudit{
		PipelineID: pip.ID,
		UserName:   u.GetUsername(),
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

// LoadAuditByID load audit for the given audit id
func LoadAuditByID(db gorp.SqlExecutor, id int64) (sdk.PipelineAudit, error) {
	var pipAudit sdk.PipelineAudit
	query := `
		SELECT pipeline_audit.*
			FROM pipeline_audit
			WHERE pipeline_audit.id = $1
	`
	var auditGorp PipelineAudit
	if err := db.SelectOne(&auditGorp, query, id); err != nil {
		return pipAudit, err
	}
	pipAudit = sdk.PipelineAudit(auditGorp)

	return pipAudit, nil
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
		return sdk.WrapError(err, "error on queryRow")
	}

	if err := json.Unmarshal(pip, &p.Pipeline); err != nil {
		return sdk.WrapError(err, "error on unmarshal job")
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
		return sdk.WrapError(err, "err on update sql")
	}
	return nil
}

const keepAudits = 50

func PurgeAudits(ctx context.Context, db gorp.SqlExecutor) error {
	var nbAuditsPerPipelinewID = []struct {
		PipelineD int64 `db:"pipeline_id"`
		NbAudits  int64 `db:"nb_audits"`
	}{}

	query := `select pipeline_id, count(id) "nb_audits" from pipeline_audit group by pipeline_id having count(id)  > $1`
	if _, err := db.Select(&nbAuditsPerPipelinewID, query, keepAudits); err != nil {
		return sdk.WithStack(err)
	}

	for _, r := range nbAuditsPerPipelinewID {
		log.Debug("purgeAudits> deleting audits for pipeline %d (%d audits)", r.PipelineD, r.NbAudits)
		var ids []int64
		query = `select id from pipeline_audit where pipeline_id = $1 order by versionned desc offset $2`
		if _, err := db.Select(&ids, query, r.PipelineD, keepAudits); err != nil {
			return sdk.WithStack(err)
		}
		for _, id := range ids {
			if err := deleteAudit(db, id); err != nil {
				log.Error(ctx, "purgeAudits> unable to delete audit %d: %v", id, err)
			}
		}
	}

	return nil
}

func deleteAudit(db gorp.SqlExecutor, id int64) error {
	_, err := db.Exec(`delete from pipeline_audit where id = $1`, id)
	return sdk.WithStack(err)
}
