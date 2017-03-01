package tat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/now"
)

const (
	// DefaultMessageMaxSize is max size of message, can be overrided by topic
	DefaultMessageMaxSize = 140
	// DefaultMessageMaxReplies is max number of replies on a message, can be overrided by topic
	DefaultMessageMaxReplies = 30

	// True in url http way -> string
	True = "true"
	// False in url http way -> string
	False = "false"
	// TreeViewNoTree is default value
	TreeViewNoTree = "notree"
	// TreeViewOneTree is onetree value for treeView
	TreeViewOneTree = "onetree"
	// TreeViewFullTree is fulltree value for treeView
	TreeViewFullTree = "fulltree"

	// MessageActionCreate for create a message
	MessageActionCreate = "create"
	// MessageActionUpdate for update action on a message
	MessageActionUpdate = "update"
	// MessageActionReply for "reply action on a message
	MessageActionReply = "reply"
	// MessageActionLike for  "like action on a message
	MessageActionLike = "like"
	// MessageActionUnlike for unlike action on a message
	MessageActionUnlike = "unlike"
	// MessageActionLabel for "label action on a message
	MessageActionLabel = "label"
	// MessageActionUnlabel for nlabel action on a message
	MessageActionUnlabel = "unlabel"
	// MessageActionVoteup for voteup action on a message
	MessageActionVoteup = "voteup"
	// MessageActionVotedown for tedown action on a message
	MessageActionVotedown = "votedown"
	// MessageActionUnvoteup for voteup action on a message
	MessageActionUnvoteup = "unvoteup"
	// MessageActionUnvotedown for tedown action on a message
	MessageActionUnvotedown = "unvotedown"
	// MessageActionRelabel for relabel action on a message
	MessageActionRelabel = "relabel"
	// MessageActionRelabelOrCreate for relabeloradd action on a message
	MessageActionRelabelOrCreate = "relabelorcreate"
	// MessageActionConcat for concat action on a message
	MessageActionConcat = "concat"
	// MessageActionMove for move action on a message
	MessageActionMove = "move"
	// MessageActionTask for task action on a message
	MessageActionTask = "task"
	// MessageActionUntask for untask action on a message
	MessageActionUntask = "untask"
)

// Author struct
type Author struct {
	Username string `bson:"username" json:"username"`
	Fullname string `bson:"fullname" json:"fullname"`
}

// Label struct
type Label struct {
	Text  string `bson:"text" json:"text"`
	Color string `bson:"color" json:"color"`
}

// Message struc
type Message struct {
	ID              string    `bson:"_id"             json:"_id"`
	Text            string    `bson:"text"            json:"text"`
	Topic           string    `bson:"topic"           json:"topic"`
	InReplyOfID     string    `bson:"inReplyOfID"     json:"inReplyOfID"`
	InReplyOfIDRoot string    `bson:"inReplyOfIDRoot" json:"inReplyOfIDRoot"`
	NbLikes         int64     `bson:"nbLikes"         json:"nbLikes"`
	Labels          []Label   `bson:"labels"          json:"labels,omitempty"`
	Likers          []string  `bson:"likers"          json:"likers,omitempty"`
	VotersUP        []string  `bson:"votersUP"        json:"votersUP,omitempty"`
	VotersDown      []string  `bson:"votersDown"      json:"votersDown,omitempty"`
	NbVotesUP       int64     `bson:"nbVotesUP"       json:"nbVotesUP"`
	NbVotesDown     int64     `bson:"nbVotesDown"     json:"nbVotesDown"`
	UserMentions    []string  `bson:"userMentions"    json:"userMentions,omitempty"`
	Urls            []string  `bson:"urls"            json:"urls,omitempty"`
	Tags            []string  `bson:"tags"            json:"tags,omitempty"`
	DateCreation    float64   `bson:"dateCreation"    json:"dateCreation"`
	DateUpdate      float64   `bson:"dateUpdate"      json:"dateUpdate"`
	Author          Author    `bson:"author"          json:"author"`
	Replies         []Message `bson:"-"               json:"replies,omitempty"`
	NbReplies       int64     `bson:"nbReplies"       json:"nbReplies"`
}

