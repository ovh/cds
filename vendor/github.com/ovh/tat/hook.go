package tat

import (
	"fmt"
)

var HookTat2XMPPHeaderKey = "Tat2xmppkey"

// HookMessageJSON represents a json sent to an external system, for a event about a message
type HookMessageJSON struct {
	Action         string          `json:"action"`
	MessageJSONOut *MessageJSONOut `json:"message"`
}

// HookJSON represents a json sent to an external system
type HookJSON struct {
	Hook        Hook             `json:"hook"`
	HookMessage *HookMessageJSON `json:"hookMessage"`
	Username    string           `json:"username"`
}

var HooksType = []string{HookTypeWebHook, HookTypeKafka, HookTypeXMPP, HookTypeXMPPOut, HookTypeXMPPIn}

var (
	HookTypeWebHook = "tathook-webhook"
	HookTypeKafka   = "tathook-kafka"
	HookTypeXMPP    = "tathook-xmpp"
	HookTypeXMPPOut = "tathook-xmpp-out"
	HookTypeXMPPIn  = "tathook-xmpp-in"
)

type Hook struct {
	ID          string `bson:"_id" json:"_id"`
	Type        string `bson:"type" json:"type"` // in HooksType
	Destination string `bson:"destination" json:"destination"`
	Errors      int    `bson:"errors" json:"errors"`
	Enabled     bool   `bson:"enabled" json:"enabled"`
	Item        string `json:"item"`   // only "message" for now
	Action      string `json:"action"` // MessageActionVoteup, MessageActionCreate, etc...
}

func checkHook(h Hook) error {
	if !ArrayContains(HooksType, h.Type) {
		return fmt.Errorf("Invalid Hook type: %s", h.Type)
	}
	if h.Destination == "" {
		return fmt.Errorf("Invalid hook, destination is empty")
	}
	return nil
}
