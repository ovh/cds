package tat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// Group struct
type Group struct {
	ID           string   `bson:"_id"          json:"_id"`
	Name         string   `bson:"name"         json:"name"`
	Description  string   `bson:"description"  json:"description"`
	Users        []string `bson:"users"        json:"users,omitempty"`
	AdminUsers   []string `bson:"adminUsers"   json:"adminUsers,omitempty"`
	DateCreation int64    `bson:"dateCreation" json:"dateCreation,omitempty"`
}

// GroupCriteria is used by List all Groups
type GroupCriteria struct {
	Skip            int
	Limit           int
	IDGroup         string
	Name            string
	Description     string
	DateMinCreation string
	DateMaxCreation string
	UserUsername    string
	SortBy          string
}

// CacheKey returns cacke key value
func (g *GroupCriteria) CacheKey() []string {
	var s = []string{}
	if g == nil {
		return s
	}
	if g.Skip != 0 {
		s = append(s, "skip="+strconv.Itoa(g.Skip))
	}
	if g.Limit != 0 {
		s = append(s, "limit="+strconv.Itoa(g.Limit))
	}
	if g.IDGroup != "" {
		s = append(s, "id_group="+g.IDGroup)
	}
	if g.Name != "" {
		s = append(s, "name="+g.Name)
	}
	if g.Description != "" {
		s = append(s, "description="+g.Description)
	}
	if g.DateMinCreation != "" {
		s = append(s, "date_min_creation="+g.DateMinCreation)
	}
	if g.DateMaxCreation != "" {
		s = append(s, "date_max_creation="+g.DateMaxCreation)
	}
	if g.UserUsername != "" {
		s = append(s, "user_username="+g.UserUsername)
	}
	if g.SortBy != "" {
		s = append(s, "sort_by="+g.SortBy)
	}
	return s
}

// GroupsJSON is used by Tat Engine, for groups list
type GroupsJSON struct {
	Count  int     `json:"count"`
	Groups []Group `json:"groups"`
}

// ParamGroupUserJSON is used for add or remove user on a group
type ParamGroupUserJSON struct {
	Groupname string `json:"groupname"`
	Username  string `json:"username"`
}

// GroupJSON contains name and description for a group
type GroupJSON struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
}

// ParamTopicGroupJSON is used for manipulate a group on a topic
type ParamTopicGroupJSON struct {
	Topic     string `json:"topic"`
	Groupname string `json:"groupname"`
	Recursive bool   `json:"recursive"`
}

// GroupList returns groups
func (c *Client) GroupList(skip, limit int) (*GroupsJSON, error) {
	path := fmt.Sprintf("/groups?skip=%d&limit=%d", skip, limit)
	out, err := c.reqWant("GET", http.StatusOK, path, nil)
	if err != nil {
		ErrorLogFunc("Error while listing groups: %s", err)
		return nil, err
	}
	groups := &GroupsJSON{}
	if err := json.Unmarshal(out, groups); err != nil {
		return nil, err
	}
	return groups, nil
}

//GroupCreate creates a group
func (c *Client) GroupCreate(g GroupJSON) (*Group, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	b, err := json.Marshal(g)
	if err != nil {
		ErrorLogFunc("Error while marshal group: %s", err)
		return nil, err
	}

	res, err := c.reqWant(http.MethodPost, http.StatusCreated, "/group", b)
	if err != nil {
		ErrorLogFunc("Error while marshal group for GroupCreate: %s", err)
		return nil, err
	}

	DebugLogFunc("GroupCreate : %s", string(res))

	newGroup := &Group{}
	if err := json.Unmarshal(res, newGroup); err != nil {
		return nil, err
	}

	return newGroup, nil
}

// GroupUpdate updates a group
func (c *Client) GroupUpdate(groupname, newGroupname, newDescription string) error {

	m := GroupJSON{Name: newGroupname, Description: newDescription}
	jsonStr, err := json.Marshal(m)
	if err != nil {
		return err
	}

	_, err = c.reqWant("PUT", http.StatusOK, "/group/edit/"+groupname, jsonStr)
	if err != nil {
		ErrorLogFunc("Error while updating group: %s", err)
		return err
	}
	return nil
}

//GroupDelete delete a group
func (c *Client) GroupDelete(groupname string) error {
	_, err := c.reqWant(http.MethodDelete, http.StatusOK, "/group/edit/"+groupname, nil)
	if err != nil {
		ErrorLogFunc("Error while deleting group: %s", err)
		return err
	}
	return nil
}

// GroupAddUsers adds users on a group
func (c *Client) GroupAddUsers(groupname string, users []string) error {
	return c.groupAddRemoveUsers("PUT", "/group/add/user", groupname, users)
}

// GroupDeleteUsers deletes users from a group
func (c *Client) GroupDeleteUsers(groupname string, users []string) error {
	return c.groupAddRemoveUsers("PUT", "/group/remove/user", groupname, users)
}

// GroupAddAdminUsers adds an admin user on a group
func (c *Client) GroupAddAdminUsers(groupname string, users []string) error {
	return c.groupAddRemoveUsers("PUT", "/group/add/adminuser", groupname, users)
}

// GroupDeleteAdminUsers removes admin users from a group
func (c *Client) GroupDeleteAdminUsers(groupname string, users []string) error {
	return c.groupAddRemoveUsers("PUT", "/group/remove/adminuser", groupname, users)
}

func (c *Client) groupAddRemoveUsers(method, path, groupname string, users []string) error {
	for _, username := range users {
		t := ParamGroupUserJSON{Groupname: groupname, Username: username}

		b, err := json.Marshal(t)
		if err != nil {
			ErrorLogFunc("Error while marshal group: %s", err)
			return err
		}

		_, err = c.reqWant(method, http.StatusCreated, path, b)
		if err != nil {
			ErrorLogFunc("Error while deleting group: %s", err)
			return err
		}
		return nil
	}
	return nil
}
