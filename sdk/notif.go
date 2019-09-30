package sdk

//const
const (
	EmailUserNotification  = "email"
	JabberUserNotification = "jabber"
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

// UserNotificationTemplate is the notification content
type UserNotificationTemplate struct {
	Subject string `json:"subject,omitempty" yaml:"subject,omitempty"`
	Body    string `json:"body,omitempty" yaml:"body,omitempty"`
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
	}
)