// MessageCriteria are used to list messages
type MessageCriteria struct {
	Skip                    int
	Limit                   int
	TreeView                string
	IDMessage               string
	InReplyOfID             string
	InReplyOfIDRoot         string
	AllIDMessage            string // search in IDMessage OR InReplyOfID OR InReplyOfIDRoot
	Text                    string
	Topic                   string
	Label                   string `bson:"label" json:"label,omitempty"`
	StartLabel              string `bson:"startLabel" json:"startLabel,omitempty"`
	NotLabel                string `bson:"notLabel" json:"notLabel,omitempty"`
	AndLabel                string `bson:"andLabel" json:"andLabel,omitempty"`
	Tag                     string `bson:"tag" json:"tag,omitempty"`
	StartTag                string `bson:"startTag" json:"startTag,omitempty"`
	NotTag                  string `bson:"notTag" json:"notTag,omitempty"`
	AndTag                  string `bson:"andTag" json:"andTag,omitempty"`
	Username                string `bson:"username" json:"username,omitempty"`
	DateMinCreation         string
	DateMaxCreation         string
	DateMinUpdate           string
	DateMaxUpdate           string
	LastMinCreation         string
	LastMaxCreation         string
	LastMinUpdate           string
	LastMaxUpdate           string
	LastHourMinCreation     string
	LastHourMaxCreation     string
	LastHourMinUpdate       string
	LastHourMaxUpdate       string
	DateRefCreation         string
	DateRefDeltaMinCreation string
	DateRefDeltaMaxCreation string
	DateRefUpdate           string
	DateRefDeltaMinUpdate   string
	DateRefDeltaMaxUpdate   string
	LimitMinNbReplies       string
	LimitMaxNbReplies       string
	LimitMinNbVotesUP       string
	LimitMinNbVotesDown     string
	LimitMaxNbVotesUP       string
	LimitMaxNbVotesDown     string
	OnlyMsgRoot             string `bson:"onlyMsgRoot" json:"onlyMsgRoot,omitempty"`
	OnlyMsgReply            string `bson:"onlyMsgReply" json:"onlyMsgReply,omitempty"`
	OnlyCount               string
	SortBy                  string `bson:"sortBy" json:"sortBy"`
}

// CacheKey returns cache key value
func (m *MessageCriteria) CacheKey() []string {
	s := []string{}
	if m == nil {
		return s
	}
	if m.Topic != "" {
		s = append(s, "Topic="+m.Topic)
	}
	if m.Skip != 0 {
		s = append(s, "Skip="+strconv.Itoa(m.Skip))
	}
	if m.Limit != 0 {
		s = append(s, "Limit="+strconv.Itoa(m.Limit))
	}
	if m.TreeView != "" {
		s = append(s, "TreeView="+m.TreeView)
	}
	if m.IDMessage != "" {
		s = append(s, "IDMessage="+m.IDMessage)
	}
	if m.InReplyOfID != "" {
		s = append(s, "InReplyOfID="+m.InReplyOfID)
	}
	if m.InReplyOfIDRoot != "" {
		s = append(s, "InReplyOfIDRoot="+m.InReplyOfIDRoot)
	}
	if m.AllIDMessage != "" {
		s = append(s, "AllIDMessage="+m.AllIDMessage)
	}
	if m.Text != "" {
		s = append(s, "Text="+m.Text)
	}
	if m.Label != "" {
		s = append(s, "Label="+m.Label)
	}
	if m.StartLabel != "" {
		s = append(s, "StartLabel="+m.StartLabel)
	}
	if m.NotLabel != "" {
		s = append(s, "NotLabel="+m.NotLabel)
	}
	if m.AndLabel != "" {
		s = append(s, "AndLabel="+m.AndLabel)
	}
	if m.Tag != "" {
		s = append(s, "Tag="+m.Tag)
	}
	if m.StartTag != "" {
		s = append(s, "StartTag="+m.StartTag)
	}
	if m.NotTag != "" {
		s = append(s, "NotTag="+m.NotTag)
	}
	if m.AndTag != "" {
		s = append(s, "AndTag="+m.AndTag)
	}
	if m.Username != "" {
		s = append(s, "Username="+m.Username)
	}
	if m.DateMinCreation != "" {
		s = append(s, "DateMinCreation="+m.DateMinCreation)
	}
	if m.DateMaxCreation != "" {
		s = append(s, "DateMaxCreation="+m.DateMaxCreation)
	}
	if m.DateMinUpdate != "" {
		s = append(s, "DateMinUpdate="+m.DateMinUpdate)
	}
	if m.DateMaxUpdate != "" {
		s = append(s, "DateMaxUpdate="+m.DateMaxUpdate)
	}
	if m.LastMinCreation != "" {
		s = append(s, "LastMinCreation="+m.LastMinCreation)
	}
	if m.LastMaxCreation != "" {
		s = append(s, "LastMaxCreation="+m.LastMaxCreation)
	}
	if m.LastMinUpdate != "" {
		s = append(s, "LastMinUpdate="+m.LastMinUpdate)
	}
	if m.LastMaxUpdate != "" {
		s = append(s, "LastMaxUpdate="+m.LastMaxUpdate)
	}
	if m.LastHourMinCreation != "" {
		s = append(s, "LastHourMinCreation="+m.LastHourMinCreation)
	}
	if m.LastHourMaxCreation != "" {
		s = append(s, "LastHourMaxCreation="+m.LastHourMaxCreation)
	}
	if m.LastHourMinUpdate != "" {
		s = append(s, "LastHourMinUpdate="+m.LastHourMinUpdate)
	}
	if m.LastHourMaxUpdate != "" {
		s = append(s, "LastHourMaxUpdate="+m.LastHourMaxUpdate)
	}
	if m.LimitMinNbReplies != "" {
		s = append(s, "LimitMinNbReplies="+m.LimitMinNbReplies)
	}
	if m.LimitMaxNbReplies != "" {
		s = append(s, "LimitMaxNbReplies="+m.LimitMaxNbReplies)
	}
	if m.LimitMinNbVotesUP != "" {
		s = append(s, "LimitMinNbVotesUP="+m.LimitMinNbVotesUP)
	}
	if m.LimitMinNbVotesDown != "" {
		s = append(s, "LimitMinNbVotesDown="+m.LimitMinNbVotesDown)
	}
	if m.LimitMaxNbVotesUP != "" {
		s = append(s, "LimitMaxNbVotesUP="+m.LimitMaxNbVotesUP)
	}
	if m.LimitMaxNbVotesDown != "" {
		s = append(s, "LimitMaxNbVotesDown="+m.LimitMaxNbVotesDown)
	}
	if m.DateRefCreation != "" {
		s = append(s, "DateRefCreation="+m.DateRefCreation)
	}
	if m.DateRefDeltaMinCreation != "" {
		s = append(s, "DateRefDeltaMinCreation="+m.DateRefDeltaMinCreation)
	}
	if m.DateRefDeltaMaxCreation != "" {
		s = append(s, "DateRefDeltaMaxCreation="+m.DateRefDeltaMaxCreation)
	}
	if m.DateRefUpdate != "" {
		s = append(s, "DateRefUpdate="+m.DateRefUpdate)
	}
	if m.DateRefDeltaMinUpdate != "" {
		s = append(s, "DateRefDeltaMinUpdate="+m.DateRefDeltaMinUpdate)
	}
	if m.DateRefDeltaMaxUpdate != "" {
		s = append(s, "DateRefDeltaMaxUpdate="+m.DateRefDeltaMaxUpdate)
	}
	if m.OnlyMsgRoot != "" {
		s = append(s, "OnlyMsgRoot="+m.OnlyMsgRoot)
	}
	if m.OnlyMsgReply != "" {
		s = append(s, "OnlyMsgReply="+m.OnlyMsgReply)
	}
	if m.OnlyCount != "" {
		s = append(s, "OnlyCount="+m.OnlyCount)
	}
	if m.SortBy != "" {
		s = append(s, "SortBy="+m.SortBy)
	}
	return s
}

