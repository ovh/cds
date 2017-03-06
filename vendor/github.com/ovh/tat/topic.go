package tat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Topic struct
type Topic struct {
	ID                   string           `bson:"_id" json:"_id,omitempty"`
	Collection           string           `bson:"collection" json:"collection"`
	Topic                string           `bson:"topic" json:"topic"`
	Description          string           `bson:"description" json:"description"`
	ROGroups             []string         `bson:"roGroups" json:"roGroups,omitempty"`
	RWGroups             []string         `bson:"rwGroups" json:"rwGroups,omitempty"`
	ROUsers              []string         `bson:"roUsers" json:"roUsers,omitempty"`
	RWUsers              []string         `bson:"rwUsers" json:"rwUsers,omitempty"`
	AdminUsers           []string         `bson:"adminUsers" json:"adminUsers,omitempty"`
	AdminGroups          []string         `bson:"adminGroups" json:"adminGroups,omitempty"`
	History              []string         `bson:"history" json:"history"`
	MaxLength            int              `bson:"maxlength" json:"maxlength"`
	MaxReplies           int              `bson:"maxreplies" json:"maxreplies"`
	CanForceDate         bool             `bson:"canForceDate" json:"canForceDate"`
	CanUpdateMsg         bool             `bson:"canUpdateMsg" json:"canUpdateMsg"`
	CanDeleteMsg         bool             `bson:"canDeleteMsg" json:"canDeleteMsg"`
	CanUpdateAllMsg      bool             `bson:"canUpdateAllMsg" json:"canUpdateAllMsg"`
	CanDeleteAllMsg      bool             `bson:"canDeleteAllMsg" json:"canDeleteAllMsg"`
	AdminCanUpdateAllMsg bool             `bson:"adminCanUpdateAllMsg" json:"adminCanUpdateAllMsg"`
	AdminCanDeleteAllMsg bool             `bson:"adminCanDeleteAllMsg" json:"adminCanDeleteAllMsg"`
	IsAutoComputeTags    bool             `bson:"isAutoComputeTags" json:"isAutoComputeTags"`
	IsAutoComputeLabels  bool             `bson:"isAutoComputeLabels" json:"isAutoComputeLabels"`
	DateModification     int64            `bson:"dateModification" json:"dateModificationn,omitempty"`
	DateCreation         int64            `bson:"dateCreation" json:"dateCreation,omitempty"`
	DateLastMessage      int64            `bson:"dateLastMessage" json:"dateLastMessage,omitempty"`
	Parameters           []TopicParameter `bson:"parameters" json:"parameters,omitempty"`
	Tags                 []string         `bson:"tags" json:"tags,omitempty"`
	Labels               []Label          `bson:"labels" json:"labels,omitempty"`
	Filters              []Filter         `bson:"filters" json:"filters"`
}

type Filter struct {
	Topic    string         `bson:"-" json:"topic"`
	ID       string         `bson:"_id" json:"_id"`
	UserID   string         `bson:"userID" json:"userID"`
	Username string         `bson:"username" json:"username"`
	Title    string         `bson:"title" json:"title"`
	Criteria FilterCriteria `bson:"criteria" json:"criteria"`
	Hooks    []Hook         `bson:"hooks" json:"hooks"`
}

// FilterCriteria are used to list messages
type FilterCriteria struct {
	Label       string `bson:"label" json:"label,omitempty"`
	NotLabel    string `bson:"notLabel" json:"notLabel,omitempty"`
	AndLabel    string `bson:"andLabel" json:"andLabel,omitempty"`
	Tag         string `bson:"tag" json:"tag,omitempty"`
	NotTag      string `bson:"notTag" json:"notTag,omitempty"`
	AndTag      string `bson:"andTag" json:"andTag,omitempty"`
	Username    string `bson:"username" json:"username,omitempty"`
	OnlyMsgRoot bool   `bson:"onlyMsgRoot" json:"onlyMsgRoot"`
}

func (c FilterCriteria) FilterCriteriaIsEmpty() bool {

	if c.Label != "" ||
		c.NotLabel != "" ||
		c.AndLabel != "" ||
		c.Tag != "" ||
		c.NotTag != "" ||
		c.AndTag != "" ||
		c.Username != "" ||
		c.OnlyMsgRoot == true {
		return false
	}
	return true
}

// TopicParameter struct, parameter on topics
type TopicParameter struct {
	Key   string `bson:"key"   json:"key"`
	Value string `bson:"value" json:"value"`
}

