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
	OnSuccess    string                   `json:"on_success,omitempty" yaml:"on_success,omitempty"`
	OnFailure    string                   `json:"on_failure,omitempty" yaml:"on_failure,omitempty"`
	OnStart      bool                     `json:"on_start,omitempty" yaml:"on_start,omitempty"`
	SendToGroups bool                     `json:"send_to_groups,omitempty" yaml:"send_to_groups,omitempty"`
	SendToAuthor bool                     `json:"send_to_author,omitempty" yaml:"send_to_author,omitempty"`
	Recipients   []string                 `json:"recipients,omitempty" yaml:"recipients,omitempty"`
	Template     UserNotificationTemplate `json:"template,omitempty" yaml:"template,omitempty"`
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
