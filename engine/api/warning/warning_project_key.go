package warning

import (
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type unusedProjectKeyWarning struct {
	commonWarn
}

func (warn unusedProjectKeyWarning) events() []string {
	return []string{
		fmt.Sprintf("%T", sdk.EventProjectKeyAdd{}),
		fmt.Sprintf("%T", sdk.EventProjectKeyDelete{}),
	}
}

func (warn unusedProjectKeyWarning) name() string {
	return sdk.WarningUnusedProjectKey
}

func (warn unusedProjectKeyWarning) compute(db gorp.SqlExecutor, e sdk.Event) error {
	switch e.EventType {
	case fmt.Sprintf("%T", sdk.EventProjectKeyAdd{}):
		payload, err := e.ToEventProjectKeyAdd()
		if err != nil {
			return sdk.WrapError(err, "unusedProjectKeyWarning.compute> Unable to get payload from EventProjectKeyAdd")
		}

		apps, pips, pipJobs := keyIsUsed(db, e.ProjectKey, payload.Key.Name)
		if len(apps) == 0 && len(pips) == 0 && len(pipJobs) == 0 {
			w := sdk.WarningV2{
				Key:     e.ProjectKey,
				Element: payload.Key.Name,
				Created: time.Now(),
				Type:    warn.name(),
				MessageParams: map[string]string{
					"KeyName":    payload.Key.Name,
					"ProjectKey": e.ProjectKey,
				},
			}
			if err := Insert(db, w); err != nil {
				return sdk.WrapError(err, "unusedProjectKeyWarning> Unable to Insert warning")
			}
		}
	case fmt.Sprintf("%T", sdk.EventProjectKeyDelete{}):
		payload, err := e.ToEventProjectKeyDelete()
		if err != nil {
			return sdk.WrapError(err, "unusedProjectKeyWarning.compute> Unable to get payload from EventProjectKeyDelete")
		}
		if err := removeProjectWarning(db, warn.name(), payload.Key.Name, e.ProjectKey); err != nil {
			return sdk.WrapError(err, "unusedProjectKeyWarning.compute> Unable to remove warning from EventProjectKeyDelete")
		}
	}
	return nil
}

type missingProjectKeyPipelineParameterWarning struct {
	commonWarn
}

func (warn missingProjectKeyPipelineParameterWarning) events() []string {
	return []string{
		fmt.Sprintf("%T", sdk.EventProjectKeyAdd{}),
		fmt.Sprintf("%T", sdk.EventProjectKeyDelete{}),
	}
}

func (warn missingProjectKeyPipelineParameterWarning) name() string {
	return sdk.WarningMissingProjectKeyPipelineParameter
}

func (warn missingProjectKeyPipelineParameterWarning) compute(db gorp.SqlExecutor, e sdk.Event) error {
	switch e.EventType {
	case fmt.Sprintf("%T", sdk.EventProjectKeyAdd{}):
		payload, err := e.ToEventProjectKeyAdd()
		if err != nil {
			return sdk.WrapError(err, "missingProjectKeyPipelineParameterWarning.compute> Unable to get payload from EventProjectKeyAdd")
		}
		if err := removeProjectWarning(db, warn.name(), payload.Key.Name, e.ProjectKey); err != nil {
			return sdk.WrapError(err, "missingProjectKeyPipelineParameterWarning.compute> Unable to remove warning from EventProjectKeyAdd")
		}
	case fmt.Sprintf("%T", sdk.EventProjectKeyDelete{}):
		payload, err := e.ToEventProjectKeyDelete()
		if err != nil {
			return sdk.WrapError(err, "missingProjectKeyPipelineParameterWarning.compute> Unable to get payload from EventProjectKeyDelete")
		}
		pips, err := pipeline.CountInParamValue(db, e.ProjectKey, payload.Key.Name)
		if err != nil {
			return sdk.WrapError(err, "missingProjectKeyPipelineParameterWarning.compute> Unable to list pipeline")
		}
		for _, p := range pips {
			w := sdk.WarningV2{
				Key:     e.ProjectKey,
				PipName: p,
				Element: payload.Key.Name,
				Created: time.Now(),
				Type:    warn.name(),
				MessageParams: map[string]string{
					"KeyName":      payload.Key.Name,
					"ProjectKey":   e.ProjectKey,
					"PipelineName": p,
				},
			}
			if err := Insert(db, w); err != nil {
				return sdk.WrapError(err, "missingProjectKeyPipelineParameterWarning.compute> Unable to Insert warning")
			}
		}
	}
	return nil
}

type missingProjectKeyPipelineJobWarning struct {
	commonWarn
}

func (warn missingProjectKeyPipelineJobWarning) events() []string {
	return []string{
		fmt.Sprintf("%T", sdk.EventProjectKeyAdd{}),
		fmt.Sprintf("%T", sdk.EventProjectKeyDelete{}),
	}
}

func (warn missingProjectKeyPipelineJobWarning) name() string {
	return sdk.WarningMissingProjectKeyPipelineJob
}

func (warn missingProjectKeyPipelineJobWarning) compute(db gorp.SqlExecutor, e sdk.Event) error {
	switch e.EventType {
	case fmt.Sprintf("%T", sdk.EventProjectKeyAdd{}):
		payload, err := e.ToEventProjectKeyAdd()
		if err != nil {
			return sdk.WrapError(err, "missingProjectKeyPipelineJobWarning.compute> Unable to get payload from EventProjectKeyAdd")
		}
		if err := removeProjectWarning(db, warn.name(), payload.Key.Name, e.ProjectKey); err != nil {
			return sdk.WrapError(err, "missingProjectKeyPipelineJobWarning.compute> Unable to remove warning from EventProjectKeyAdd")
		}
	case fmt.Sprintf("%T", sdk.EventProjectKeyDelete{}):
		payload, err := e.ToEventProjectKeyDelete()
		if err != nil {
			return sdk.WrapError(err, "missingProjectKeyPipelineJobWarning.compute> Unable to get payload from EventProjectKeyDelete")
		}
		pips, err := pipeline.CountInPipelines(db, e.ProjectKey, payload.Key.Name)
		if err != nil {
			return sdk.WrapError(err, "missingProjectKeyPipelineJobWarning.compute> Unable to list pipeline")
		}
		for _, p := range pips {
			w := sdk.WarningV2{
				Key:     e.ProjectKey,
				PipName: p.PipName,
				Element: payload.Key.Name,
				Created: time.Now(),
				Type:    warn.name(),
				MessageParams: map[string]string{
					"KeyName":      payload.Key.Name,
					"ProjectKey":   e.ProjectKey,
					"PipelineName": p.PipName,
					"StageName":    p.StageName,
					"JobName":      p.JobName,
				},
			}
			if err := Insert(db, w); err != nil {
				return sdk.WrapError(err, "missingProjectKeyPipelineJobWarning.compute> Unable to Insert warning")
			}
		}
	}
	return nil
}

type missingProjectKeyApplicationWarning struct {
	commonWarn
}

func (warn missingProjectKeyApplicationWarning) events() []string {
	return []string{
		fmt.Sprintf("%T", sdk.EventProjectKeyAdd{}),
		fmt.Sprintf("%T", sdk.EventProjectKeyDelete{}),
	}
}

func (warn missingProjectKeyApplicationWarning) name() string {
	return sdk.WarningMissingProjectKeyApplication
}

func (warn missingProjectKeyApplicationWarning) compute(db gorp.SqlExecutor, e sdk.Event) error {
	switch e.EventType {
	case fmt.Sprintf("%T", sdk.EventProjectKeyAdd{}):
		payload, err := e.ToEventProjectKeyAdd()
		if err != nil {
			return sdk.WrapError(err, "missingProjectKeyApplicationWarning.compute> Unable to get payload from EventProjectKeyAdd")
		}
		if err := removeProjectWarning(db, warn.name(), payload.Key.Name, e.ProjectKey); err != nil {
			log.Warning("missingProjectKeyApplicationWarning.compute> Unable to remove warning from EventProjectKeyAdd")
		}
	case fmt.Sprintf("%T", sdk.EventProjectKeyDelete{}):
		payload, err := e.ToEventProjectKeyDelete()
		if err != nil {
			return sdk.WrapError(err, "missingProjectKeyApplicationWarning.compute> Unable to get payload from EventProjectKeyDelete")
		}
		apps, err := application.CountApplicationByVcsConfigurationKeys(db, e.ProjectKey, payload.Key.Name)
		if err != nil {
			return sdk.WrapError(err, "missingProjectKeyApplicationWarning.compute> Unable to list pipeline")
		}
		for _, a := range apps {
			w := sdk.WarningV2{
				Key:     e.ProjectKey,
				AppName: a,
				Element: payload.Key.Name,
				Created: time.Now(),
				Type:    warn.name(),
				MessageParams: map[string]string{
					"KeyName":         payload.Key.Name,
					"ProjectKey":      e.ProjectKey,
					"ApplicationName": a,
				},
			}
			if err := Insert(db, w); err != nil {
				return sdk.WrapError(err, "missingProjectKeyApplicationWarning.compute> Unable to Insert warning")
			}
		}
	}
	return nil
}
