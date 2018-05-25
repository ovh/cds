package sdk

// EventWarningAdd represents the event when adding a warning
type EventWarningAdd struct {
	WarningV2
}

// EventWarningDelete represents the event when deleting a warning
type EventWarningDelete struct {
	Type    string
	Element string
}
