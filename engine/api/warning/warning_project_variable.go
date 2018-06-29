package warning

import (
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

type unusedProjectVariableWarning struct {
	commonWarn
}

func (warn unusedProjectVariableWarning) events() []string {
	return []string{
		fmt.Sprintf("%T", sdk.EventProjectVariableAdd{}),
		fmt.Sprintf("%T", sdk.EventProjectVariableUpdate{}),
		fmt.Sprintf("%T", sdk.EventProjectVariableDelete{}),
		fmt.Sprintf("%T", sdk.EventApplicationVariableAdd{}),
		fmt.Sprintf("%T", sdk.EventApplicationVariableUpdate{}),
		fmt.Sprintf("%T", sdk.EventApplicationVariableDelete{}),
		fmt.Sprintf("%T", sdk.EventPipelineJobAdd{}),
		fmt.Sprintf("%T", sdk.EventPipelineJobUpdate{}),
		fmt.Sprintf("%T", sdk.EventPipelineJobDelete{}),
		fmt.Sprintf("%T", sdk.EventEnvironmentVariableAdd{}),
		fmt.Sprintf("%T", sdk.EventEnvironmentVariableUpdate{}),
		fmt.Sprintf("%T", sdk.EventEnvironmentVariableDelete{}),
		fmt.Sprintf("%T", sdk.EventWorkflowAdd{}),
		fmt.Sprintf("%T", sdk.EventWorkflowUpdate{}),
		fmt.Sprintf("%T", sdk.EventWorkflowDelete{}),
	}
}

func (warn unusedProjectVariableWarning) name() string {
	return sdk.WarningUnusedProjectVariable
}

func (warn unusedProjectVariableWarning) compute(db gorp.SqlExecutor, e sdk.Event) error {
	switch e.EventType {
	case fmt.Sprintf("%T", sdk.EventProjectVariableAdd{}):

		payload, err := e.ToEventProjectVariableAdd()
		if err != nil {
			return sdk.WrapError(err, "unusedProjectVariableWarning.compute> Unable to get payload from EventProjectVariableAdd")
		}
		varName := fmt.Sprintf("cds.proj.%s", payload.Variable.Name)
		if err := checkUnusedProjectVariable(db, e.ProjectKey, varName, warn.name()); err != nil {
			return sdk.WrapError(err, "unusedProjectVariableWarning.checkProjectVariable>EventProjectVariableAdd")
		}

	case fmt.Sprintf("%T", sdk.EventProjectVariableUpdate{}):

		payload, err := e.ToEventProjectVariableUpdate()
		if err != nil {
			return sdk.WrapError(err, "unusedProjectVariableWarning.compute> Unable to get payload from EventProjectVariableUpdate")
		}
		if err := removeProjectWarning(db, warn.name(), fmt.Sprintf("cds.proj.%s", payload.OldVariable.Name), e.ProjectKey); err != nil {
			return sdk.WrapError(err, "unusedProjectVariableWarning.compute> Unable to remove warning from EventProjectVariableUpdate")
		}
		varName := fmt.Sprintf("cds.proj.%s", payload.NewVariable.Name)
		if err := checkUnusedProjectVariable(db, e.ProjectKey, varName, warn.name()); err != nil {
			return sdk.WrapError(err, "unusedProjectVariableWarning.checkProjectVariable>EventProjectVariableUpdate")
		}

	case fmt.Sprintf("%T", sdk.EventProjectVariableDelete{}):

		payload, err := e.ToEventProjectVariableDelete()
		if err != nil {
			return sdk.WrapError(err, "unusedProjectVariableWarning.compute> Unable to get payload from EventProjectVariableDelete")
		}
		varName := fmt.Sprintf("cds.proj.%s", payload.Variable.Name)
		if err := removeProjectWarning(db, warn.name(), varName, e.ProjectKey); err != nil {
			return sdk.WrapError(err, "unusedProjectVariableWarning.compute> Unable to remove warning from EventProjectVariableDelete")
		}

	case fmt.Sprintf("%T", sdk.EventApplicationVariableAdd{}):

		payload, err := e.ToEventApplicationVariableAdd()
		if err != nil {
			return sdk.WrapError(err, "unusedProjectVariableWarning.compute.EventApplicationVariableAdd> Unable to get payload")
		}

		if err := checkContentValueToRemoveUnusedWarning(db, e.ProjectKey, payload.Variable.Value, "cds.proj", projVarRegexp, warn.name()); err != nil {
			return sdk.WrapError(err, "unusedProjectVariableWarning.compute.EventApplicationVariableAdd> Unable to ckeck content and remove warning: %v", err)
		}

	case fmt.Sprintf("%T", sdk.EventApplicationVariableUpdate{}):

		payload, err := e.ToEventApplicationVariableUpdate()
		if err != nil {
			return sdk.WrapError(err, "unusedProjectVariableWarning.compute.EventApplicationVariableUpdate> Unable to get payload")
		}

		if err := checkContentValueToAddUnusedWarning(db, e.ProjectKey, payload.OldVariable.Value, "cds.proj", projVarRegexp, warn.name()); err != nil {
			return sdk.WrapError(err, "unusedProjectVariableWarning.compute.EventApplicationVariableUpdate> Unable to check content value to add unused warning")
		}

		if err := checkContentValueToRemoveUnusedWarning(db, e.ProjectKey, payload.NewVariable.Value, "cds.proj", projVarRegexp, warn.name()); err != nil {
			return sdk.WrapError(err, "unusedProjectVariableWarning.compute.EventApplicationVariableUpdate> Unable to ckeck content and remove warning: %v", err)
		}

	case fmt.Sprintf("%T", sdk.EventApplicationVariableDelete{}):
		payload, err := e.ToEventApplicationVariableDelete()
		if err != nil {
			return sdk.WrapError(err, "unusedProjectVariableWarning.compute.EventApplicationVariableDelete> Unable to get payload")
		}

		if err := checkContentValueToAddUnusedWarning(db, e.ProjectKey, payload.Variable.Value, "cds.proj", projVarRegexp, warn.name()); err != nil {
			return sdk.WrapError(err, "unusedProjectVariableWarning.compute.EventApplicationVariableDelete> Unable to check content value to add unused warning")
		}
	case fmt.Sprintf("%T", sdk.EventPipelineJobAdd{}):
		// TODO
	case fmt.Sprintf("%T", sdk.EventPipelineJobUpdate{}):
		// TODO
	case fmt.Sprintf("%T", sdk.EventPipelineJobDelete{}):
		// TODO
	case fmt.Sprintf("%T", sdk.EventEnvironmentVariableAdd{}):
		// TODO
	case fmt.Sprintf("%T", sdk.EventEnvironmentVariableUpdate{}):
		// TODO
	case fmt.Sprintf("%T", sdk.EventEnvironmentVariableDelete{}):
		// TODO
	case fmt.Sprintf("%T", sdk.EventWorkflowAdd{}):
		// TODO
	case fmt.Sprintf("%T", sdk.EventWorkflowUpdate{}):
		// TODO
	case fmt.Sprintf("%T", sdk.EventWorkflowDelete{}):
		// TODO
	}
	return nil
}

type missingProjectVariableEnv struct {
	commonWarn
}

func (warn missingProjectVariableEnv) events() []string {
	return []string{
		fmt.Sprintf("%T", sdk.EventProjectVariableAdd{}),
		fmt.Sprintf("%T", sdk.EventProjectVariableUpdate{}),
		fmt.Sprintf("%T", sdk.EventProjectVariableDelete{}),
	}
}

func (warn missingProjectVariableEnv) name() string {
	return sdk.WarningMissingProjectVariableEnv
}

func (warn missingProjectVariableEnv) compute(db gorp.SqlExecutor, e sdk.Event) error {
	switch e.EventType {
	case fmt.Sprintf("%T", sdk.EventProjectVariableAdd{}):
		payload, err := e.ToEventProjectVariableAdd()
		if err != nil {
			return sdk.WrapError(err, "missingProjectVariableEnv.compute> Unable to get payload from EventProjectVariableAdd")
		}
		if err := removeProjectWarning(db, warn.name(), fmt.Sprintf("cds.proj.%s", payload.Variable.Name), e.ProjectKey); err != nil {
			return sdk.WrapError(err, "missingProjectVariableEnv.compute> Unable to remove warning from EventProjectVariableUpdate")
		}
	case fmt.Sprintf("%T", sdk.EventProjectVariableUpdate{}):
		payload, err := e.ToEventProjectVariableUpdate()
		if err != nil {
			return sdk.WrapError(err, "missingProjectVariableEnv.compute> Unable to get payload from EventProjectVariableUpdate")
		}
		if err := removeProjectWarning(db, warn.name(), fmt.Sprintf("cds.proj.%s", payload.NewVariable.Name), e.ProjectKey); err != nil {
			return sdk.WrapError(err, "missingProjectVariableEnv.compute> Unable to remove warning from EventProjectVariableUpdate")
		}
	case fmt.Sprintf("%T", sdk.EventProjectVariableDelete{}):
		payload, err := e.ToEventProjectVariableDelete()
		if err != nil {
			return sdk.WrapError(err, "missingProjectVariableEnv.compute> Unable to get payload from EventProjectVariableDelete")
		}
		varName := fmt.Sprintf("cds.proj.%s", payload.Variable.Name)
		envs, errE := environment.CountEnvironmentByVarValue(db, e.ProjectKey, varName)
		if errE != nil {
			return sdk.WrapError(errE, "missingProjectVariableEnv> Unable to list environment")
		}
		for _, envName := range envs {
			w := sdk.Warning{
				Key:     e.ProjectKey,
				EnvName: envName,
				Element: varName,
				Created: time.Now(),
				Type:    warn.name(),
				MessageParams: map[string]string{
					"VarName":         varName,
					"ProjectKey":      e.ProjectKey,
					"EnvironmentName": envName,
				},
			}
			if err := Insert(db, w); err != nil {
				return sdk.WrapError(err, "missingProjectVariableEnv.compute> Unable to Insert warning")
			}
		}
	}
	return nil
}

type missingProjectVariableWorkflow struct {
	commonWarn
}

func (warn missingProjectVariableWorkflow) events() []string {
	return []string{
		fmt.Sprintf("%T", sdk.EventProjectVariableAdd{}),
		fmt.Sprintf("%T", sdk.EventProjectVariableUpdate{}),
		fmt.Sprintf("%T", sdk.EventProjectVariableDelete{}),
	}
}

func (warn missingProjectVariableWorkflow) name() string {
	return sdk.WarningMissingProjectVariableWorkflow
}

func (warn missingProjectVariableWorkflow) compute(db gorp.SqlExecutor, e sdk.Event) error {
	switch e.EventType {
	case fmt.Sprintf("%T", sdk.EventProjectVariableAdd{}):
		payload, err := e.ToEventProjectVariableAdd()
		if err != nil {
			return sdk.WrapError(err, "missingProjectVariableWorkflow.compute> Unable to get payload from EventProjectVariableAdd")
		}
		if err := removeProjectWarning(db, warn.name(), fmt.Sprintf("cds.proj.%s", payload.Variable.Name), e.ProjectKey); err != nil {
			return sdk.WrapError(err, "missingProjectVariableWorkflow.compute> Unable to remove warning from EventProjectVariableUpdate")
		}
	case fmt.Sprintf("%T", sdk.EventProjectVariableUpdate{}):
		payload, err := e.ToEventProjectVariableUpdate()
		if err != nil {
			return sdk.WrapError(err, "missingProjectVariableWorkflow.compute> Unable to get payload from EventProjectVariableUpdate")
		}
		if err := removeProjectWarning(db, warn.name(), fmt.Sprintf("cds.proj.%s", payload.NewVariable.Name), e.ProjectKey); err != nil {
			return sdk.WrapError(err, "missingProjectVariableWorkflow.compute> Unable to remove warning from EventProjectVariableUpdate")
		}
	case fmt.Sprintf("%T", sdk.EventProjectVariableDelete{}):
		payload, err := e.ToEventProjectVariableDelete()
		if err != nil {
			return sdk.WrapError(err, "missingProjectVariableWorkflow.compute> Unable to get payload from EventProjectVariableDelete")
		}
		varName := fmt.Sprintf("cds.proj.%s", payload.Variable.Name)
		workflows, errW := workflow.CountVariableInWorkflow(db, e.ProjectKey, varName)
		if errW != nil {
			return sdk.WrapError(errW, "missingProjectVariableWorkflow.compute> Unable to get workflows")
		}
		for _, wName := range workflows {
			w := sdk.Warning{
				Key:          e.ProjectKey,
				WorkflowName: wName.WorkflowName,
				Element:      varName,
				Created:      time.Now(),
				Type:         warn.name(),
				MessageParams: map[string]string{
					"VarName":      varName,
					"ProjectKey":   e.ProjectKey,
					"WorkflowName": wName.WorkflowName,
					"NodeName":     wName.NodeName,
				},
			}
			if err := Insert(db, w); err != nil {
				return sdk.WrapError(err, "missingProjectVariableWorkflow.compute> Unable to Insert warning")
			}
		}
	}
	return nil
}

type missingProjectVariableApplication struct {
	commonWarn
}

func (warn missingProjectVariableApplication) events() []string {
	return []string{
		fmt.Sprintf("%T", sdk.EventProjectVariableAdd{}),
		fmt.Sprintf("%T", sdk.EventProjectVariableUpdate{}),
		fmt.Sprintf("%T", sdk.EventProjectVariableDelete{}),
	}
}

func (warn missingProjectVariableApplication) name() string {
	return sdk.WarningMissingProjectVariableApplication
}

func (warn missingProjectVariableApplication) compute(db gorp.SqlExecutor, e sdk.Event) error {
	switch e.EventType {
	case fmt.Sprintf("%T", sdk.EventProjectVariableAdd{}):
		payload, err := e.ToEventProjectVariableAdd()
		if err != nil {
			return sdk.WrapError(err, "missingProjectVariableApplication.compute> Unable to get payload from EventProjectVariableAdd")
		}
		if err := removeProjectWarning(db, warn.name(), fmt.Sprintf("cds.proj.%s", payload.Variable.Name), e.ProjectKey); err != nil {
			return sdk.WrapError(err, "missingProjectVariableApplication.compute> Unable to remove warning from EventProjectVariableUpdate")
		}
	case fmt.Sprintf("%T", sdk.EventProjectVariableUpdate{}):
		payload, err := e.ToEventProjectVariableUpdate()
		if err != nil {
			return sdk.WrapError(err, "missingProjectVariableApplication.compute> Unable to get payload from EventProjectVariableUpdate")
		}
		if err := removeProjectWarning(db, warn.name(), fmt.Sprintf("cds.proj.%s", payload.NewVariable.Name), e.ProjectKey); err != nil {
			return sdk.WrapError(err, "missingProjectVariableApplication.compute> Unable to remove warning from EventProjectVariableUpdate")
		}
	case fmt.Sprintf("%T", sdk.EventProjectVariableDelete{}):
		payload, err := e.ToEventProjectVariableDelete()
		if err != nil {
			return sdk.WrapError(err, "missingProjectVariableApplication.compute> Unable to get payload from EventProjectVariableDelete")
		}
		varName := fmt.Sprintf("cds.proj.%s", payload.Variable.Name)
		apps, errA := application.CountInVarValue(db, e.ProjectKey, varName)
		if errA != nil {
			return sdk.WrapError(errA, "missingProjectVariableApplication.compute> Unable to list application")
		}
		for _, appName := range apps {
			w := sdk.Warning{
				Key:     e.ProjectKey,
				AppName: appName,
				Element: varName,
				Created: time.Now(),
				Type:    warn.name(),
				MessageParams: map[string]string{
					"VarName":         varName,
					"ProjectKey":      e.ProjectKey,
					"ApplicationName": appName,
				},
			}
			if err := Insert(db, w); err != nil {
				return sdk.WrapError(err, "missingProjectVariableApplication.compute> Unable to Insert warning")
			}
		}
	}
	return nil
}

type missingProjectVariablePipelineParameter struct {
	commonWarn
}

func (warn missingProjectVariablePipelineParameter) events() []string {
	return []string{
		fmt.Sprintf("%T", sdk.EventProjectVariableAdd{}),
		fmt.Sprintf("%T", sdk.EventProjectVariableUpdate{}),
		fmt.Sprintf("%T", sdk.EventProjectVariableDelete{}),
	}
}

func (warn missingProjectVariablePipelineParameter) name() string {
	return sdk.WarningMissingProjectVariablePipelineParameter
}

func (warn missingProjectVariablePipelineParameter) compute(db gorp.SqlExecutor, e sdk.Event) error {
	switch e.EventType {
	case fmt.Sprintf("%T", sdk.EventProjectVariableAdd{}):
		payload, err := e.ToEventProjectVariableAdd()
		if err != nil {
			return sdk.WrapError(err, "missingProjectVariablePipelineParameter.compute> Unable to get payload from EventProjectVariableAdd")
		}
		if err := removeProjectWarning(db, warn.name(), fmt.Sprintf("cds.proj.%s", payload.Variable.Name), e.ProjectKey); err != nil {
			return sdk.WrapError(err, "missingProjectVariablePipelineParameter.compute> Unable to remove warning from EventProjectVariableUpdate")
		}
	case fmt.Sprintf("%T", sdk.EventProjectVariableUpdate{}):
		payload, err := e.ToEventProjectVariableUpdate()
		if err != nil {
			return sdk.WrapError(err, "missingProjectVariablePipelineParameter.compute> Unable to get payload from EventProjectVariableUpdate")
		}
		if err := removeProjectWarning(db, warn.name(), fmt.Sprintf("cds.proj.%s", payload.NewVariable.Name), e.ProjectKey); err != nil {
			return sdk.WrapError(err, "missingProjectVariablePipelineParameter.compute> Unable to remove warning from EventProjectVariableUpdate")
		}
	case fmt.Sprintf("%T", sdk.EventProjectVariableDelete{}):
		payload, err := e.ToEventProjectVariableDelete()
		if err != nil {
			return sdk.WrapError(err, "missingProjectVariablePipelineParameter.compute> Unable to get payload from EventProjectVariableDelete")
		}
		varName := fmt.Sprintf("cds.proj.%s", payload.Variable.Name)
		pips, errP := pipeline.CountInParamValue(db, e.ProjectKey, varName)
		if errP != nil {
			return sdk.WrapError(errP, "missingProjectVariablePipelineParameter.compute> Unable to list pipeline")
		}
		for _, pipName := range pips {
			w := sdk.Warning{
				Key:     e.ProjectKey,
				PipName: pipName,
				Element: varName,
				Created: time.Now(),
				Type:    warn.name(),
				MessageParams: map[string]string{
					"VarName":      varName,
					"ProjectKey":   e.ProjectKey,
					"PipelineName": pipName,
				},
			}
			if err := Insert(db, w); err != nil {
				return sdk.WrapError(err, "missingProjectVariablePipelineParameter.compute> Unable to Insert warning")
			}
		}
	}
	return nil
}

type missingProjectVariablePipelineJob struct {
	commonWarn
}

func (warn missingProjectVariablePipelineJob) events() []string {
	return []string{
		fmt.Sprintf("%T", sdk.EventProjectVariableAdd{}),
		fmt.Sprintf("%T", sdk.EventProjectVariableUpdate{}),
		fmt.Sprintf("%T", sdk.EventProjectVariableDelete{}),
	}
}

func (warn missingProjectVariablePipelineJob) name() string {
	return sdk.WarningMissingProjectVariablePipelineJob
}

func (warn missingProjectVariablePipelineJob) compute(db gorp.SqlExecutor, e sdk.Event) error {
	switch e.EventType {
	case fmt.Sprintf("%T", sdk.EventProjectVariableAdd{}):
		payload, err := e.ToEventProjectVariableAdd()
		if err != nil {
			return sdk.WrapError(err, "missingProjectVariablePipelineJob.compute> Unable to get payload from EventProjectVariableAdd")
		}
		if err := removeProjectWarning(db, warn.name(), fmt.Sprintf("cds.proj.%s", payload.Variable.Name), e.ProjectKey); err != nil {
			return sdk.WrapError(err, "missingProjectVariablePipelineJob.compute> Unable to remove warning from EventProjectVariableUpdate")
		}
	case fmt.Sprintf("%T", sdk.EventProjectVariableUpdate{}):
		payload, err := e.ToEventProjectVariableUpdate()
		if err != nil {
			return sdk.WrapError(err, "missingProjectVariablePipelineJob.compute> Unable to get payload from EventProjectVariableUpdate")
		}
		if err := removeProjectWarning(db, warn.name(), fmt.Sprintf("cds.proj.%s", payload.NewVariable.Name), e.ProjectKey); err != nil {
			return sdk.WrapError(err, "missingProjectVariablePipelineJob.compute> Unable to remove warning from EventProjectVariableUpdate")
		}
	case fmt.Sprintf("%T", sdk.EventProjectVariableDelete{}):
		payload, err := e.ToEventProjectVariableDelete()
		if err != nil {
			return sdk.WrapError(err, "missingProjectVariablePipelineJob.compute> Unable to get payload from EventProjectVariableDelete")
		}
		varName := fmt.Sprintf("cds.proj.%s", payload.Variable.Name)
		pips, errP := pipeline.CountInPipelines(db, e.ProjectKey, varName)
		if errP != nil {
			return sdk.WrapError(errP, "missingProjectVariablePipelineJob.compute> Unable to list pipeline")
		}
		for _, pip := range pips {
			w := sdk.Warning{
				Key:     e.ProjectKey,
				PipName: pip.PipName,
				Element: varName,
				Created: time.Now(),
				Type:    warn.name(),
				MessageParams: map[string]string{
					"VarName":      varName,
					"ProjectKey":   e.ProjectKey,
					"PipelineName": pip.PipName,
					"StageName":    pip.StageName,
					"JobName":      pip.JobName,
				},
			}
			if err := Insert(db, w); err != nil {
				return sdk.WrapError(err, "missingProjectVariablePipelineJob.compute> Unable to Insert warning")
			}
		}
	}
	return nil
}
