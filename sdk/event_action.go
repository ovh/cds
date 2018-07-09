package sdk

// EventActionAdd represents the event when adding an action
type EventActionAdd struct {
	Action
}

// EventActionUpdate represents the event when updating an action
type EventActionUpdate struct {
	OldAction Action
	NewAction Action
}

// EventActionDelete represents the event when deleting an action
type EventActionDelete struct {
	Action
}
