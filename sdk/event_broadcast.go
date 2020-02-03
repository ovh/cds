package sdk

// EventBroadcastAdd represents the event when adding a broadcast
type EventBroadcastAdd struct {
	Broadcast
}

// EventBroadcastUpdate represents the event when updating a broadcast
type EventBroadcastUpdate struct {
	OldBroadcast Broadcast
	NewBroadcast Broadcast
}

// EventBroadcastDelete represents the event when deleting a broadcast
type EventBroadcastDelete struct {
	BroadcastID int64
}
