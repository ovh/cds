package pipeline

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/sdk"
)

// LoadStage Get a stage from its ID and pipeline ID
func LoadStage(ctx context.Context, db gorp.SqlExecutor, pipelineID int64, stageID int64) (*sdk.Stage, error) {
	query := gorpmapping.NewQuery(`
		SELECT * 
		FROM pipeline_stage
		WHERE pipeline_id = $1 AND id = $2
	`).Args(pipelineID, stageID)
	var dbStage dbPipelineStage
	if _, err := gorpmapping.Get(ctx, db, query, &dbStage); err != nil {
		return nil, err
	}
	stage := dbStage.Stage()
	return &stage, nil
}

// InsertStage insert given stage into given database
func InsertStage(db gorp.SqlExecutor, s *sdk.Stage) error {
	if s.Conditions.LuaScript != "" {
		s.Conditions.PlainConditions = nil
	}
	s.LastModified = time.Now()
	dbStage := newdbStage(*s)
	if err := gorpmapping.Insert(db, &dbStage); err != nil {
		return err
	}
	s.ID = dbStage.ID
	return nil
}

// LoadPipelineStage loads pipeline stage
func LoadPipelineStage(ctx context.Context, db gorp.SqlExecutor, p *sdk.Pipeline) error {
	_, end := observability.Span(ctx, "pipeline.LoadPipelineStage")
	defer end()

	var dbStages []dbPipelineStage
	query := gorpmapping.NewQuery(`
		SELECT * from pipeline_stage
		WHERE pipeline_id = $1
    `).Args(p.ID)
	if err := gorpmapping.GetAll(ctx, db, query, &dbStages); err != nil {
		return err
	}
	p.Stages = make([]sdk.Stage, len(dbStages))
	stageIDs := make([]int64, len(dbStages))
	for i, dbS := range dbStages {
		p.Stages[i] = dbS.Stage()
		stageIDs[i] = dbS.ID
	}

	// Get all jobs
	jobs, err := getJobs(ctx, db, stageIDs)
	if err != nil {
		return err
	}

	// Add job in stage
	for _, j := range jobs {
		for i := range p.Stages {
			s := &p.Stages[i]
			if s.ID == j.PipelineStageID {
				s.Jobs = append(s.Jobs, j)
				break
			}
		}
	}
	return nil
}

// UpdateStage update Stage and all its prequisites
func UpdateStage(db gorp.SqlExecutor, s *sdk.Stage) error {
	dbStage := newdbStage(*s)
	if err := gorpmapping.Update(db, &dbStage); err != nil {
		return err
	}
	return nil
}

// DeleteStageByID Delete stage with associated pipeline action
func DeleteStageByID(ctx context.Context, tx gorp.SqlExecutor, s *sdk.Stage) error {
	// TODO  refactor to use delete cascade

	nbOfStages, err := CountStageByPipelineID(tx, s.PipelineID)
	if err != nil {
		return err
	}

	if err := DeletePipelineActionByStage(ctx, tx, s.ID); err != nil {
		return err
	}

	if err := deleteStageByID(tx, s); err != nil {
		return err
	}

	return moveDownStages(tx, s.PipelineID, s.BuildOrder, nbOfStages)
}

func deleteStageByID(tx gorp.SqlExecutor, s *sdk.Stage) error {
	dbStage := newdbStage(*s)
	return gorpmapping.Delete(tx, &dbStage)
}

// CountStageByPipelineID Count the number of stages for the given pipeline
func CountStageByPipelineID(db gorp.SqlExecutor, pipelineID int64) (int, error) {
	query := `SELECT count(id) FROM "pipeline_stage"
	 		  WHERE pipeline_id = $1`
	countStages, err := gorp.SelectInt(db, query, pipelineID)
	if err != nil {
		return 0, sdk.WithStack(err)
	}
	return int(countStages), nil
}

func seleteAllStageID(db gorp.SqlExecutor, pipelineID int64) ([]int64, error) {
	var stageIDs []int64

	query := `
		SELECT id FROM "pipeline_stage"
		WHERE pipeline_id = $1
	`
	if _, err := db.Select(&stageIDs, query, pipelineID); err != nil {
		return nil, err
	}
	return stageIDs, nil
}

// DeleteAllStage  Delete all stages from pipeline ID
func DeleteAllStage(ctx context.Context, db gorp.SqlExecutor, pipelineID int64) error {
	stageIDs, err := seleteAllStageID(db, pipelineID)
	if err != nil {
		return err
	}

	for _, id := range stageIDs {
		if err := DeletePipelineActionByStage(ctx, db, id); err != nil {
			return err
		}
	}

	queryDelete := `DELETE FROM pipeline_stage WHERE pipeline_id = $1`
	_, err = db.Exec(queryDelete, pipelineID)
	return sdk.WithStack(err)
}

// MoveStage Move a stage
func MoveStage(db gorp.SqlExecutor, stageToMove *sdk.Stage, newBuildOrder int) error {
	if stageToMove.BuildOrder > newBuildOrder {
		if err := moveUpStages(db, stageToMove.PipelineID, stageToMove.BuildOrder, newBuildOrder); err != nil {
			return err
		}
	} else if stageToMove.BuildOrder < newBuildOrder {
		if err := moveDownStages(db, stageToMove.PipelineID, stageToMove.BuildOrder, newBuildOrder); err != nil {
			return err
		}
	}
	stageToMove.BuildOrder = newBuildOrder
	return UpdateStage(db, stageToMove)
}

func moveUpStages(db gorp.SqlExecutor, pipelineID int64, oldPosition, newPosition int) error {
	query := `
		UPDATE pipeline_stage
		SET build_order=build_order+1
 		WHERE build_order < $1
		AND build_order >= $2
		AND pipeline_id = $3`
	_, err := db.Exec(query, oldPosition, newPosition, pipelineID)
	return sdk.WithStack(err)
}

func moveDownStages(db gorp.SqlExecutor, pipelineID int64, oldPosition, newPosition int) error {
	query := `UPDATE pipeline_stage
		  SET build_order=build_order-1
		  WHERE build_order <= $1
		  AND build_order > $2
		  AND pipeline_id = $3`
	_, err := db.Exec(query, newPosition, oldPosition, pipelineID)
	return sdk.WithStack(err)
}