// TopicCriteria struct, used by List Topic
type TopicCriteria struct {
	Skip                 int
	Limit                int
	IDTopic              string
	Topic                string
	TopicPath            string
	Description          string
	DateMinCreation      string
	DateMaxCreation      string
	GetNbMsgUnread       string
	OnlyFavorites        string
	GetForTatAdmin       string
	GetForAllTasksTopics bool
	Group                string
	SortBy               string
}

// CacheKey returns cache key value
func (t *TopicCriteria) CacheKey() []string {
	var s = []string{}
	if t == nil {
		return s
	}
	if t.Skip != 0 {
		s = append(s, "skip="+strconv.Itoa(t.Skip))
	}
	if t.Limit != 0 {
		s = append(s, "limit="+strconv.Itoa(t.Limit))
	}
	if t.IDTopic != "" {
		s = append(s, "id_topic="+t.IDTopic)
	}
	if t.Topic != "" {
		s = append(s, "topic="+t.Topic)
	}
	if t.TopicPath != "" {
		s = append(s, "topic_path="+t.TopicPath)
	}
	if t.Description != "" {
		s = append(s, "description="+t.Description)
	}
	if t.DateMinCreation != "" {
		s = append(s, "date_min_creation="+t.DateMinCreation)
	}
	if t.DateMaxCreation != "" {
		s = append(s, "date_max_creation="+t.DateMaxCreation)
	}
	if t.GetNbMsgUnread != "" {
		s = append(s, "get_nb_msg_unread="+t.GetNbMsgUnread)
	}
	if t.OnlyFavorites != "" {
		s = append(s, "only_favorites="+t.OnlyFavorites)
	}
	if t.GetForTatAdmin != "" {
		s = append(s, "get_for_tat_admin="+t.GetForTatAdmin)
	}
	if t.GetForAllTasksTopics {
		s = append(s, "get_for_all_tasks_topics="+strconv.FormatBool(t.GetForAllTasksTopics))
	}
	if t.Group != "" {
		s = append(s, "group="+t.Group)
	}
	if t.SortBy != "" {
		s = append(s, "sort_by="+t.SortBy)
	}
	return s
}

// ParamTopicUserJSON is used to update a parameter on topic
type ParamTopicUserJSON struct {
	Topic     string `json:"topic"` // topic topic
	Username  string `json:"username"`
	Recursive bool   `json:"recursive"`
}

// TopicCreateJSON is used to create a parameter on topic
type TopicCreateJSON struct {
	Topic       string `json:"topic" binding:"required"`
	Description string `json:"description" binding:"required"`
}

