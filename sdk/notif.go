package sdk

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"text/template"
	"time"

	"github.com/ovh/venom"

	"github.com/ovh/cds/sdk/interpolate"
)

//const
const (
	EmailUserNotification  = "email"
	JabberUserNotification = "jabber"
	VCSUserNotification    = "vcs"
	EventsNotification     = "event"
)

//const
const (
	UserNotificationAlways = "always"
	UserNotificationNever  = "never"
	UserNotificationChange = "change"
)

// UserNotification is a settings on application_pipeline/env
// to trigger notification on pipeline event
type UserNotification struct {
	ApplicationPipelineID int64                               `json:"application_pipeline_id"`
	Pipeline              Pipeline                            `json:"pipeline"`
	Environment           Environment                         `json:"environment"`
	Notifications         map[string]UserNotificationSettings `json:"notifications"`
}

// UserNotificationSettings are jabber or email settings
type UserNotificationSettings struct {
	OnSuccess    string                    `json:"on_success,omitempty" yaml:"on_success,omitempty"`         // default is "onChange", empty means onChange
	OnFailure    string                    `json:"on_failure,omitempty" yaml:"on_failure,omitempty"`         // default is "always", empty means always
	OnStart      *bool                     `json:"on_start,omitempty" yaml:"on_start,omitempty"`             // default is false, nil is false
	SendToGroups *bool                     `json:"send_to_groups,omitempty" yaml:"send_to_groups,omitempty"` // default is false, nil is false
	SendToAuthor *bool                     `json:"send_to_author,omitempty" yaml:"send_to_author,omitempty"` // default is true, nil is true
	Recipients   []string                  `json:"recipients,omitempty" yaml:"recipients,omitempty"`
	Template     *UserNotificationTemplate `json:"template,omitempty" yaml:"template,omitempty"`
	Conditions   WorkflowNodeConditions    `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// Value returns driver.Value from Metadata.
func (a UserNotificationSettings) Value() (driver.Value, error) {
	j, err := json.Marshal(a)
	return j, WrapError(err, "cannot marshal UserNotificationSettings")
}

// Scan UserNotificationSettings.
func (a *UserNotificationSettings) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(json.Unmarshal(source, a), "cannot unmarshal UserNotificationSettings")
}

// UserNotificationTemplate is the notification content
type UserNotificationTemplate struct {
	Subject string `json:"subject,omitempty" yaml:"subject,omitempty"`
	Body    string `json:"body,omitempty" yaml:"body,omitempty"`

	// For VCS
	DisableComment *bool `json:"disable_comment,omitempty" yaml:"disable_comment,omitempty"`
	DisableStatus  *bool `json:"disable_status,omitempty" yaml:"disable_status,omitempty"`
}

//userNotificationInput is a way to parse notification
type userNotificationInput struct {
	Notifications         map[string]interface{} `json:"notifications"`
	ApplicationPipelineID int64                  `json:"application_pipeline_id"`
	Environment           Environment            `json:"environment"`
	Pipeline              Pipeline               `json:"pipeline"`
}

//Default template values
var (
	UserNotificationTemplateEmail = UserNotificationTemplate{
		Subject: "{{.cds.project}}/{{.cds.workflow}}#{{.cds.version}} {{.cds.status}}",
		Body: `Project : {{.cds.project}}
Workflow : {{.cds.workflow}}#{{.cds.version}}
Pipeline : {{.cds.node}}
Status : {{.cds.status}}
Details : {{.cds.buildURL}}
Triggered by : {{.cds.triggered_by.username}}
Branch : {{.git.branch | default "n/a"}}`,
	}

	UserNotificationTemplateJabber = UserNotificationTemplate{
		Subject: "{{.cds.project}}/{{.cds.workflow}}#{{.cds.version}} {{.cds.status}}",
		Body:    `{{.cds.buildURL}}`,
	}

	UserNotificationTemplateMap = map[string]UserNotificationTemplate{
		EmailUserNotification:  UserNotificationTemplateEmail,
		JabberUserNotification: UserNotificationTemplateJabber,
		VCSUserNotification: {
			Body: DefaultWorkflowNodeRunReport,
		},
	}
)

const DefaultWorkflowNodeRunReport = `[[- if .Stages ]]
CDS Report [[.WorkflowNodeName]]#[[.Number]].[[.SubNumber]] [[ if eq .Status "Success" -]] ✔ [[ else ]][[ if eq .Status "Fail" -]] ✘ [[ else ]][[ if eq .Status "Stopped" -]] ■ [[ else ]]- [[ end ]] [[ end ]] [[ end ]]
[[- range $s := .Stages]]
[[- if $s.RunJobs ]]
* [[$s.Name]]
[[- range $j := $s.RunJobs]]
  * [[$j.Job.Action.Name]] [[ if eq $j.Status "Success" -]] ✔ [[ else ]][[ if eq $j.Status "Fail" -]] ✘ [[ else ]][[ if eq $j.Status "Stopped" -]] ■ [[ else ]]- [[ end ]] [[ end ]] [[ end ]]
[[- end]]
[[- end]]
[[- end]]
[[- end]]

[[- if .Tests ]]
[[- if gt .Tests.TotalKO 0]]
Unit Tests Report

[[- range $ts := .Tests.TestSuites]]
* [[ $ts.Name ]]
[[range $tc := $ts.TestCases]]
  [[- if or ($tc.Errors) ($tc.Failures) ]]  * [[ $tc.Name ]] ✘ [[- end]]
[[end]]
[[- end]]
[[- end]]
[[- end]]
`

func (nr WorkflowNodeRun) Report() (string, error) {
	reportStr := DefaultWorkflowNodeRunReport
	if nr.VCSReport != "" {
		reportStr = nr.VCSReport
	}

	tmpl, err := template.New("vcsreport").Delims("[[", "]]").Funcs(interpolate.InterpolateHelperFuncs).Parse(reportStr)
	if err != nil {
		return "", WrapError(err, "cannot create new template for first part")
	}

	nrData := struct {
		WorkflowNodeName string
		Status           string
		Number           int64
		SubNumber        int64
		Stages           []Stage
		Start            time.Time
		Done             time.Time
		Tests            *venom.Tests
	}{
		WorkflowNodeName: nr.WorkflowNodeName,
		Status:           nr.Status,
		Number:           nr.Number,
		SubNumber:        nr.SubNumber,
		Stages:           nr.Stages,
		Start:            nr.Start,
		Done:             nr.Done,
		Tests:            nr.Tests,
	}

	outFirst := new(bytes.Buffer)
	if err := tmpl.Execute(outFirst, nrData); err != nil {
		return "", WrapError(err, "cannot execute template for first part")
	}

	return interpolate.Do(outFirst.String(), ParametersToMap(nr.BuildParameters))
}