// MessageReferenceJSON is used for and action On A Existing Message
type MessageReferenceJSON struct {
	TagReference        string `json:"tagReference"`
	StartTagReference   string `json:"startTagReference"`
	LabelReference      string `json:"labelReference"`
	StartLabelReference string `json:"startLabelReference"`
	IDReference         string `json:"idReference"`
}

// MessagesJSON represents a message and information if current topic is RW
type MessagesJSON struct {
	Messages     []Message `json:"messages"`
	IsTopicRw    bool      `json:"isTopicRw"`
	IsTopicAdmin bool      `json:"isTopicAdmin"`
}

// MessagesCountJSON represents count of messages
type MessagesCountJSON struct {
	Count int `json:"count"`
}

// MessageJSONOut represents a message and an additional info
type MessageJSONOut struct {
	Message Message `json:"message"`
	Info    string  `json:"info"`
}

type MessagesJSONIn struct {
	Messages []*MessageJSON `json:"messages"`
}

// MessageJSON represents a message with action on it
type MessageJSON struct {
	ID                  string `json:"_id"`
	Text                string `json:"text"`
	Option              string `json:"option"`
	Topic               string
	IDReference         string        `json:"idReference"`
	StartTagReference   string        `json:"startTagReference"`
	StartLabelReference string        `json:"startLabelReference"`
	TagReference        string        `json:"tagReference"`
	LabelReference      string        `json:"labelReference"`
	OnlyRootReference   string        `json:"onlyRootReference"`
	Action              string        `json:"action"`
	DateCreation        float64       `json:"dateCreation"`
	Labels              []Label       `json:"labels"`
	Options             []string      `json:"options"`
	Replies             []string      `json:"replies"`
	Messages            []MessageJSON `json:"messages"` // same as replies, but with Labels...
}

// MessageAdd post a tat message
func (c *Client) MessageAdd(message MessageJSON) (*MessageJSONOut, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	if message.Topic == "" {
		return nil, fmt.Errorf("A message must have a Topic")
	}

	return c.processForMessageJSONOut("POST", "/message"+message.Topic, 201, message)
}

// MessageAddBulk post many tat message (root msg or replies)
func (c *Client) MessageAddBulk(messages []MessageJSON) ([]MessageJSONOut, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	msgs := []MessageJSONOut{}
	for _, message := range messages {
		if message.Topic == "" {
			return nil, fmt.Errorf("A message must have a Topic")
		}

		m, err := c.processForMessageJSONOut("POST", "/message"+message.Topic, 201, message)
		if err != nil {
			return msgs, err
		}
		msgs = append(msgs, *m)
	}
	return msgs, nil
}

// MessageReply post a reply to a message
func (c *Client) MessageReply(topic, idMessage string, reply string) (*MessageJSONOut, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	message := MessageJSON{
		Action:      MessageActionReply,
		Topic:       topic,
		IDReference: idMessage,
		Text:        reply,
	}
	return c.processForMessageJSONOut("POST", "/message"+message.Topic, 201, message)
}

