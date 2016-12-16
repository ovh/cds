package sdk

// EventType reprensents a type of event
type EventType string

// Type of event
const (
	// UserEvent represent an event that a user want to have
	// a mail, a jabb
	UserEvent   EventType = "userEvent"
	SystemEvent EventType = "systemEvent"
)

// EventAction reprensents a type of event's notification
type EventAction string

// Type of event
const (
	UpdateEvent EventAction = "update"
	CreateEvent EventAction = "create"
)

// Event represents a event from API
// Event is "create", "update", "delete"
// Status is  "Waiting" "Building" "Success" "Fail" "Unknown", optional
// DateEvent is a date (timestamp format)
type Event struct {
	ID            int64          `json:"id"`
	Action        EventAction    `json:"action"` // update, create
	DateEvent     int64          `json:"date_event"`
	EventType     EventType      `json:"type_event"` // userEvent, systemEvent
	Status        Status         `json:"status,omitempty"`
	PipelineBuild *PipelineBuild `json:"pipeline_build,omitempty"`
	ActionBuild   *ActionBuild   `json:"action_build,omitempty"`
	Destination   string         `json:"destination,omitempty"`
	Recipients    []string       `json:"recipients,omitempty"`
	Title         string         `json:"title,omitempty"`
	Message       string         `json:"message,omitempty"`
}
