package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/ovh/tat"
	groupDB "github.com/ovh/tat/api/group"
	messageDB "github.com/ovh/tat/api/message"
	presenceDB "github.com/ovh/tat/api/presence"
	topicDB "github.com/ovh/tat/api/topic"
	userDB "github.com/ovh/tat/api/user"
	"github.com/spf13/viper"
)

// TopicsController contains all methods about topics manipulation
type TopicsController struct{}

func (*TopicsController) buildCriteria(ctx *gin.Context, user *tat.User) *tat.TopicCriteria {
	c := tat.TopicCriteria{}
	skip, e := strconv.Atoi(ctx.DefaultQuery("skip", "0"))
	if e != nil {
		skip = 0
	}
	c.Skip = skip
	limit, e2 := strconv.Atoi(ctx.DefaultQuery("limit", "500"))
	if e2 != nil {
		limit = 500
	}
	c.Limit = limit
	c.IDTopic = ctx.Query("idTopic")
	c.Topic = ctx.Query("topic")
	if c.Topic != "" && !strings.HasPrefix(c.Topic, "/") {
		c.Topic = "/" + c.Topic
	}
	c.Description = ctx.Query("description")
	c.DateMinCreation = ctx.Query("dateMinCreation")
	c.DateMaxCreation = ctx.Query("dateMaxCreation")
	c.GetNbMsgUnread = ctx.Query("getNbMsgUnread")
	c.OnlyFavorites = ctx.Query("onlyFavorites")
	c.GetForTatAdmin = ctx.Query("getForTatAdmin")
	c.TopicPath = ctx.Query("topicPath")

	if c.OnlyFavorites == "true" {
		c.Topic = strings.Join(user.FavoritesTopics, ",")
	}

	if c.SortBy == "" {
		c.SortBy = "topic"
	}
	return &c
}

// List returns the list of topics that can be viewed by user
func (t *TopicsController) List(ctx *gin.Context) {
	var user = &tat.User{}
	found, err := userDB.FindByUsername(user, getCtxUsername(ctx))
	if !found {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "User unknown"})
		return
	} else if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error while fetching user."})
		return
	}
	criteria := t.buildCriteria(ctx, user)
	count, topics, err := topicDB.ListTopics(criteria, user, false, false, false)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error while fetching topics."})
		return
	}

	out := &tat.TopicsJSON{Topics: topics, Count: count}

	if criteria.GetNbMsgUnread == "true" {
		c := &tat.PresenceCriteria{
			Username: user.Username,
		}
		count, presences, err := presenceDB.ListPresencesAllFields(c)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		unread := make(map[string]int)
		var knownPresence bool
		for _, topic := range topics {
			if tat.ArrayContains(user.OffNotificationsTopics, topic.Topic) {
				continue
			}
			knownPresence = false
			for _, presence := range presences {
				if topic.Topic != presence.Topic {
					continue
				}
				knownPresence = true
				if topic.DateLastMessage > presence.DatePresence {
					unread[presence.Topic] = 1
				}
				break
			}
			if !knownPresence {
				unread[topic.Topic] = -1
			}
		}
		out.TopicsMsgUnread = unread
		out.CountTopicsMsgUnread = count
	}
	ctx.JSON(http.StatusOK, out)
}

