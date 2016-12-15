package sdk

// NotifType reprensents a type of notification
type NotifType string

// Type of notifications
const (
	ActionBuildNotif   NotifType = "actionBuild"
	PipelineBuildNotif NotifType = "pipelineBuild"
	BuiltinNotif       NotifType = "builtinNotif"
	UserNotif          NotifType = "userNotif"
)

// NotifEventType reprensents a type of event's notification
type NotifEventType string

// Type of notifications
const (
	UpdateNotifEvent NotifEventType = "update"
	CreateNotifEvent NotifEventType = "create"
)

//UserNotificationSettingsType of notification
type UserNotificationSettingsType string

//const
const (
	EmailUserNotification  UserNotificationSettingsType = "email"
	JabberUserNotification UserNotificationSettingsType = "jabber"
	TATUserNotification    UserNotificationSettingsType = "tat"
)

//UserNotificationEventType always/never/change
type UserNotificationEventType string

//const
const (
	UserNotificationAlways UserNotificationEventType = "always"
	UserNotificationNever  UserNotificationEventType = "never"
	UserNotificationChange UserNotificationEventType = "change"
)

// Notif represents a notification from API
// Event is "create", "update", "delete"
// Status is  "Waiting" "Building" "Success" "Fail" "Unknown", optional
// DateNotif is a date (timestamp format)
type Notif struct {
	ID          int64          `json:"id"`
	Event       NotifEventType `json:"event"`
	DateNotif   int64          `json:"date_notif"`
	NotifType   NotifType      `json:"type_notif"`
	Status      Status         `json:"status,omitempty"`
	Build       *PipelineBuild `json:"pipeline_build,omitempty"`
	ActionBuild *ActionBuild   `json:"action_build,omitempty"`
	Destination string         `json:"destination,omitempty"`
	Recipients  []string       `json:"recipients,omitempty"`
	Title       string         `json:"title,omitempty"`
	Message     string         `json:"message,omitempty"`
}

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
