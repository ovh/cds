package sdk

// EventActionAdd represents the event when adding an action.
//easyjson:json
type EventActionAdd struct {
	Action Action
}

// EventActionUpdate represents the event when updating an action.
//easyjson:json
type EventActionUpdate struct {
	OldAction Action
	NewAction Action
}
