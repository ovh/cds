package sdk

// EventWarningAdd represents the event when adding a warning
type EventWarningAdd struct {
	Warning
}

// EventWarningUpdate represents the event when updating a warning
type EventWarningUpdate struct {
	Warning
}

// EventWarningDelete represents the event when deleting a warning
type EventWarningDelete struct {
	Type    string
	Element string
}