// OneTopic returns only requested topic, and only if user has read access
func (t *TopicsController) OneTopic(ctx *gin.Context) {
	topicRequest, err := GetParam(ctx, "topic")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, fmt.Errorf("Error while getting topic in param"))
		return
	}
	out, _, code, err := t.innerOneTopic(ctx, topicRequest)
	if err != nil {
		ctx.JSON(code, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(code, out)
}

func (t *TopicsController) innerOneTopic(ctx *gin.Context, topicRequest string) (*tat.TopicJSON, *tat.User, int, error) {
	var user = tat.User{}
	found, err := userDB.FindByUsername(&user, getCtxUsername(ctx))
	if !found {
		return nil, nil, http.StatusInternalServerError, fmt.Errorf("User unknown")
	} else if err != nil {
		return nil, nil, http.StatusInternalServerError, fmt.Errorf("Error while fetching user.")
	}
	topic, errfind := topicDB.FindByTopic(topicRequest, user.IsAdmin, true, true, &user)
	if errfind != nil {
		topic, _, err = checkDMTopic(ctx, topicRequest)
		if err != nil {
			return nil, nil, http.StatusBadRequest, fmt.Errorf("topic " + topicRequest + " does not exist or you have no access on it")
		}
	}

	filters := []tat.Filter{}
	for _, f := range topic.Filters {
		if f.UserID == user.ID {
			filters = append(filters, f)
		}
	}
	topic.Filters = filters
	out := &tat.TopicJSON{Topic: topic}
	out.IsTopicRw, out.IsTopicAdmin = topicDB.GetUserRights(topic, &user)
	return out, &user, http.StatusOK, nil
}

// Create creates a new topic
func (*TopicsController) Create(ctx *gin.Context) {
	var topicIn tat.TopicCreateJSON
	ctx.Bind(&topicIn)

	var user = tat.User{}
	found, err := userDB.FindByUsername(&user, getCtxUsername(ctx))
	if !found {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User unknown"})
		return
	} else if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error while fetching user."})
		return
	}

	var topic tat.Topic
	topic.Topic = topicIn.Topic
	topic.Description = topicIn.Description

	err = topicDB.Insert(&topic, &user)
	if err != nil {
		log.Errorf("Error while InsertTopic %s", err)
		ctx.JSON(tat.Error(err))
		return
	}
	ctx.JSON(http.StatusCreated, topic)
}

// Delete deletes requested topic only if user is Tat admin, or admin on topic
func (t *TopicsController) Delete(ctx *gin.Context) {
	topicRequest, err := GetParam(ctx, "topic")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid Topic"})
		return
	}

	var user = tat.User{}
	found, err := userDB.FindByUsername(&user, getCtxUsername(ctx))
	if !found {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "User unknown"})
		return
	} else if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error while fetching user"})
		return
	}

	paramJSON := tat.ParamTopicUserJSON{
		Topic:     topicRequest,
		Username:  user.Username,
		Recursive: false,
	}

	topic, e := t.preCheckUser(ctx, &paramJSON)
	if e != nil {
		return
	}
	// If user delete a Topic under /Private/username, no check or RW to delete
	if !strings.HasPrefix(topic.Topic, "/Private/"+user.Username) {
		// check if user is Tat admin or admin on this topic
		hasRW := topicDB.IsUserAdmin(topic, &user)
		if !hasRW {
			ctx.JSON(http.StatusForbidden, gin.H{"error": fmt.Errorf("No RW access to topic %s (to delete it)", topic.Topic)})
			return
		}
	}

	c := &tat.MessageCriteria{Topic: topic.Topic, OnlyCount: "true"}
	count, err := messageDB.CountMessages(c, *topic)
	if err != nil {
		log.Errorf("Error while list Messages in Delete %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error while list Messages in Delete topic"})
		return
	}

	if count > 0 {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Could not delete this topic, this topic have messages"})
		return
	}

	if err = topicDB.Delete(topic, &user); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"info": fmt.Sprintf("Topic %s is deleted", topic.Topic)})
}