// MessageDelete delete a message.
// cascade : delete message and its replies. cascadeForce : delete message and its replies, event if it's in a Tasks Topic of one user
func (c *Client) MessageDelete(id, topic string, cascade bool, cascadeForce bool) ([]byte, error) {
	var err error
	var out []byte
	if cascade {
		out, err = c.reqWant(http.MethodDelete, 200, "/message/cascade/"+id+topic, nil)
	} else if cascadeForce {
		out, err = c.reqWant(http.MethodDelete, 200, "/message/cascadeforce/"+id+topic, nil)
	} else {
		out, err = c.reqWant(http.MethodDelete, 200, "/message/nocascade/"+id+topic, nil)
	}

	if err != nil {
		ErrorLogFunc("Error deleting message: %s", err)
		return nil, err
	}
	return out, nil
}

// MessagesDeleteBulk Delete a list of messages
// delete message and its replies. cascadeForce : delete message and its replies, event if it's in a Tasks Topic of one user
func (c *Client) MessagesDeleteBulk(topic string, cascade bool, cascadeForce bool, criteria MessageCriteria) ([]byte, error) {
	var err error
	var out []byte

	path := fmt.Sprintf("%s?%s", topic, criteria.GetURL())

	if cascade {
		out, err = c.reqWant(http.MethodDelete, 200, "/messages/cascade"+path, nil)
	} else if cascadeForce {
		out, err = c.reqWant(http.MethodDelete, 200, "/messages/cascadeforce"+path, nil)
	} else {
		out, err = c.reqWant(http.MethodDelete, 200, "/messages/nocascade"+path, nil)
	}

	if err != nil {
		ErrorLogFunc("Error deleting messages: %s", err)
		return nil, err
	}
	return out, nil
}

// MessageUpdate updates a message
func (c *Client) MessageUpdate(topic, idMessage string, newText string) (*MessageJSONOut, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	message := MessageJSON{
		Action:      MessageActionUpdate,
		Topic:       topic,
		IDReference: idMessage,
		Text:        newText,
	}
	return c.processForMessageJSONOut("PUT", "/message"+message.Topic, 200, message)
}

// MessageConcat is same as:
/*```
curl -XPUT \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "idReference": "9797q87KJhqsfO7Usdqd", "action": "concat", "text": " additional text"}'\
	https://<tatHostname>:<tatPort>/message/topic/sub-topic
```
*/
func (c *Client) MessageConcat(topic, idMessage string, addText string) (*MessageJSONOut, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	message := MessageJSON{
		Action:      MessageActionConcat,
		Topic:       topic,
		IDReference: idMessage,
		Text:        addText,
	}
	return c.processForMessageJSONOut("PUT", "/message"+message.Topic, 200, message)
}

// MessageMove moves a message from a topic to another
func (c *Client) MessageMove(oldTopic, idMessage, newTopic string) ([]byte, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	message := MessageJSON{
		Topic:       oldTopic,
		IDReference: idMessage,
		Action:      MessageActionMove,
		Option:      newTopic,
	}
	return c.processForMessageJSONOutBytes("PUT", "/message"+message.Topic, 201, message)
}

// MessageTask creates a task from a message
func (c *Client) MessageTask(topic, idMessage string) (*MessageJSONOut, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	message := MessageJSON{
		Topic:       topic,
		IDReference: idMessage,
		Action:      MessageActionTask,
	}
	return c.processForMessageJSONOut("PUT", "/message"+message.Topic, 201, message)
}

// MessageUntask removes doing and doing:username label from a message
func (c *Client) MessageUntask(topic, idMessage string) (*MessageJSONOut, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	message := MessageJSON{
		Topic:       topic,
		IDReference: idMessage,
		Action:      MessageActionUntask,
	}
	return c.processForMessageJSONOut("PUT", "/message"+message.Topic, 201, message)
}

// MessageLike add a like to a message
func (c *Client) MessageLike(topic, idMessage string) (*MessageJSONOut, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	message := MessageJSON{
		Topic:       topic,
		IDReference: idMessage,
		Action:      MessageActionLike,
	}
	return c.processForMessageJSONOut("PUT", "/message"+message.Topic, 201, message)
}

// MessageUnlike removes a like from a message
func (c *Client) MessageUnlike(topic, idMessage string) (*MessageJSONOut, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	message := MessageJSON{
		Topic:       topic,
		IDReference: idMessage,
		Action:      MessageActionUnlike,
	}
	return c.processForMessageJSONOut("PUT", "/message"+message.Topic, 201, message)
}

// MessageVoteUP add a vote UP to a message
func (c *Client) MessageVoteUP(topic, idMessage string) (*MessageJSONOut, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	message := MessageJSON{
		Topic:       topic,
		IDReference: idMessage,
		Action:      MessageActionVoteup,
	}
	return c.processForMessageJSONOut("PUT", "/message"+message.Topic, 201, message)
}

