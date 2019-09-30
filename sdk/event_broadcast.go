package sdk

// EventBroadcastAdd represents the event when adding a broadcast
//easyjson:json
type EventBroadcastAdd struct {
	Broadcast
}

// EventBroadcastUpdate represents the event when updating a broadcast
//easyjson:json
type EventBroadcastUpdate struct {
	OldBroadcast Broadcast
	NewBroadcast Broadcast
}

// EventBroadcastDelete represents the event when deleting a broadcast
//easyjson:json
type EventBroadcastDelete struct {
	BroadcastID int64
}