// Truncate deletes all messages in a topic only if user is Tat admin, or admin on topic
func (t *TopicsController) Truncate(ctx *gin.Context) {
	var paramJSON tat.TopicNameJSON
	ctx.Bind(&paramJSON)
	topic, e := t.preCheckUserAdminOnTopic(ctx, paramJSON.Topic)
	if e != nil {
		ctx.JSON(tat.Error(e))
		return
	}

	nbRemoved, err := topicDB.Truncate(topic)
	if err != nil {
		log.Errorf("Error while truncate topic %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "error while truncate topic " + topic.Topic})
		return
	}
	// 201 returns
	ctx.JSON(http.StatusCreated, gin.H{"info": fmt.Sprintf("%d messages removed", nbRemoved)})
}

// ComputeTags computes tags on one topic
func (t *TopicsController) ComputeTags(ctx *gin.Context) {
	var paramJSON tat.TopicNameJSON
	ctx.Bind(&paramJSON)
	topic, e := t.preCheckUserAdminOnTopic(ctx, paramJSON.Topic)
	if e != nil {
		ctx.JSON(tat.Error(e))
		return
	}

	nbComputed, err := topicDB.ComputeTags(topic)
	if err != nil {
		log.Errorf("Error while compute tags on topic %s: %s", topic.Topic, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "error while compute tags on topic " + topic.Topic})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"info": fmt.Sprintf("%d tags computed", nbComputed)})
}

// ComputeLabels computes labels on one topic
func (t *TopicsController) ComputeLabels(ctx *gin.Context) {
	var paramJSON tat.TopicNameJSON
	ctx.Bind(&paramJSON)
	topic, e := t.preCheckUserAdminOnTopic(ctx, paramJSON.Topic)
	if e != nil {
		ctx.JSON(tat.Error(e))
		return
	}

	nbComputed, err := topicDB.ComputeLabels(topic)
	if err != nil {
		log.Errorf("Error while compute labels on topic %s: %s", topic.Topic, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "error while compute labels on topic " + topic.Topic})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"info": fmt.Sprintf("%d labels computed", nbComputed)})
}

// TruncateTags clear tags on one topic
func (t *TopicsController) TruncateTags(ctx *gin.Context) {
	var paramJSON tat.TopicNameJSON
	ctx.Bind(&paramJSON)
	topic, e := t.preCheckUserAdminOnTopic(ctx, paramJSON.Topic)
	if e != nil {
		ctx.JSON(tat.Error(e))
		return
	}

	if err := topicDB.TruncateTags(topic); err != nil {
		log.Errorf("Error while clear tags on topic %s: %s", topic.Topic, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "error while clear tags on topic " + topic.Topic})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"info": fmt.Sprintf("%d tags cleared", len(topic.Tags))})
}

// TruncateLabels clear labels on one topic
func (t *TopicsController) TruncateLabels(ctx *gin.Context) {
	var paramJSON tat.TopicNameJSON
	ctx.Bind(&paramJSON)
	topic, e := t.preCheckUserAdminOnTopic(ctx, paramJSON.Topic)
	if e != nil {
		ctx.JSON(tat.Error(e))
		return
	}

	if err := topicDB.TruncateLabels(topic); err != nil {
		log.Errorf("Error while clear labels on topic %s: %s", topic.Topic, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "error while clear labels on topic " + topic.Topic})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"info": fmt.Sprintf("%d labels cleared", len(topic.Labels))})
}

// preCheckUser checks if user in paramJSON exists and if current user is admin on topic
func (t *TopicsController) preCheckUser(ctx *gin.Context, paramJSON *tat.ParamTopicUserJSON) (*tat.Topic, error) {
	user := tat.User{}
	found, err := userDB.FindByUsername(&user, paramJSON.Username)
	if !found {
		e := errors.New("username " + paramJSON.Username + " does not exist")
		ctx.AbortWithError(http.StatusInternalServerError, e)
		return nil, e
	} else if err != nil {
		e := errors.New("Error while fetching username username " + paramJSON.Username)
		ctx.AbortWithError(http.StatusInternalServerError, e)
		return nil, e
	}
	return t.preCheckUserAdminOnTopic(ctx, paramJSON.Topic)
}

// preCheckGroup checks if group exists and is admin on topic
func (t *TopicsController) preCheckGroup(ctx *gin.Context, paramJSON *tat.ParamTopicGroupJSON) (*tat.Topic, error) {
	if groupExists := groupDB.IsGroupnameExists(paramJSON.Groupname); !groupExists {
		return nil, tat.NewError(http.StatusNotFound, "groupname %s does not exist", paramJSON.Groupname)
	}
	return t.preCheckUserAdminOnTopic(ctx, paramJSON.Topic)
}

