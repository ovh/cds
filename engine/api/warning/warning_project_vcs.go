package warning

import (
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/sdk"
)

type unusedProjectVCSWarning struct {
	commonWarn
}

func (warn unusedProjectVCSWarning) events() []string {
	return []string{
		fmt.Sprintf("%T", sdk.EventProjectVCSServerAdd{}),
		fmt.Sprintf("%T", sdk.EventProjectVCSServerDelete{}),
		fmt.Sprintf("%T", sdk.EventApplicationRepositoryAdd{}),
		fmt.Sprintf("%T", sdk.EventApplicationRepositoryDelete{}),
	}
}

func (warn unusedProjectVCSWarning) name() string {
	return sdk.WarningUnusedProjectVCSServer
}

func (warn unusedProjectVCSWarning) compute(db gorp.SqlExecutor, e sdk.Event) error {
	switch e.EventType {
	case fmt.Sprintf("%T", sdk.EventApplicationRepositoryAdd{}):
		payload, err := e.ToEventApplicationRepositoryAdd()
		if err != nil {
			return sdk.WrapError(err, "unusedProjectVCSWarning.compute> Unable to get payload from EventApplicationRepositoryAdd")
		}
		if err := removeProjectWarning(db, warn.name(), payload.VCSServer, e.ProjectKey); err != nil {
			return sdk.WrapError(err, "unusedProjectVCSWarning.compute> Unable to remove warning from EventApplicationRepositoryAdd")
		}
	case fmt.Sprintf("%T", sdk.EventApplicationRepositoryDelete{}):
		payload, err := e.ToEventApplicationRepositoryDelete()
		if err != nil {
			return sdk.WrapError(err, "unusedProjectVCSWarning.compute> Unable to get payload from EventApplicationRepositoryDelete")
		}
		apps, err := application.GetNameByVCSServer(db, payload.VCSServer, e.ProjectKey)
		if err != nil {
			return sdk.WrapError(err, "unusedProjectVCSWarning.compute> Cannot list application from EventApplicationRepositoryDelete")
		}
		if len(apps) == 0 {
			w := sdk.WarningV2{
				Key:     e.ProjectKey,
				Element: payload.VCSServer,
				Created: time.Now(),
				Type:    warn.name(),
				MessageParams: map[string]string{
					"VCSName":    payload.VCSServer,
					"ProjectKey": e.ProjectKey,
				},
			}
			if err := Insert(db, w); err != nil {
				return sdk.WrapError(err, "unusedProjectVCSWarning.compute> Unable to insert warning from EventApplicationRepositoryDelete")
			}
		}
	case fmt.Sprintf("%T", sdk.EventProjectVCSServerAdd{}):
		payload, err := e.ToEventProjectVCSServerAdd()
		if err != nil {
			return sdk.WrapError(err, "unusedProjectVCSWarning.compute> Unable to get payload from ToEventProjectVCSServerAdd")
		}

		apps, err := application.GetNameByVCSServer(db, payload.VCSServerName, e.ProjectKey)
		if err != nil {
			return sdk.WrapError(err, "unusedProjectVCSWarning.compute> Cannot list application from EventApplicationRepositoryDelete")
		}
		if len(apps) == 0 {
			w := sdk.WarningV2{
				Key:     e.ProjectKey,
				Element: payload.VCSServerName,
				Created: time.Now(),
				Type:    warn.name(),
				MessageParams: map[string]string{
					"VCSName":    payload.VCSServerName,
					"ProjectKey": e.ProjectKey,
				},
			}
			if err := Insert(db, w); err != nil {
				return sdk.WrapError(err, "unusedProjectVCSWarning.compute> Unable to insert warning from EventApplicationRepositoryDelete")
			}
		}
	case fmt.Sprintf("%T", sdk.EventProjectVCSServerDelete{}):
		payload, err := e.ToEventProjectVCSServerDelete()
		if err != nil {
			return sdk.WrapError(err, "unusedProjectVCSWarning.compute> Unable to get payload from EventProjectKeyDelete")
		}
		if err := removeProjectWarning(db, warn.name(), payload.VCSServerName, e.ProjectKey); err != nil {
			return sdk.WrapError(err, "unusedProjectVCSWarning.compute> Unable to remove warning from EventProjectKeyDelete")
		}
	}
	return nil
}

type missingProjectVCSWarning struct {
	commonWarn
}

func (warn missingProjectVCSWarning) events() []string {
	return []string{
		"sdk.EventProjectVCSServerAdd",
		"sdk.EventProjectVCSServerDelete",
	}
}

func (warn missingProjectVCSWarning) name() string {
	return sdk.WarningMissingProjectVCSServer
}

func (warn missingProjectVCSWarning) compute(db gorp.SqlExecutor, e sdk.Event) error {
	switch e.EventType {
	case fmt.Sprintf("%T", sdk.EventProjectVCSServerAdd{}):
		payload, err := e.ToEventProjectVCSServerAdd()
		if err != nil {
			return sdk.WrapError(err, "missingProjectVCSWarning.compute> Unable to get payload from ToEventProjectVCSServerAdd")
		}
		if err := removeProjectWarning(db, warn.name(), payload.VCSServerName, e.ProjectKey); err != nil {
			return sdk.WrapError(err, "missingProjectVCSWarning.compute> Unable to remove warning")
		}
	case fmt.Sprintf("%T", sdk.EventProjectVCSServerDelete{}):
		payload, err := e.ToEventProjectVCSServerDelete()
		if err != nil {
			return sdk.WrapError(err, "missingProjectVCSWarning.compute> Unable to get payload from EventProjectKeyDelete")
		}
		apps, err := application.GetNameByVCSServer(db, payload.VCSServerName, e.ProjectKey)
		if err != nil {
			return sdk.WrapError(err, "missingProjectVCSWarning.compute>")
		}

		for _, app := range apps {
			w := sdk.WarningV2{
				Key:     e.ProjectKey,
				AppName: app,
				Element: payload.VCSServerName,
				Created: time.Now(),
				Type:    warn.name(),
				MessageParams: map[string]string{
					"VCSName":         payload.VCSServerName,
					"ProjectKey":      e.ProjectKey,
					"ApplicationName": app,
				},
			}
			if err := Insert(db, w); err != nil {
				return sdk.WrapError(err, "missingProjectVCSWarning.compute> Unable to insert warning")
			}
		}

	}
	return nil
}
