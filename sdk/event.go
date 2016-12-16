package sdk

// EventSource reprensents a type of event
type EventSource string

// Type of event
const (
	// UserEvent represent an event that a user want to have
	// a mail, a jabb
	UserEvent   EventSource = "userEvent"
	SystemEvent EventSource = "systemEvent"
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
	ID          int64       `json:"id"`
	Action      EventAction `json:"action"` // update, create
	DateEvent   int64       `json:"date_event"`
	EventSource EventSource `json:"source_event"` // userEvent, systemEvent
	EventType   string      `json:"type_event"`   // type of payload
	Payload     []byte      `json:"payload"`
	Destination string      `json:"destination,omitempty"`
	Recipients  []string    `json:"recipients,omitempty"`
}