// MessageVoteDown add a vote down to a message
func (c *Client) MessageVoteDown(topic, idMessage string) (*MessageJSONOut, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	message := MessageJSON{
		Topic:       topic,
		IDReference: idMessage,
		Action:      MessageActionVotedown,
	}
	return c.processForMessageJSONOut("PUT", "/message"+message.Topic, 201, message)
}

// MessageUnVoteUP removes a vote UP from a message
func (c *Client) MessageUnVoteUP(topic, idMessage string) (*MessageJSONOut, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	message := MessageJSON{
		Topic:       topic,
		IDReference: idMessage,
		Action:      MessageActionUnvoteup,
	}
	return c.processForMessageJSONOut("PUT", "/message"+message.Topic, 201, message)
}

// MessageUnVoteDown removes a vote down
func (c *Client) MessageUnVoteDown(topic, idMessage string) (*MessageJSONOut, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	message := MessageJSON{
		Topic:       topic,
		IDReference: idMessage,
		Action:      MessageActionUnvotedown,
	}
	return c.processForMessageJSONOut("PUT", "/message"+message.Topic, 201, message)
}

// MessageLabel add a label to a message
func (c *Client) MessageLabel(topic, idMessage string, label Label) (*MessageJSONOut, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	message := MessageJSON{
		Topic:       topic,
		IDReference: idMessage,
		Text:        label.Text,
		Option:      label.Color,
		Action:      MessageActionLabel,
	}
	return c.processForMessageJSONOut("PUT", "/message"+message.Topic, 201, message)
}

// MessageUnlabel removes a label from one message
func (c *Client) MessageUnlabel(topic, idMessage, label string) (*MessageJSONOut, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	message := MessageJSON{
		Topic:       topic,
		IDReference: idMessage,
		Text:        label,
		Action:      MessageActionUnlabel,
	}
	return c.processForMessageJSONOut("PUT", "/message"+message.Topic, 201, message)
}

// MessageRelabel removes all labels and add new ones to a message
func (c *Client) MessageRelabel(topic, idMessage string, labels []Label, options []string) (*MessageJSONOut, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	message := MessageJSON{
		Topic:       topic,
		IDReference: idMessage,
		Labels:      labels,
		Options:     options,
		Action:      MessageActionRelabel,
	}

	return c.processForMessageJSONOut("PUT", "/message"+message.Topic, 201, message)
}

// MessageRelabelOrCreate removes all labels and add new ones to a message if message exists, create message otherwise
//  msg := tat.MessageJSON{
//    Text:         "a text with a #tag",
//    Labels:       []tat.Label{{Text:"textLabel", Color:"red"}},
//    TagReference: "a #tag",
//    Topic:        "/Internal/YourTopic",
//  }
//  if _, err := getClient().MessageRelabelOrCreate(msg); err != nil {
//    return fmt.Errorf("Error while MessageAdd:%s", err)
//  }
func (c *Client) MessageRelabelOrCreate(msg MessageJSON) (*MessageJSONOut, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	if msg.Topic == "" {
		return nil, fmt.Errorf("A message must have a Topic")
	}

	msg.Action = MessageActionRelabelOrCreate

	return c.processForMessageJSONOut("PUT", "/message"+msg.Topic, 201, msg)
}

func (c *Client) processForMessageJSONOutBytes(method, path string, want int, message MessageJSON) ([]byte, error) {
	b, err := json.Marshal(message)
	if err != nil {
		ErrorLogFunc("Error while marshal message: %s", err)
		return nil, err
	}

	body, err := c.reqWant(method, want, path, b)
	if err != nil {
		ErrorLogFunc("Error while marshal message: %s", err)
		return nil, err
	}
	return body, err
}

