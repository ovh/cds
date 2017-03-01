package tat

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// StatsCountJSON contains all globals counters
type StatsCountJSON struct {
	Date      int64     `json:"date"`
	DateHuman time.Time `json:"dateHuman"`
	Version   string    `json:"version"`
	Groups    int       `json:"groups"`
	Messages  int       `json:"messages"`
	Presences int       `json:"presences"`
	Topics    int       `json:"topics"`
	Users     int       `json:"users"`
}

// StatsDistributionTopicsJSON is used by GET /distribution/topics
type StatsDistributionTopicsJSON struct {
	Total  int                     `json:"total"`
	Info   string                  `json:"info"`
	Topics []TopicDistributionJSON `json:"topics"`
}

// StatsCount calls GET /stats/count
func (c *Client) StatsCount() (*StatsCountJSON, error) {
	body, err := c.reqWant("GET", 200, "/stats/count", nil)
	if err != nil {
		return nil, err
	}

	out := &StatsCountJSON{}
	if err := json.Unmarshal(body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// StatsDistributionTopics returns Stats Distribution per topics and per users
func (c *Client) StatsDistributionTopics(skip, limit int) (*StatsDistributionTopicsJSON, error) {
	v := url.Values{}
	v.Set("skip", strconv.Itoa(skip))
	v.Set("limit", strconv.Itoa(limit))
	path := fmt.Sprintf("/stats/distribution/topics?%s", v.Encode())
	body, err := c.reqWant("GET", 200, path, nil)
	if err != nil {
		return nil, err
	}

	out := &StatsDistributionTopicsJSON{}
	if err := json.Unmarshal(body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// StatsDBStats returns DB Stats
func (c *Client) StatsDBStats() ([]byte, error) {
	return c.simpleGetAndGetBytes("/stats/db/stats")
}

// StatsDBServerStatus returns DB Server Status
func (c *Client) StatsDBServerStatus() ([]byte, error) {
	return c.simpleGetAndGetBytes("/stats/db/serverStatus")
}

// StatsDBReplSetGetConfig returns DB Relica Set Config
func (c *Client) StatsDBReplSetGetConfig() ([]byte, error) {
	return c.simpleGetAndGetBytes("/stats/db/replSetGetConfig")
}

// StatsDBReplSetGetStatus returns Replica Set Status
func (c *Client) StatsDBReplSetGetStatus() ([]byte, error) {
	return c.simpleGetAndGetBytes("/stats/db/replSetGetStatus")
}

// StatsDBCollections returns nb msg for each collections
func (c *Client) StatsDBCollections() ([]byte, error) {
	return c.simpleGetAndGetBytes("/stats/db/collections")
}

// StatsDBSlowestQueries returns DB slowest Queries
func (c *Client) StatsDBSlowestQueries() ([]byte, error) {
	return c.simpleGetAndGetBytes("/stats/db/slowestQueries")
}

// StatsInstance returns DB Instance
func (c *Client) StatsInstance() ([]byte, error) {
	return c.simpleGetAndGetBytes("/stats/instance")
}
