package sdk

//UserNotificationSettingsType of notification
type UserNotificationSettingsType string

//const
const (
	EmailUserNotification  UserNotificationSettingsType = "email"
	JabberUserNotification UserNotificationSettingsType = "jabber"
)

//UserNotificationEventType always/never/change
type UserNotificationEventType string

//const
const (
	UserNotificationAlways UserNotificationEventType = "always"
	UserNotificationNever  UserNotificationEventType = "never"
	UserNotificationChange UserNotificationEventType = "change"
)

// UserNotification is a settings on application_pipeline/env
// to trigger notification on pipeline event
type UserNotification struct {
	ApplicationPipelineID int64                                                     `json:"application_pipeline_id"`
	Pipeline              Pipeline                                                  `json:"pipeline"`
	Environment           Environment                                               `json:"environment"`
	Notifications         map[UserNotificationSettingsType]UserNotificationSettings `json:"notifications"`
}

// UserNotificationSettings are common settings
type UserNotificationSettings interface {
	Success() UserNotificationEventType
	Failure() UserNotificationEventType
	Start() bool
}

// JabberEmailUserNotificationSettings are jabber or email settings
type JabberEmailUserNotificationSettings struct {
	OnSuccess    UserNotificationEventType `json:"on_success"`
	OnFailure    UserNotificationEventType `json:"on_failure"`
	OnStart      bool                      `json:"on_start"`
	SendToGroups bool                      `json:"send_to_groups"`
	SendToAuthor bool                      `json:"send_to_author"`
	Recipients   []string                  `json:"recipients"`
	Template     UserNotificationTemplate  `json:"template"`
}

//Success returns always/never/change
func (n *JabberEmailUserNotificationSettings) Success() UserNotificationEventType {
	return n.OnSuccess
}

//Failure returns always/never/change
func (n *JabberEmailUserNotificationSettings) Failure() UserNotificationEventType {
	return n.OnFailure
}

//Start returns always/never/change
func (n *JabberEmailUserNotificationSettings) Start() bool {
	return n.OnStart
}

// UserNotificationTemplate is the notification content
type UserNotificationTemplate struct {
	Subject string `json:"subject,omitempty"`
	Body    string `json:"body,omitempty"`
}
