package tat

import (
	"encoding/json"
	"fmt"
	"time"
)

// UserPresence struct
type UserPresence struct {
	Username string `bson:"username" json:"username"`
	Fullname string `bson:"fullname" json:"fullname"`
}

// Presence struct
type Presence struct {
	ID               string       `bson:"_id,omitempty"    json:"_id"`
	Status           string       `bson:"status"           json:"status"`
	Topic            string       `bson:"topic"            json:"topic"`
	DatePresence     int64        `bson:"datePresence"     json:"datePresence"`
	DateTimePresence time.Time    `bson:"dateTimePresence" json:"dateTimePresence"`
	UserPresence     UserPresence `bson:"userPresence"     json:"userPresence"`
}

// PresenceCriteria used by Presences List
type PresenceCriteria struct {
	Skip            int
	Limit           int
	IDPresence      string
	Status          string
	Topic           string
	Username        string
	DateMinPresence string
	DateMaxPresence string
	SortBy		string
}

// PresencesJSON represents list of presences with count for total
type PresencesJSON struct {
	Count     int        `json:"count"`
	Presences []Presence `json:"presences"`
}

// PresenceJSONOut represents a presence
type PresenceJSONOut struct {
	Presence Presence `json:"presence"`
}

// PresenceJSON represents a status on a topic
type PresenceJSON struct {
	Status   string `json:"status" binding:"required"`
	Username string `json:"username,omitempty"`
	Topic    string
}

// PresenceAddAndGet adds a new presence and get presences on topic
func (c *Client) PresenceAddAndGet(topic, status string) (*PresencesJSON, error) {
	p := PresenceJSON{Status: status}
	jsonStr, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	body, err := c.reqWant("POST", 200, "/presenceget"+topic, jsonStr)
	if err != nil {
		return nil, err
	}

	out := &PresencesJSON{}
	if err := json.Unmarshal(body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// PresenceDelete deletes a presence
func (c *Client) PresenceDelete(topic, username string) error {
	p := PresenceJSON{Username: username}
	jsonStr, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = c.reqWant("DELETE", 200, "/presences"+topic, jsonStr)
	return err
}

// PresenceList returns presences on topic
func (c *Client) PresenceList(topic string, skip, limit int) (*PresencesJSON, error) {
	path := fmt.Sprintf("/presences%s?skip=%d&limit=%d", topic, skip, limit)
	body, err := c.reqWant("GET", 200, path, nil)
	if err != nil {
		return nil, err
	}

	out := &PresencesJSON{}
	if err := json.Unmarshal(body, out); err != nil {
		return nil, err
	}
	return out, nil
}
