package sdk

// EventWarningAdd represents the event when adding a warning
//easyjson:json
type EventWarningAdd struct {
	Warning
}

// EventWarningUpdate represents the event when updating a warning
//easyjson:json
type EventWarningUpdate struct {
	Warning
}

// EventWarningDelete represents the event when deleting a warning
//easyjson:json
type EventWarningDelete struct {
	Type    string
	Element string
}
