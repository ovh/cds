package migrate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type prerequisitesDB struct {
	sdk.Prerequisite
	PipelineStageID int64 `json:"pipeline_stage_id"`
}

func StageConditions(store cache.Store, DBFunc func() *gorp.DbMap) error {
	db := DBFunc()

	var prerequisites []prerequisitesDB
	rows, err := db.Query("SELECT pipeline_stage_id, parameter, expected_value FROM pipeline_stage_prerequisite")
	if err != nil {
		return sdk.WrapError(err, "cannot load all pipeline stage prerequisites")
	}
	defer rows.Close()

	for rows.Next() {
		var prereq prerequisitesDB
		if err := rows.Scan(&prereq.PipelineStageID, &prereq.Parameter, &prereq.ExpectedValue); err != nil {
			return sdk.WrapError(err, "cannot scan row")
		}
		prerequisites = append(prerequisites, prereq)
	}
	fmt.Printf("%+v\n", prerequisites)

	conditionsMap := convertToNewConditions(prerequisites)
	fmt.Printf("%+v\n", conditionsMap)
	for stageID, condition := range conditionsMap {
		conditionBts, err := json.Marshal(condition)
		if err != nil {
			log.Error("Cannot json to null string for condition %+v : %v", condition, err)
			continue
		}

		if _, err := db.Exec("UPDATE pipeline_stage SET conditions = $1 WHERE pipeline_stage.id = $2", conditionBts, stageID); err != nil {
			log.Error("Cannot update pipeline_stage conditions for id %d : %v", stageID, err)
		}
	}

	return nil
}

func convertToNewConditions(prerequisites []prerequisitesDB) map[int64]sdk.WorkflowNodeConditions {
	conditionsMap := map[int64]sdk.WorkflowNodeConditions{}
	for _, p := range prerequisites {
		if !strings.HasPrefix(p.Parameter, "workflow.") && !strings.HasPrefix(p.Parameter, "git.") {
			p.Parameter = "cds.pip." + p.Parameter
		}
		cond := sdk.WorkflowNodeCondition{
			Value:    p.ExpectedValue,
			Variable: p.Parameter,
			Operator: sdk.WorkflowConditionsOperatorRegex,
		}
		plainConditions := append([]sdk.WorkflowNodeCondition{}, conditionsMap[p.PipelineStageID].PlainConditions...)
		plainConditions = append(plainConditions, cond)
		conditionsMap[p.PipelineStageID] = sdk.WorkflowNodeConditions{PlainConditions: plainConditions}
	}

	return conditionsMap
}
