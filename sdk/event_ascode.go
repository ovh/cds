package sdk

// EventAsCodeEvent represents the event when add/update a workflow event
//easyjson:json
type EventAsCodeEvent struct {
	Event AsCodeEvent `json:"as_code_event"`
}