// TopicParameterJSON is used to manipulate a parameter on a topic
type TopicParameterJSON struct {
	Topic     string `json:"topic"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Recursive bool   `json:"recursive"`
}

// TopicsJSON represents struct used by Engine while returns list of topics
type TopicsJSON struct {
	Count                int            `json:"count"`
	Topics               []Topic        `json:"topics"`
	CountTopicsMsgUnread int            `json:"countTopicsMsgUnread"`
	TopicsMsgUnread      map[string]int `json:"topicsMsgUnread"`
}

// TopicJSON represents struct used by Engine while returns one topic
type TopicJSON struct {
	Topic        *Topic `json:"topic"`
	IsTopicRw    bool   `json:"isTopicRw"`
	IsTopicAdmin bool   `json:"isTopicAdmin"`
}

// TopicDistributionJSON represents struct used by Engine while returns topic distribution
type TopicDistributionJSON struct {
	ID         string `json:"id"`
	Topic      string `json:"topic"`
	Count      int    `json:"count"`
	Dedicated  bool   `json:"dedicated"`
	Collection string `json:"collection"`
}

// TopicNameJSON represents struct, only topic name
type TopicNameJSON struct {
	Topic string `json:"topic"`
}

// ParamJSON is used to update a param on a topic (attr. Parameters on Topic struct)
type ParamJSON struct {
	ParamName  string `json:"paramName"`
	ParamValue string `json:"paramValue"`
}

// CheckAndFixNameTopic Add a / to topic name is it is not present
// return an error if length of name is < 4 or > 100
func CheckAndFixNameTopic(topicName string) (string, error) {
	name := strings.TrimSpace(topicName)

	if len(name) > 1 && string(name[0]) != "/" {
		name = "/" + name
	}

	if len(name) < 4 {
		return topicName, fmt.Errorf("Invalid topic length (3 or more characters, beginning with slash. Ex: /ABC): %s", topicName)
	}

	if len(name)-1 == strings.LastIndex(name, "/") {
		name = name[0 : len(name)-1]
	}

	if len(name) > 100 {
		return topicName, fmt.Errorf("Invalid topic length (max 100 characters):%s", topicName)
	}

	return name, nil
}

// TopicCreate creates a topic
func (c *Client) TopicCreate(t TopicCreateJSON) (*Topic, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	b, err := json.Marshal(t)
	if err != nil {
		ErrorLogFunc("Error while marshal topic: %s", err)
		return nil, err
	}

	res, err := c.reqWant("POST", http.StatusCreated, "/topic", b)
	if err != nil {
		ErrorLogFunc("Error while marshal message for CreateTopic: %s", err)
		return nil, err
	}

	DebugLogFunc("createTopicResponse : %s", string(res))

	newTopic := &Topic{}
	if err := json.Unmarshal(res, newTopic); err != nil {
		return nil, err
	}

	return newTopic, nil
}

// TopicOne returns one topic, and flags isUserRW / isUserAdmin on topic
func (c *Client) TopicOne(topic string) (*TopicJSON, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	path := fmt.Sprintf("/topic%s", topic)

	body, err := c.reqWant(http.MethodGet, 200, path, nil)
	if err != nil {
		ErrorLogFunc("Error getting one topic: %s", err)
		return nil, err
	}

	DebugLogFunc("One Topic Response: %s", string(body))
	var out = TopicJSON{}
	if err := json.Unmarshal(body, &out); err != nil {
		ErrorLogFunc("Error getting one topic: %s", err)
		return nil, err
	}

	return &out, nil
}

// TopicList list all topics according to criterias. Default behavior (criteria is Nil) will limit 10 topics.
func (c *Client) TopicList(criteria *TopicCriteria) (*TopicsJSON, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	if criteria == nil {
		criteria = &TopicCriteria{
			Skip:  0,
			Limit: 100,
		}
	}

	v := url.Values{}
	v.Set("skip", strconv.Itoa(criteria.Skip))
	v.Set("limit", strconv.Itoa(criteria.Limit))

	if criteria.Topic != "" {
		v.Set("topic", criteria.Topic)
	}
	if criteria.TopicPath != "" {
		v.Set("topicPath", criteria.TopicPath)
	}
	if criteria.IDTopic != "" {
		v.Set("idTopic", criteria.IDTopic)
	}
	if criteria.Description != "" {
		v.Set("Description", criteria.Description)
	}
	if criteria.DateMinCreation != "" {
		v.Set("DateMinCreation", criteria.DateMinCreation)
	}
	if criteria.DateMaxCreation != "" {
		v.Set("DateMaxCreation", criteria.DateMaxCreation)
	}
	if criteria.GetNbMsgUnread != "" {
		v.Set("getNbMsgUnread", criteria.GetNbMsgUnread)
	}
	if criteria.OnlyFavorites != "" {
		v.Set("onlyFavorites", criteria.OnlyFavorites)
	}
	if criteria.GetForTatAdmin == "true" {
		v.Set("getForTatAdmin", criteria.GetForTatAdmin)
	}

	path := fmt.Sprintf("/topics?%s", v.Encode())

	body, err := c.reqWant(http.MethodGet, 200, path, nil)
	if err != nil {
		ErrorLogFunc("Error getting topic list: %s", err)
		return nil, err
	}

	DebugLogFunc("Topic List Response: %s", string(body))
	var topics = TopicsJSON{}
	if err := json.Unmarshal(body, &topics); err != nil {
		ErrorLogFunc("Error getting topic list: %s", err)
		return nil, err
	}

	return &topics, nil
}

// TopicDelete delete a topics
func (c *Client) TopicDelete(t TopicNameJSON) ([]byte, error) {
	out, err := c.reqWant(http.MethodDelete, 200, "/topic"+t.Topic, nil)
	if err != nil {
		ErrorLogFunc("Error deleting topic list: %s", err)
		return nil, err
	}
	return out, nil
}

// TopicTruncate deletes all messages in a topic
func (c *Client) TopicTruncate(t TopicNameJSON) ([]byte, error) {
	return c.simplePutAndGetBytes("/topic/truncate", 201, t)
}

// TopicComputeLabels computes labels on a topic
func (c *Client) TopicComputeLabels(t TopicNameJSON) ([]byte, error) {
	return c.simplePutAndGetBytes("/topic/compute/labels", 201, t)
}

// TopicTruncateLabels removes all labels computed on topic
func (c *Client) TopicTruncateLabels(t TopicNameJSON) ([]byte, error) {
	return c.simplePutAndGetBytes("/topic/truncate/labels", 200, t)
}

// TopicComputeTags computes tags on a topic
func (c *Client) TopicComputeTags(t TopicNameJSON) ([]byte, error) {
	return c.simplePutAndGetBytes("/topic/compute/tags", 201, t)
}

// TopicTruncateTags removes all tags computed on topic
func (c *Client) TopicTruncateTags(t TopicNameJSON) ([]byte, error) {
	return c.simplePutAndGetBytes("/topic/truncate/tags", 200, t)
}

// TopicAllComputeLabels computes labels on all topics
func (c *Client) TopicAllComputeLabels() ([]byte, error) {
	return c.simplePutAndGetBytes("/topics/compute/labels", 201, nil)
}

// TopicAllComputeTags computes tags on all topics
func (c *Client) TopicAllComputeTags() ([]byte, error) {
	return c.simplePutAndGetBytes("/topics/compute/tags", 201, nil)
}

// TopicAllComputeReplies computes replies on all topics
func (c *Client) TopicAllComputeReplies() ([]byte, error) {
	return c.simplePutAndGetBytes("/topics/compute/replies", 200, nil)
}

// TopicAllSetParam sets a param on all topics
func (c *Client) TopicAllSetParam(p ParamJSON) ([]byte, error) {
	jsonStr, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	return c.reqWant("PUT", http.StatusOK, "/topics/param", jsonStr)
}

// TopicAddRoUsers adds a read-only user on a topic
func (c *Client) TopicAddRoUsers(topic string, users []string, recursive bool) error {
	return c.topicActionOnUsers("/topic/add/rouser", 201, topic, users, recursive)
}

// TopicAddRwUsers adds a read-write user on a topic
func (c *Client) TopicAddRwUsers(topic string, users []string, recursive bool) error {
	return c.topicActionOnUsers("/topic/add/rwuser", 201, topic, users, recursive)
}

// TopicAddAdminUsers adds admin users on a topic
func (c *Client) TopicAddAdminUsers(topic string, users []string, recursive bool) error {
	return c.topicActionOnUsers("/topic/add/adminuser", 201, topic, users, recursive)
}

// TopicDeleteRoUsers deletes a read-only user on a topic
func (c *Client) TopicDeleteRoUsers(topic string, users []string, recursive bool) error {
	return c.topicActionOnUsers("/topic/remove/rouser", 200, topic, users, recursive)
}

// TopicDeleteRwUsers deletes some read-write users on a topic
func (c *Client) TopicDeleteRwUsers(topic string, users []string, recursive bool) error {
	return c.topicActionOnUsers("/topic/remove/rwuser", 200, topic, users, recursive)
}

// TopicDeleteAdminUsers deletes some admin users on a topic
func (c *Client) TopicDeleteAdminUsers(topic string, users []string, recursive bool) error {
	return c.topicActionOnUsers("/topic/remove/adminuser", 200, topic, users, recursive)
}

func (c *Client) topicActionOnUsers(url string, want int, topic string, users []string, recursive bool) error {
	for _, username := range users {
		if _, err := c.topicActionOnUser(url, want, topic, username, recursive); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) topicActionOnUser(url string, want int, topic, username string, recursive bool) ([]byte, error) {
	t := ParamTopicUserJSON{
		Topic:     topic,
		Username:  username,
		Recursive: recursive,
	}
	out, err := c.simplePutAndGetBytes(url, want, t)
	if err != nil {
		ErrorLogFunc("Error removing on url: %s", url, err)
		return nil, err
	}
	return out, nil
}

// TopicAddRoGroups adds a read-only group on a topic
func (c *Client) TopicAddRoGroups(topic string, groups []string, recursive bool) error {
	return c.topicActionOnGroups("/topic/add/rogroup", 201, topic, groups, recursive)
}

// TopicAddRwGroups adds a read-write group on a topic
func (c *Client) TopicAddRwGroups(topic string, groups []string, recursive bool) error {
	return c.topicActionOnGroups("/topic/add/rwgroup", 201, topic, groups, recursive)
}

// TopicAddAdminGroups adds admin groups on a topic
func (c *Client) TopicAddAdminGroups(topic string, groups []string, recursive bool) error {
	return c.topicActionOnGroups("/topic/add/admingroup", 201, topic, groups, recursive)
}

// TopicDeleteRoGroups deletes a read-only group on a topic
func (c *Client) TopicDeleteRoGroups(topic string, groups []string, recursive bool) error {
	return c.topicActionOnGroups("/topic/remove/rogroup", 200, topic, groups, recursive)
}

// TopicDeleteRwGroups deletes some read-write groups on a topic
func (c *Client) TopicDeleteRwGroups(topic string, groups []string, recursive bool) error {
	return c.topicActionOnGroups("/topic/remove/rwgroup", 200, topic, groups, recursive)
}

// TopicDeleteAdminGroups deletes some admin groups on a topic
func (c *Client) TopicDeleteAdminGroups(topic string, groups []string, recursive bool) error {
	return c.topicActionOnGroups("/topic/remove/admingroup", 200, topic, groups, recursive)
}

func (c *Client) topicActionOnGroups(url string, want int, topic string, groups []string, recursive bool) error {
	for _, groupname := range groups {
		if _, err := c.topicActionOnGroup(url, want, topic, groupname, recursive); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) topicActionOnGroup(url string, want int, topic, groupname string, recursive bool) ([]byte, error) {
	t := ParamTopicGroupJSON{
		Topic:     topic,
		Groupname: groupname,
		Recursive: recursive,
	}
	out, err := c.simplePutAndGetBytes(url, want, t)
	if err != nil {
		ErrorLogFunc("Error removing on url: %s", url, err)
		return nil, err
	}
	return out, nil
}

// TopicAddParameter adds a parameter on a topic
func (c *Client) TopicAddParameter(topic, key, value string, recursive bool) ([]byte, error) {
	t := TopicParameterJSON{
		Topic:     topic,
		Key:       key,
		Value:     value,
		Recursive: recursive,
	}
	out, err := c.simplePutAndGetBytes("/topic/add/parameter", 201, t)
	if err != nil {
		ErrorLogFunc("Error removing a parameter: %s", err)
		return nil, err
	}
	return out, nil
}

// TopicDeleteParameters removes a parameter on a topic
func (c *Client) TopicDeleteParameters(topic string, params []string, recursive bool) error {
	for _, key := range params {
		t := TopicParameterJSON{
			Topic:     topic,
			Key:       key,
			Recursive: recursive,
		}
		_, err := c.simplePutAndGetBytes("/topic/remove/parameter", 201, t)
		if err != nil {
			ErrorLogFunc("Error removing a parameter: %s", err)
			return err
		}
	}
	return nil
}

// TopicParameters updates param on one topic
type TopicParameters struct {
	Topic                string `json:"topic"`
	MaxLength            int    `json:"maxlength"`
	MaxReplies           int    `json:"maxreplies"`
	CanForceDate         bool   `json:"canForceDate"`
	CanUpdateMsg         bool   `json:"canUpdateMsg"`
	CanDeleteMsg         bool   `json:"canDeleteMsg"`
	CanUpdateAllMsg      bool   `json:"canUpdateAllMsg"`
	CanDeleteAllMsg      bool   `json:"canDeleteAllMsg"`
	AdminCanUpdateAllMsg bool   `json:"adminCanUpdateAllMsg"`
	AdminCanDeleteAllMsg bool   `json:"adminCanDeleteAllMsg"`
	IsAutoComputeTags    bool   `json:"isAutoComputeTags"`
	IsAutoComputeLabels  bool   `json:"isAutoComputeLabels"`
	Recursive            bool   `json:"recursive"`
}

// TopicParameter updates param on one topic
func (c *Client) TopicParameter(params TopicParameters) ([]byte, error) {
	b, err := json.Marshal(params)
	if err != nil {
		ErrorLogFunc("Error while Unmarshal topic params: %s", err)
		return nil, err
	}

	out, err := c.reqWant(http.MethodPut, 201, "/topic/param", b)
	if err != nil {
		ErrorLogFunc("Error updating params: %s", err)
		return nil, err
	}
	return out, nil
}

// TopicAddFilter adds a filter on a topic
func (c *Client) TopicAddFilter(filter Filter) ([]byte, error) {
	out, err := c.simplePutAndGetBytes(fmt.Sprintf("/topic/add/filter"), 201, filter)
	if err != nil {
		ErrorLogFunc("Error removing a filter on topic %s: %s", filter.Topic, err)
		return nil, err
	}
	return out, nil
}

// TopicDeleteFilters removes a filter on a topic
func (c *Client) TopicRemoveFilter(filter Filter) ([]byte, error) {
	out, err := c.simplePutAndGetBytes(fmt.Sprintf("/topic/remove/filter"), 201, filter)
	if err != nil {
		ErrorLogFunc("Error removing a filter on topic %s: %s", filter.Topic, err)
		return nil, err
	}
	return out, nil
}

// TopicUpdateFilters removes a filter on a topic
func (c *Client) TopicUpdateFilter(filter Filter) ([]byte, error) {
	out, err := c.simplePutAndGetBytes(fmt.Sprintf("/topic/update/filter"), 201, filter)
	if err != nil {
		ErrorLogFunc("Error updating a filter on topic %s: %s", filter.Topic, err)
		return nil, err
	}
	return out, nil
}