func (t *TopicsController) preCheckUserAdminOnTopic(ctx *gin.Context, topicName string) (*tat.Topic, error) {

	topic, errfind := topicDB.FindByTopic(topicName, true, false, false, nil)
	if errfind != nil {
		e := errors.New(errfind.Error())
		return nil, e
	}

	if isTatAdmin(ctx) { // if Tat admin, ok
		return topic, nil
	}

	user, err := PreCheckUser(ctx)
	if err != nil {
		return nil, err
	}

	if !topicDB.IsUserAdmin(topic, &user) {
		e := fmt.Errorf("user %s is not admin on topic %s", user.Username, topic.Topic)
		ctx.JSON(http.StatusForbidden, gin.H{"error": e})
		return nil, e
	}

	return topic, nil
}

// AddRoUser add a readonly user on selected topic
func (t *TopicsController) AddRoUser(ctx *gin.Context) {
	var paramJSON tat.ParamTopicUserJSON
	ctx.Bind(&paramJSON)
	topic, e := t.preCheckUser(ctx, &paramJSON)
	if e != nil {
		return
	}
	err := topicDB.AddRoUser(topic, getCtxUsername(ctx), paramJSON.Username, paramJSON.Recursive)
	if err != nil {
		log.Errorf("Error while adding read only user: %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, "")
}

// AddRwUser add a read / write user on selected topic
func (t *TopicsController) AddRwUser(ctx *gin.Context) {
	var paramJSON tat.ParamTopicUserJSON
	ctx.Bind(&paramJSON)
	topic, e := t.preCheckUser(ctx, &paramJSON)
	if e != nil {
		return
	}

	err := topicDB.AddRwUser(topic, getCtxUsername(ctx), paramJSON.Username, paramJSON.Recursive)
	if err != nil {
		log.Errorf("Error while adding read write user: %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, "")
}

// AddAdminUser add an admin user on selected topic
func (t *TopicsController) AddAdminUser(ctx *gin.Context) {
	var paramJSON tat.ParamTopicUserJSON
	ctx.Bind(&paramJSON)
	topic, e := t.preCheckUser(ctx, &paramJSON)
	if e != nil {
		return
	}

	if err := topicDB.AddAdminUser(topic, getCtxUsername(ctx), paramJSON.Username, paramJSON.Recursive); err != nil {
		log.Errorf("Error while adding admin user: %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, "")
}

// RemoveRoUser removes a readonly user on selected topic
func (t *TopicsController) RemoveRoUser(ctx *gin.Context) {
	var paramJSON tat.ParamTopicUserJSON
	ctx.Bind(&paramJSON)
	topic, e := t.preCheckUser(ctx, &paramJSON)
	if e != nil {
		return
	}

	err := topicDB.RemoveRoUser(topic, getCtxUsername(ctx), paramJSON.Username, paramJSON.Recursive)
	if err != nil {
		log.Errorf("Error while removing read only user: %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, "")
}

// RemoveRwUser removes a read / write user on selected topic
func (t *TopicsController) RemoveRwUser(ctx *gin.Context) {
	var paramJSON tat.ParamTopicUserJSON
	ctx.Bind(&paramJSON)
	topic, e := t.preCheckUser(ctx, &paramJSON)
	if e != nil {
		return
	}

	if err := topicDB.RemoveRwUser(topic, getCtxUsername(ctx), paramJSON.Username, paramJSON.Recursive); err != nil {
		log.Errorf("Error while removing read write user: %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, "")
}

// RemoveAdminUser removes an admin user on selected topic
func (t *TopicsController) RemoveAdminUser(ctx *gin.Context) {
	var paramJSON tat.ParamTopicUserJSON
	ctx.Bind(&paramJSON)
	topic, e := t.preCheckUser(ctx, &paramJSON)
	if e != nil {
		return
	}

	if err := topicDB.RemoveAdminUser(topic, getCtxUsername(ctx), paramJSON.Username, paramJSON.Recursive); err != nil {
		log.Errorf("Error while removing admin user: %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, "")
}

// AddRoGroup add a readonly group on selected topic
func (t *TopicsController) AddRoGroup(ctx *gin.Context) {
	var paramJSON tat.ParamTopicGroupJSON
	ctx.Bind(&paramJSON)
	topic, e := t.preCheckGroup(ctx, &paramJSON)
	if e != nil {
		ctx.JSON(tat.Error(e))
		return
	}
	if err := topicDB.AddRoGroup(topic, getCtxUsername(ctx), paramJSON.Groupname, paramJSON.Recursive); err != nil {
		log.Errorf("Error while adding admin read only group: %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, "")
}

// AddRwGroup add a read write group on selected topic
func (t *TopicsController) AddRwGroup(ctx *gin.Context) {
	var paramJSON tat.ParamTopicGroupJSON
	ctx.Bind(&paramJSON)
	topic, e := t.preCheckGroup(ctx, &paramJSON)
	if e != nil {
		ctx.JSON(tat.Error(e))
		return
	}

	if err := topicDB.AddRwGroup(topic, getCtxUsername(ctx), paramJSON.Groupname, paramJSON.Recursive); err != nil {
		log.Errorf("Error while adding admin read write group: %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, "")
}

// AddAdminGroup add an admin group on selected topic
func (t *TopicsController) AddAdminGroup(ctx *gin.Context) {
	var paramJSON tat.ParamTopicGroupJSON
	ctx.Bind(&paramJSON)
	topic, e := t.preCheckGroup(ctx, &paramJSON)
	if e != nil {
		ctx.JSON(tat.Error(e))
		return
	}

	if err := topicDB.AddAdminGroup(topic, getCtxUsername(ctx), paramJSON.Groupname, paramJSON.Recursive); err != nil {
		log.Errorf("Error while adding admin admin group: %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, "")
}

// AddParameter add a parameter on selected topic
func (t *TopicsController) AddParameter(ctx *gin.Context) {
	var topicParameterBind tat.TopicParameterJSON
	ctx.Bind(&topicParameterBind)
	topic, e := t.preCheckUserAdminOnTopic(ctx, topicParameterBind.Topic)
	if e != nil {
		ctx.JSON(tat.Error(e))
		return
	}

	err := topicDB.AddParameter(topic, getCtxUsername(ctx), topicParameterBind.Key, topicParameterBind.Value, topicParameterBind.Recursive)
	if err != nil {
		log.Errorf("Error while adding parameter: %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, "")
}

// RemoveParameter add a parameter on selected topic
func (t *TopicsController) RemoveParameter(ctx *gin.Context) {
	var topicParameterBind tat.TopicParameterJSON
	ctx.Bind(&topicParameterBind)

	topic, e := t.preCheckUserAdminOnTopic(ctx, topicParameterBind.Topic)
	if e != nil {
		ctx.JSON(tat.Error(e))
		return
	}

	err := topicDB.RemoveParameter(topic, getCtxUsername(ctx), topicParameterBind.Key, topicParameterBind.Value, topicParameterBind.Recursive)
	if err != nil {
		log.Errorf("Error while removing parameter: %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, "")
}

// AddFilter add a filter on selected topic
func (t *TopicsController) AddFilter(ctx *gin.Context) {
	var topicFilterBind tat.Filter
	if err := ctx.Bind(&topicFilterBind); err != nil {
		log.Errorf("AddFilter err:%s", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Post"})
		return
	}

	if c, e := checkFilter(&topicFilterBind); e != nil {
		ctx.JSON(c, gin.H{"error": e.Error()})
		return
	}

	out, user, code, err := t.innerOneTopic(ctx, topicFilterBind.Topic)
	if err != nil {
		ctx.JSON(code, gin.H{"error": err.Error()})
		return
	}

	if err := topicDB.AddFilter(out.Topic, user, &topicFilterBind); err != nil {
		log.Errorf("Error while adding filter: %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"info": "filter added on topic", "filter": topicFilterBind})
}

func checkFilter(f *tat.Filter) (int, error) {
	if f.Title == "" {
		return http.StatusBadRequest, fmt.Errorf("Filter: Title is mandatory")
	}
	if f.Criteria.FilterCriteriaIsEmpty() {
		return http.StatusBadRequest, fmt.Errorf("Filter: A criteria is mandatory")
	}
	for _, h := range f.Hooks {
		if h.Type != tat.HookTypeWebHook && h.Type != tat.HookTypeXMPPOut {
			return http.StatusBadRequest, fmt.Errorf("Filter: Invalid hook, only tathook-webhook and tathook-xmpp-out are valid")
		}
		if h.Destination == "" || h.Type == "" || h.Action == "" {
			return http.StatusBadRequest, fmt.Errorf("Filter: Invalid hook, destination and action are mandatory")
		}
		if h.Item == "" {
			h.Item = "message"
		}
	}
	return -1, nil
}

// RemoveFilter add a filter on selected topic
func (t *TopicsController) RemoveFilter(ctx *gin.Context) {
	var topicFilterBind tat.Filter
	if err := ctx.Bind(&topicFilterBind); err != nil {
		log.Errorf("RemoveFilter err:%s", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Post"})
		return
	}

	out, user, code, err := t.innerOneTopic(ctx, topicFilterBind.Topic)
	if err != nil {
		ctx.JSON(code, gin.H{"error": err.Error()})
		return
	}

	if topicFilterBind.ID == "" {
		ctx.JSON(code, gin.H{"error": "invalid filter id"})
		return
	}

	for _, f := range out.Topic.Filters {
		if f.ID == topicFilterBind.ID && f.UserID != user.ID {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "You can not remove a filter which not belong to you"})
			return
		}
	}

	if err := topicDB.RemoveFilter(out.Topic, &topicFilterBind); err != nil {
		log.Errorf("Error while removing filter: %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"info": "filter removed from topic", "filter": topicFilterBind})
}

// UpdateFilter add a filter on selected topic
func (t *TopicsController) UpdateFilter(ctx *gin.Context) {
	var topicFilterBind tat.Filter
	if err := ctx.Bind(&topicFilterBind); err != nil {
		log.Errorf("UpdateFilter err:%s", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Post"})
		return
	}

	log.Warnf("topicFilterBind: %+v", topicFilterBind)
	if c, e := checkFilter(&topicFilterBind); e != nil {
		ctx.JSON(c, gin.H{"error": e.Error()})
		return
	}

	out, user, code, err := t.innerOneTopic(ctx, topicFilterBind.Topic)
	if err != nil {
		ctx.JSON(code, gin.H{"error": err.Error()})
		return
	}

	for _, f := range out.Topic.Filters {
		if f.ID == topicFilterBind.ID && f.UserID != user.ID {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "You can not update a filter which not belong to you"})
			return
		}
	}

	topicFilterBind.UserID = user.ID // userID is not sent by UI
	topicFilterBind.Username = user.Username

	if err := topicDB.UpdateFilter(out.Topic, &topicFilterBind); err != nil {
		log.Errorf("Error while updating filter on topic %s err: %s", out.Topic.Topic, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"info": "filter updated on topic", "filter": topicFilterBind})
}

// RemoveRoGroup removes a read only group on selected topic
func (t *TopicsController) RemoveRoGroup(ctx *gin.Context) {
	var paramJSON tat.ParamTopicGroupJSON
	ctx.Bind(&paramJSON)
	topic, e := t.preCheckGroup(ctx, &paramJSON)
	if e != nil {
		ctx.JSON(tat.Error(e))
		return
	}

	err := topicDB.RemoveRoGroup(topic, getCtxUsername(ctx), paramJSON.Groupname, paramJSON.Recursive)
	if err != nil {
		log.Errorf("Error while removing read only group: %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, "")
}

// RemoveRwGroup removes a read write group on selected topic
func (t *TopicsController) RemoveRwGroup(ctx *gin.Context) {
	var paramJSON tat.ParamTopicGroupJSON
	ctx.Bind(&paramJSON)
	topic, e := t.preCheckGroup(ctx, &paramJSON)
	if e != nil {
		ctx.JSON(tat.Error(e))
		return
	}

	err := topicDB.RemoveRwGroup(topic, getCtxUsername(ctx), paramJSON.Groupname, paramJSON.Recursive)
	if err != nil {
		log.Errorf("Error while removing read write group: %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, "")
}

// RemoveAdminGroup removes an admin group on selected topic
func (t *TopicsController) RemoveAdminGroup(ctx *gin.Context) {
	var paramJSON tat.ParamTopicGroupJSON
	ctx.Bind(&paramJSON)
	topic, e := t.preCheckGroup(ctx, &paramJSON)
	if e != nil {
		ctx.JSON(tat.Error(e))
		return
	}

	err := topicDB.RemoveAdminGroup(topic, getCtxUsername(ctx), paramJSON.Groupname, paramJSON.Recursive)
	if err != nil {
		log.Errorf("Error while removing admin group: %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, "")
}

type paramsJSON struct {
	Topic                string               `json:"topic"`
	MaxLength            int                  `json:"maxlength"`
	MaxReplies           int                  `json:"maxreplies"`
	CanForceDate         bool                 `json:"canForceDate"`
	CanUpdateMsg         bool                 `json:"canUpdateMsg"`
	CanDeleteMsg         bool                 `json:"canDeleteMsg"`
	CanUpdateAllMsg      bool                 `json:"canUpdateAllMsg"`
	CanDeleteAllMsg      bool                 `json:"canDeleteAllMsg"`
	AdminCanUpdateAllMsg bool                 `json:"adminCanUpdateAllMsg"`
	AdminCanDeleteAllMsg bool                 `json:"adminCanDeleteAllMsg"`
	IsAutoComputeTags    bool                 `json:"isAutoComputeTags"`
	IsAutoComputeLabels  bool                 `json:"isAutoComputeLabels"`
	Recursive            bool                 `json:"recursive"`
	Parameters           []tat.TopicParameter `json:"parameters"`
}

// SetParam update Topic Parameters : MaxLength, MaxReplies, CanForeceDate, CanUpdateMsg, CanDeleteMsg, CanUpdateAllMsg, CanDeleteAllMsg, AdminCanDeleteAllMsg
// admin only, except on Private topic
func (t *TopicsController) SetParam(ctx *gin.Context) {
	var paramsBind paramsJSON
	ctx.Bind(&paramsBind)

	topic := &tat.Topic{}
	var errFind, err error
	if strings.HasPrefix(paramsBind.Topic, "/Private/"+getCtxUsername(ctx)) {
		topic, errFind = topicDB.FindByTopic(paramsBind.Topic, false, false, false, nil)
		if errFind != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error while fetching topic /Private/" + getCtxUsername(ctx)})
			return
		}
	} else {
		topic, err = t.preCheckUserAdminOnTopic(ctx, paramsBind.Topic)
		if err != nil {
			ctx.JSON(tat.Error(err))
			return
		}
	}

	for _, p := range paramsBind.Parameters {
		if strings.HasPrefix(p.Key, tat.HookTypeKafka) && !isTatAdmin(ctx) {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Only Tat Admin can use tathook-kafka"})
			return
		}
	}

	err = topicDB.SetParam(topic, getCtxUsername(ctx),
		paramsBind.Recursive,
		paramsBind.MaxLength,
		paramsBind.MaxReplies,
		paramsBind.CanForceDate,
		paramsBind.CanUpdateMsg,
		paramsBind.CanDeleteMsg,
		paramsBind.CanUpdateAllMsg,
		paramsBind.CanDeleteAllMsg,
		paramsBind.AdminCanUpdateAllMsg,
		paramsBind.AdminCanDeleteAllMsg,
		paramsBind.IsAutoComputeTags,
		paramsBind.IsAutoComputeLabels,
		paramsBind.Parameters)

	// add tat2xmpp_username RO or RW on this topic if a key is xmpp
	for _, p := range paramsBind.Parameters {
		if strings.HasPrefix(p.Key, tat.HookTypeXMPPOut) {
			found := false
			for _, u := range topic.ROUsers {
				if u == viper.GetString("tat2xmpp_username") {
					found = true
				}
			}
			for _, u := range topic.RWUsers {
				if u == viper.GetString("tat2xmpp_username") {
					found = true
				}
			}
			if !found {
				if errf := topicDB.AddRoUser(topic, getCtxUsername(ctx), viper.GetString("tat2xmpp_username"), false); errf != nil {
					log.Errorf("Error while adding read only user tat2xmpp_username: %s", errf)
					ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
		} else if strings.HasPrefix(p.Key, tat.HookTypeXMPP) {
			found := false
			for _, u := range topic.RWUsers {
				if u == viper.GetString("tat2xmpp_username") {
					found = true
				}
			}
			if !found {
				if errf := topicDB.AddRwUser(topic, getCtxUsername(ctx), viper.GetString("tat2xmpp_username"), false); errf != nil {
					log.Errorf("Error while adding read write user tat2xmpp_username: %s", errf)
					ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
		}
	}

	if err != nil {
		log.Errorf("Error while setting parameters: %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"info": fmt.Sprintf("Topic %s updated", topic.Topic)})
}

// AllComputeTags Compute tags on all topics
func (t *TopicsController) AllComputeTags(ctx *gin.Context) {
	// It's only for admin, admin already checked in route
	info, err := topicDB.AllTopicsComputeTags()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"info": info})
}

// AllComputeLabels Compute tags on all topics
func (t *TopicsController) AllComputeLabels(ctx *gin.Context) {
	// It's only for admin, admin already checked in route
	info, err := topicDB.AllTopicsComputeLabels()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"info": info})
}

// AllSetParam set a param on all topics
func (t *TopicsController) AllSetParam(ctx *gin.Context) {
	// It's only for admin, admin already checked in route
	var param tat.ParamJSON
	ctx.Bind(&param)

	info, err := topicDB.AllTopicsSetParam(param.ParamName, param.ParamValue)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"info": info})
}

// AllComputeReplies computes replies on all topics
func (t *TopicsController) AllComputeReplies(ctx *gin.Context) {
	// It's only for admin, admin already checked in route
	var param tat.ParamJSON
	ctx.Bind(&param)

	info, err := messageDB.AllTopicsComputeReplies()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"info": info})
}

// MigrateToDedicatedTopic migrates a topic to dedicated collection on mongo
func (t *TopicsController) MigrateToDedicatedTopic(ctx *gin.Context) {
	topicRequest, err := GetParam(ctx, "topic")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	topic, errfind := topicDB.FindByTopic(topicRequest, true, false, false, nil)
	if errfind != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if errMigrate := topicDB.MigrateToDedicatedTopic(topic); errMigrate != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": errMigrate.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"info": fmt.Sprintf("%s is now dedicated", topicRequest)})
}

// MigrateMessagesForDedicatedTopic migrates all msg of a topic to a dedicated collection
func (t *TopicsController) MigrateMessagesForDedicatedTopic(ctx *gin.Context) {
	slimit, e1 := GetParam(ctx, "limit")
	if e1 != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": e1.Error()})
		return
	}

	limit, e2 := strconv.Atoi(slimit)
	if e2 != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": e2.Error()})
		return
	}

	topicRequest, err := GetParam(ctx, "topic")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	topic, errfind := topicDB.FindByTopic(topicRequest, true, false, false, nil)
	if errfind != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": errfind.Error()})
		return
	}

	nMigrate, err := messageDB.MigrateMessagesToDedicatedTopic(topic, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error after %d migrate, err:%s", nMigrate, err.Error())})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"info": fmt.Sprintf("No error after migrate %d messages (%d asked for migrate)", nMigrate, limit)})
}