func (c *Client) processForMessageJSONOut(method, path string, want int, message MessageJSON) (*MessageJSONOut, error) {
	body, err := c.processForMessageJSONOutBytes(method, path, want, message)
	if err != nil {
		return nil, err
	}
	out := &MessageJSONOut{}
	if err := json.Unmarshal(body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetURL returns URL for messageCriteria
func (m *MessageCriteria) GetURL() string {
	v := url.Values{}
	v.Set("skip", strconv.Itoa(m.Skip))
	v.Set("limit", strconv.Itoa(m.Limit))

	if m.TreeView != "" {
		v.Set("treeView", m.TreeView)
	}
	if m.IDMessage != "" {
		v.Set("idMessage", m.IDMessage)
	}
	if m.InReplyOfID != "" {
		v.Set("inReplyOfID", m.InReplyOfID)
	}
	if m.InReplyOfIDRoot != "" {
		v.Set("inReplyOfIDRoot", m.InReplyOfIDRoot)
	}
	if m.AllIDMessage != "" {
		v.Set("allIDMessage", m.AllIDMessage)
	}
	if m.Text != "" {
		v.Set("text", m.Text)
	}
	if m.Topic != "" {
		v.Set("topic", m.Topic)
	}
	if m.Label != "" {
		v.Set("label", m.Label)
	}
	if m.StartLabel != "" {
		v.Set("startLabel", m.StartLabel)
	}
	if m.NotLabel != "" {
		v.Set("notLabel", m.NotLabel)
	}
	if m.AndLabel != "" {
		v.Set("andLabel", m.AndLabel)
	}
	if m.Tag != "" {
		v.Set("tag", m.Tag)
	}
	if m.StartTag != "" {
		v.Set("startTag", m.StartTag)
	}
	if m.NotTag != "" {
		v.Set("notTag", m.NotTag)
	}
	if m.AndTag != "" {
		v.Set("andTag", m.AndTag)
	}
	if m.DateMinCreation != "" {
		v.Set("dateMinCreation", m.DateMinCreation)
	}
	if m.DateMaxCreation != "" {
		v.Set("dateMaxCreation", m.DateMaxCreation)
	}
	if m.DateMinUpdate != "" {
		v.Set("dateMinUpdate", m.DateMinUpdate)
	}
	if m.DateMaxUpdate != "" {
		v.Set("dateMaxUpdate", m.DateMaxUpdate)
	}
	if m.DateRefCreation != "" {
		v.Set("dateRefCreation", m.DateRefCreation)
	}
	if m.DateRefDeltaMinCreation != "" {
		v.Set("dateRefDeltaMinCreation", m.DateRefDeltaMinCreation)
	}
	if m.DateRefDeltaMaxCreation != "" {
		v.Set("dateRefDeltaMaxCreation", m.DateRefDeltaMaxCreation)
	}
	if m.DateRefUpdate != "" {
		v.Set("dateRefUpdate", m.DateRefUpdate)
	}
	if m.DateRefDeltaMinUpdate != "" {
		v.Set("dateRefDeltaMinUpdate", m.DateRefDeltaMinUpdate)
	}
	if m.DateRefDeltaMaxUpdate != "" {
		v.Set("dateRefDeltaMaxUpdate", m.DateRefDeltaMaxUpdate)
	}
	if m.LastMinCreation != "" {
		v.Set("lastMinCreation", m.LastMinCreation)
	}
	if m.LastMaxCreation != "" {
		v.Set("lastMaxCreation", m.LastMaxCreation)
	}
	if m.LastMinUpdate != "" {
		v.Set("lastMinUpdate", m.LastMinUpdate)
	}
	if m.LastMaxUpdate != "" {
		v.Set("lastMaxUpdate", m.LastMaxUpdate)
	}
	if m.LastHourMinCreation != "" {
		v.Set("lastHourMinCreation", m.LastHourMinCreation)
	}
	if m.LastHourMaxCreation != "" {
		v.Set("lastHourMaxCreation", m.LastHourMaxCreation)
	}
	if m.LastHourMinUpdate != "" {
		v.Set("lastHourMinUpdate", m.LastHourMinUpdate)
	}
	if m.LastHourMaxUpdate != "" {
		v.Set("lastHourMaxUpdate", m.LastHourMaxUpdate)
	}
	if m.Username != "" {
		v.Set("username", m.Username)
	}
	if m.LimitMinNbReplies != "" {
		v.Set("limitMinNbReplies", m.LimitMinNbReplies)
	}
	if m.LimitMaxNbReplies != "" {
		v.Set("limitMaxNbReplies", m.LimitMaxNbReplies)
	}
	if m.LimitMinNbVotesUP != "" {
		v.Set("limitMinNbVotesUP", m.LimitMinNbVotesUP)
	}
	if m.LimitMaxNbVotesUP != "" {
		v.Set("limitMaxNbVotesUP", m.LimitMaxNbVotesUP)
	}
	if m.LimitMinNbVotesDown != "" {
		v.Set("limitMinNbVotesDown", m.LimitMinNbVotesDown)
	}
	if m.LimitMaxNbVotesDown != "" {
		v.Set("limitMaxNbVotesDown", m.LimitMaxNbVotesDown)
	}
	if m.OnlyMsgRoot == True {
		v.Set("onlyMsgRoot", "true")
	}
	if m.OnlyMsgReply == True {
		v.Set("onlyMsgReply", "true")
	}
	if m.OnlyCount == True {
		v.Set("onlyCount", "true")
	}
	if m.SortBy != "" {
		v.Set("sortBy", m.SortBy)
	}
	return v.Encode()
}

// GetMessageCriteriaFromURLValues returns a MessagesCriteria from a url.Values
func GetMessageCriteriaFromURLValues(values url.Values) (*MessageCriteria, error) {
	c := &MessageCriteria{}
	for k, v := range values {
		switch k {
		case "skip":
			i, err := strconv.Atoi(v[0])
			if err != nil {
				c.Skip = 0
			} else {
				c.Skip = i
			}
		case "limit":
			i, err := strconv.Atoi(v[0])
			if err != nil {
				c.Limit = 10
			} else {
				c.Limit = i
			}
		case "treeView":
			c.TreeView = v[0]
		case "idMessage":
			c.IDMessage = v[0]
		case "inReplyOfID":
			c.InReplyOfID = v[0]
		case "inReplyOfIDRoot":
			c.InReplyOfIDRoot = v[0]
		case "allIDMessage":
			c.AllIDMessage = v[0]
		case "text":
			c.Text = v[0]
		case "topic":
			c.Topic = v[0]
		case "label":
			c.Label = v[0]
		case "startLabel":
			c.StartLabel = v[0]
		case "notLabel":
			c.NotLabel = v[0]
		case "andLabel":
			c.AndLabel = v[0]
		case "tag":
			c.Tag = v[0]
		case "startTag":
			c.StartTag = v[0]
		case "notTag":
			c.NotTag = v[0]
		case "andTag":
			c.AndTag = v[0]
		case "dateMinCreation":
			c.DateMinCreation = v[0]
		case "dateMaxCreation":
			c.DateMaxCreation = v[0]
		case "dateMinUpdate":
			c.DateMinUpdate = v[0]
		case "dateMaxUpdate":
			c.DateMaxUpdate = v[0]
		case "lastMinCreation":
			c.LastMinCreation = v[0]
		case "lastMaxCreation":
			c.LastMaxCreation = v[0]
		case "lastMinUpdate":
			c.LastMinUpdate = v[0]
		case "lastMaxUpdate":
			c.LastMaxUpdate = v[0]
		case "lastHourMinCreation":
			c.LastHourMinCreation = v[0]
		case "lastHourMaxCreation":
			c.LastHourMaxCreation = v[0]
		case "lastHourMinUpdate":
			c.LastHourMinUpdate = v[0]
		case "lastHourMaxUpdate":
			c.LastHourMaxUpdate = v[0]
		case "dateRefCreation":
			c.DateRefCreation = v[0]
		case "dateRefDeltaMinCreation":
			c.DateRefDeltaMinCreation = v[0]
		case "dateRefDeltaMaxCreation":
			c.DateRefDeltaMaxCreation = v[0]
		case "dateRefUpdate":
			c.DateRefUpdate = v[0]
		case "dateRefDeltaMinUpdate":
			c.DateRefDeltaMinUpdate = v[0]
		case "dateRefDeltaMaxUpdate":
			c.DateRefDeltaMaxUpdate = v[0]
		case "username":
			c.Username = v[0]
		case "limitMinNbReplies":
			c.LimitMinNbReplies = v[0]
		case "limitMaxNbReplies":
			c.LimitMaxNbReplies = v[0]
		case "limitMinNbVotesUP":
			c.LimitMinNbVotesUP = v[0]
		case "limitMaxNbVotesUP":
			c.LimitMaxNbVotesUP = v[0]
		case "limitMinNbVotesDown":
			c.LimitMinNbVotesDown = v[0]
		case "limitMaxNbVotesDown":
			c.LimitMaxNbVotesDown = v[0]
		case "onlyMsgRoot":
			c.OnlyMsgRoot = v[0]
		case "onlyMsgReply":
			c.OnlyMsgReply = v[0]
		case "onlyCount":
			c.OnlyCount = v[0]
		case "sortBy":
			c.SortBy = v[0]
		}
	}

	return c, nil
}

//MessageCount count messages on a topic according to criterias
func (c *Client) MessageCount(topic string, criteria *MessageCriteria) (*MessagesCountJSON, error) {

	criteria.OnlyCount = "true"
	body, err := c.messagesList(topic, criteria)
	if err != nil {
		return nil, err
	}

	var messages = MessagesCountJSON{}
	if err := json.Unmarshal(body, &messages); err != nil {
		ErrorLogFunc("Error getting messages list (count): %s", err)
		return nil, err
	}

	return &messages, nil
}

//MessageList lists messages on a topic according to criterias
func (c *Client) MessageList(topic string, criteria *MessageCriteria) (*MessagesJSON, error) {

	criteria.OnlyCount = "false"
	body, err := c.messagesList(topic, criteria)
	if err != nil {
		return nil, err
	}
	var messages = MessagesJSON{}
	if err := json.Unmarshal(body, &messages); err != nil {
		ErrorLogFunc("Error getting messages list: %s", err)
		return nil, err
	}

	return &messages, nil
}

func (c *Client) messagesList(topic string, criteria *MessageCriteria) ([]byte, error) {
	if criteria == nil {
		criteria = &MessageCriteria{
			Skip:  0,
			Limit: 100,
		}
	}
	criteria.Topic = topic

	path := fmt.Sprintf("/messages%s?%s", criteria.Topic, criteria.GetURL())
	DebugLogFunc("MessageList: Path requested: %s", path)

	body, err := c.reqWant(http.MethodGet, 200, path, nil)
	if err != nil {
		ErrorLogFunc("messagesList: Error getting messages list: %s", err)
		return nil, err
	}

	DebugLogFunc("MessageList: Messages List Response, len body %d", len(body))
	return body, err
}

// GetLabel returns label, and position if message contains label
func (m *Message) GetLabel(label string) (int, Label, error) {
	for idx, cur := range m.Labels {
		if cur.Text == label {
			return idx, cur, nil
		}
	}
	l := Label{}
	return -1, l, fmt.Errorf("label %s not found", label)
}

// ContainsLabel returns true if message contains label
func (m *Message) ContainsLabel(label string) bool {
	_, _, err := m.GetLabel(label)
	return err == nil
}

// IsDoing returns true if message contains label doing or starts with doing:
func (m *Message) IsDoing() bool {
	for _, label := range m.Labels {
		if label.Text == "doing" || strings.HasPrefix(label.Text, "doing:") {
			return true
		}
	}
	return false
}

// GetTag returns position, tag is message contains tag
func (m *Message) GetTag(tag string) (int, string, error) {
	for idx, cur := range m.Tags {
		if cur == tag {
			return idx, cur, nil
		}
	}
	return -1, "", fmt.Errorf("tag %s not found", tag)
}

// ContainsTag returns true if message contains tag
func (m *Message) ContainsTag(tag string) bool {
	_, _, err := m.GetTag(tag)
	return err == nil
}

func GetDateRef(pattern string) (time.Time, error) {
	var dateRef time.Time
	now.FirstDayMonday = true
	switch pattern {
	case "BeginningOfMinute":
		return now.BeginningOfMinute(), nil
	case "BeginningOfHour":
		return now.BeginningOfHour(), nil
	case "BeginningOfDay":
		return now.BeginningOfDay(), nil
	case "BeginningOfWeek":
		return now.BeginningOfWeek(), nil
	case "BeginningOfMonth":
		return now.BeginningOfMonth(), nil
	case "BeginningOfQuarter":
		return now.BeginningOfQuarter(), nil
	case "BeginningOfYear":
		return now.BeginningOfYear(), nil
	}
	return dateRef, fmt.Errorf("Invalid pattern:%s", pattern)
}

// Format return string formatted message
//  default format: format:dateUpdate,username,text,labels
// Avalable fields:
// id,text,topic,inReplyOfID,inReplyOfIDRoot,nbLikes,labels,
// votersUP,votersDown,nbVotesUP,nbVotesDown,userMentions,
// urls,tags,dateCreation,dateUpdate,username,fullname,nbReplies,tatwebuiURL
func (m *Message) Format(format string, tatwebuiBaseURL string) (string, error) {

	if format == "" {
		format = "dateUpdate,username,text,labels"
	}

	labels := ""
	for _, l := range m.Labels {
		labels += l.Text + ","
	}
	if labels != "" {
		// remove last ","
		labels = strings.TrimSuffix(labels, ",")
	}

	f := strings.Split(format, ",")
	if len(f) == 0 {
		return "", fmt.Errorf("Invalid format:%s", format)
	}
	out := ""
	for _, t := range f {
		switch t {
		case "id":
			out += fmt.Sprintf("%s ", m.ID)
		case "text":
			out += fmt.Sprintf("%s ", m.Text)
		case "topic":
			out += fmt.Sprintf("%s ", m.Topic)
		case "inReplyOfID":
			out += fmt.Sprintf("inReplyOfID:%s ", m.InReplyOfID)
		case "inReplyOfIDRoot":
			out += fmt.Sprintf("inReplyOfIDRoot:%s ", m.InReplyOfIDRoot)
		case "nbLikes":
			out += fmt.Sprintf("nbLikes:%d ", m.NbLikes)
		case "labels":
			out += fmt.Sprintf("labels:%s ", labels)
		case "votersUP":
			out += fmt.Sprintf("votersUP:%s ", strings.Join(m.VotersUP, ","))
		case "votersDown":
			out += fmt.Sprintf("votersDown:%s ", strings.Join(m.VotersDown, ","))
		case "nbVotesUP":
			out += fmt.Sprintf("nbVotesUP:%d ", m.NbVotesUP)
		case "nbVotesDown":
			out += fmt.Sprintf("nbVotesDown:%d ", m.NbVotesDown)
		case "userMentions":
			out += fmt.Sprintf("%s ", strings.Join(m.UserMentions, ","))
		case "urls":
			out += fmt.Sprintf("url:%s ", strings.Join(m.Urls, ","))
		case "tags":
			out += fmt.Sprintf("tags:%s ", strings.Join(m.Tags, ","))
		case "dateCreation":
			out += fmt.Sprintf("%s ", time.Unix(int64(m.DateCreation), 0).Format(time.Stamp))
		case "dateUpdate":
			out += fmt.Sprintf("%s ", time.Unix(int64(m.DateUpdate), 0).Format(time.Stamp))
		case "username":
			out += fmt.Sprintf("%s ", m.Author.Username)
		case "fullname":
			out += fmt.Sprintf("%s ", m.Author.Fullname)
		case "nbReplies":
			out += fmt.Sprintf("nbReplies:%d ", m.NbReplies)
		case "tatwebuiURL":
			out += fmt.Sprintf("tatwebui:%s%s?idMessage=%s", tatwebuiBaseURL, m.Topic, m.ID)
		}
	}

	if out == "" {
		return "", fmt.Errorf("Invalid format:%s", format)
	}
	return out, nil
}
