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
	"github.com/ovh/tat/api/hook"
	messageDB "github.com/ovh/tat/api/message"
	presenceDB "github.com/ovh/tat/api/presence"
	topicDB "github.com/ovh/tat/api/topic"
	userDB "github.com/ovh/tat/api/user"
)

// MessagesController contains all methods about messages manipulation
type MessagesController struct{}

func (*MessagesController) buildCriteria(ctx *gin.Context) *tat.MessageCriteria {
	c := tat.MessageCriteria{}
	skip, e := strconv.Atoi(ctx.DefaultQuery("skip", "0"))
	if e != nil {
		skip = 0
	}
	c.Skip = skip
	limit, e2 := strconv.Atoi(ctx.DefaultQuery("limit", "100"))
	if e2 != nil || limit <= 0 {
		limit = 10
	}

	c.Limit = limit
	c.TreeView = ctx.Query("treeView")
	c.IDMessage = ctx.Query("idMessage")
	c.InReplyOfID = ctx.Query("inReplyOfID")
	c.InReplyOfIDRoot = ctx.Query("inReplyOfIDRoot")
	c.AllIDMessage = ctx.Query("allIDMessage")
	c.Text = ctx.Query("text")
	c.Label = ctx.Query("label")
	c.StartLabel = ctx.Query("startLabel")
	c.NotLabel = ctx.Query("notLabel")
	c.AndLabel = ctx.Query("andLabel")
	c.StartTag = ctx.Query("startTag")
	c.Tag = ctx.Query("tag")
	c.NotTag = ctx.Query("notTag")
	c.AndTag = ctx.Query("andTag")
	c.DateMinCreation = ctx.Query("dateMinCreation")
	c.DateMaxCreation = ctx.Query("dateMaxCreation")
	c.DateMinUpdate = ctx.Query("dateMinUpdate")
	c.DateMaxUpdate = ctx.Query("dateMaxUpdate")
	c.LastMinCreation = ctx.Query("lastMinCreation")
	c.LastMaxCreation = ctx.Query("lastMaxCreation")
	c.LastMinUpdate = ctx.Query("lastMinUpdate")
	c.LastMaxUpdate = ctx.Query("lastMaxUpdate")
	c.LastHourMinCreation = ctx.Query("lastHourMinCreation")
	c.LastHourMaxCreation = ctx.Query("lastHourMaxCreation")
	c.LastHourMinUpdate = ctx.Query("lastHourMinUpdate")
	c.LastHourMaxUpdate = ctx.Query("lastHourMaxUpdate")
	c.DateRefCreation = ctx.Query("dateRefCreation")
	c.DateRefDeltaMinCreation = ctx.Query("dateRefDeltaMinCreation")
	c.DateRefDeltaMaxCreation = ctx.Query("dateRefDeltaMaxCreation")
	c.DateRefUpdate = ctx.Query("dateRefUpdate")
	c.DateRefDeltaMinUpdate = ctx.Query("dateRefDeltaMinUpdate")
	c.DateRefDeltaMaxUpdate = ctx.Query("dateRefDeltaMaxUpdate")
	c.LimitMinNbVotesUP = ctx.Query("limitMinNbVotesUP")
	c.LimitMaxNbVotesUP = ctx.Query("limitMaxNbVotesUP")
	c.LimitMinNbVotesDown = ctx.Query("limitMinNbVotesDown")
	c.LimitMaxNbVotesDown = ctx.Query("limitMaxNbVotesDown")
	c.Username = ctx.Query("username")
	c.LimitMinNbReplies = ctx.Query("limitMinNbReplies")
	c.LimitMaxNbReplies = ctx.Query("limitMaxNbReplies")
	c.OnlyMsgRoot = ctx.Query("onlyMsgRoot")
	c.OnlyMsgReply = ctx.Query("onlyMsgReply")
	c.OnlyCount = ctx.Query("onlyCount")
	c.SortBy = ctx.Query("sortBy")
	return &c
}

// List messages on one topic, with given criteria
func (m *MessagesController) List(ctx *gin.Context) {
	out, user, topic, criteria, httpCode, err := m.innerList(ctx)

	if err != nil {
		ctx.JSON(httpCode, gin.H{"error": err.Error()})
		return
	}

	if criteria.OnlyCount == tat.True {
		count, e := messageDB.CountMessages(criteria, topic)
		if e != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})
			return
		}
		ctx.JSON(http.StatusOK, &tat.MessagesCountJSON{Count: count})
		return
	}

	// send presence
	presenceArg := ctx.Query("presence")
	if presenceArg != "" && !user.IsSystem {
		go func() {
			var presence = tat.Presence{}
			if e := presenceDB.Upsert(&presence, user, topic, presenceArg); e != nil {
				log.Errorf("Error while InsertPresence %s", e)
			}
		}()
	}

	messages, err := messageDB.ListMessages(criteria, user.Username, topic)
	if err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	}
	out.Messages = messages
	ctx.JSON(http.StatusOK, out)

}

func (m *MessagesController) innerList(ctx *gin.Context) (*tat.MessagesJSON, tat.User, tat.Topic, *tat.MessageCriteria, int, error) {
	var criteria = m.buildCriteria(ctx)

	if criteria.Limit <= 0 || criteria.Limit > 1000 {
		return nil, tat.User{}, tat.Topic{}, criteria, http.StatusBadRequest, fmt.Errorf("Please put a limit <= 50 for fetching message")
	}

	log.Debugf("criteria::---> %+v", criteria)
	out := &tat.MessagesJSON{}

	// we can't use NotLabel or NotTag with fulltree or onetree
	// this avoid potential wrong results associated with a short param limit
	if (criteria.NotLabel != "" || criteria.NotTag != "") &&
		(criteria.TreeView == tat.TreeViewFullTree || criteria.TreeView == tat.TreeViewOneTree) && criteria.OnlyMsgRoot == "" {
		return out, tat.User{}, tat.Topic{}, criteria, http.StatusBadRequest, fmt.Errorf("You can't use fulltree or onetree with NotLabel or NotTag")
	}

	topicIn, err := GetParam(ctx, "topic")
	if err != nil {
		return out, tat.User{}, tat.Topic{}, criteria, http.StatusBadRequest, fmt.Errorf("Invalid topic")
	}
	criteria.Topic = topicIn

	// add / if search on topic
	// as topic is in path, it can't start with a /
	if criteria.Topic != "" && string(criteria.Topic[0]) != "/" {
		criteria.Topic = "/" + criteria.Topic
	}

	var user tat.User
	var e error

	if getCtxUsername(ctx) != "" {
		user, e = PreCheckUser(ctx)
		if e != nil {
			return out, tat.User{}, tat.Topic{}, criteria, http.StatusBadRequest, e
		}
	}

	topic, errt := topicDB.FindByTopic(criteria.Topic, true, false, false, &user)
	if errt != nil {
		var topicCriteria string
		_, topicCriteria, err = checkDMTopic(ctx, criteria.Topic)
		if err != nil {
			return out, tat.User{}, tat.Topic{}, criteria, http.StatusBadRequest, fmt.Errorf("topic " + criteria.Topic + " does not exist or you have no read access on it")
		}
		// hack to get new created DM Topic
		topic, errt = topicDB.FindByTopic(topicCriteria, true, false, false, &user)
		if errt != nil {
			return out, tat.User{}, tat.Topic{}, criteria, http.StatusBadRequest, fmt.Errorf("topic " + criteria.Topic + " does not exist or you have no read access on it (2)")
		}
		criteria.Topic = topicCriteria
	}

	out.IsTopicRw, out.IsTopicAdmin = topicDB.GetUserRights(topic, &user)

	return out, user, *topic, criteria, -1, nil
}

func (m *MessagesController) preCheckTopic(ctx *gin.Context, messageIn *tat.MessageJSON) (tat.Message, tat.Topic, *tat.User, error) {
	var message = tat.Message{}

	user, e := PreCheckUser(ctx)
	if e != nil {
		return message, tat.Topic{}, nil, e
	}

	topicIn, err := GetParam(ctx, "topic")
	if err != nil {
		return message, tat.Topic{}, nil, err
	}
	messageIn.Topic = topicIn

	if messageIn.Topic == "/" && messageIn.IDReference != "" {
		log.Warnf("preCheckTopic fallback to FindByIDDefaultCollection for %s", messageIn.IDReference)
		if efind := messageDB.FindByIDDefaultCollection(&message, messageIn.IDReference); efind != nil {
			e := errors.New("Invalid request, no topic and message " + messageIn.IDReference + " not found in default collection")
			ctx.JSON(http.StatusNotFound, gin.H{"error": e.Error()})
			return message, tat.Topic{}, nil, e
		}
		messageIn.Topic = message.Topic
	}

	topic, efind := topicDB.FindByTopic(messageIn.Topic, true, true, true, &user)
	if efind != nil {
		topica, _, edm := checkDMTopic(ctx, messageIn.Topic)
		if edm != nil {
			e := errors.New("Topic " + messageIn.Topic + " does not exist or you have no read access on it")
			ctx.JSON(http.StatusNotFound, gin.H{"error": e.Error()})
			return message, tat.Topic{}, nil, e
		}
		topic = topica
	}

	if messageIn.IDReference == "" &&
		messageIn.TagReference == "" &&
		messageIn.StartTagReference == "" &&
		messageIn.LabelReference == "" &&
		messageIn.StartLabelReference == "" {
		// nothing here
	} else if messageIn.IDReference != "" ||
		messageIn.StartTagReference != "" || messageIn.TagReference != "" ||
		messageIn.StartLabelReference != "" || messageIn.LabelReference != "" {
		if messageIn.IDReference != "" {
			if efind := messageDB.FindByID(&message, messageIn.IDReference, *topic); efind != nil {
				e := errors.New("Message " + messageIn.IDReference + " does not exist or you have no read access on it")
				ctx.JSON(http.StatusNotFound, gin.H{"error": e.Error()})
				return message, tat.Topic{}, nil, e
			}
		} else { // TagReference, StartTagReference,LabelReference, StartLabelReference
			onlyMsgRoot := tat.True // default value must be true
			if messageIn.OnlyRootReference == tat.False {
				onlyMsgRoot = tat.False
			}
			c := &tat.MessageCriteria{
				AndTag:      messageIn.TagReference,
				StartTag:    messageIn.StartTagReference,
				AndLabel:    messageIn.LabelReference,
				StartLabel:  messageIn.StartLabelReference,
				OnlyMsgRoot: onlyMsgRoot,
				Topic:       topic.Topic,
			}
			mlist, efind := messageDB.ListMessages(c, user.Username, *topic)
			if efind != nil {
				e := errors.New("Searched Message does not exist or you have no read access on it")
				ctx.JSON(http.StatusNotFound, gin.H{"error": e.Error()})
				return message, tat.Topic{}, nil, e
			}

			if messageIn.Action == tat.MessageActionRelabelOrCreate {
				if len(mlist) == 1 {
					message = mlist[0]
				}
			} else if len(mlist) != 1 {
				if messageIn.Action != "" {
					e := fmt.Errorf("Searched Message, expected 1 message and %d message(s) matching on tat", len(mlist))
					ctx.JSON(http.StatusNotFound, gin.H{"error": e.Error()})
					return message, tat.Topic{}, nil, e
				}
				// take last root message
				if len(mlist) > 0 {
					message = mlist[0]
				}
			} else {
				message = mlist[0]
			}
		}

		topicName := ""
		if messageIn.Action == tat.MessageActionUpdate {
			topicName = messageIn.Topic
		} else if messageIn.Action == "" || messageIn.Action == tat.MessageActionReply ||
			messageIn.Action == tat.MessageActionLike || messageIn.Action == tat.MessageActionUnlike ||
			messageIn.Action == tat.MessageActionLabel || messageIn.Action == tat.MessageActionUnlabel ||
			messageIn.Action == tat.MessageActionVoteup || messageIn.Action == tat.MessageActionVotedown ||
			messageIn.Action == tat.MessageActionUnvoteup || messageIn.Action == tat.MessageActionUnvotedown ||
			messageIn.Action == tat.MessageActionRelabel || messageIn.Action == tat.MessageActionRelabelOrCreate ||
			messageIn.Action == tat.MessageActionConcat {
			topicName = m.inverseIfDMTopic(ctx, message.Topic)
		} else if messageIn.Action == tat.MessageActionMove {
			topicName = topicIn
		} else if messageIn.Action == tat.MessageActionTask || messageIn.Action == tat.MessageActionUntask {
			topicName = m.inverseIfDMTopic(ctx, message.Topic)
		} else {
			e := errors.New("Invalid Call. IDReference not empty with unknown action")
			ctx.JSON(http.StatusBadRequest, gin.H{"error": e.Error()})
			return message, tat.Topic{}, nil, e
		}
		if topicName == "" && (messageIn.Action == "" || messageIn.Action == tat.MessageActionRelabelOrCreate) {
			topicName = messageIn.Topic
		}

		topic, err = topicDB.FindByTopic(topicName, true, true, true, &user)
		if err != nil {
			e := errors.New("Topic '" + topicName + "' does not exist")
			ctx.JSON(http.StatusNotFound, gin.H{"error": e.Error()})
			return message, tat.Topic{}, nil, e
		}
	} else {
		e := errors.New("Topic and Reference (ID, StartTag, Tag, StartLabel, Label) are null. Wrong request")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": e.Error()})
		return message, tat.Topic{}, nil, e
	}
	return message, *topic, &user, nil
}

// CreateBulk creates messages on one topic
func (m *MessagesController) CreateBulk(ctx *gin.Context) {
	messagesIn := &tat.MessagesJSONIn{}
	ctx.Bind(messagesIn)
	var msgs []*tat.MessageJSONOut
	for _, messageIn := range messagesIn.Messages {
		m, code, err := m.createSingle(ctx, messageIn)
		if err != nil {
			ctx.JSON(code, gin.H{"error": err.Error()})
			return
		}
		msgs = append(msgs, m)
	}
	ctx.JSON(http.StatusCreated, msgs)
}

// Create a new message on one topic
func (m *MessagesController) Create(ctx *gin.Context) {
	messageIn := &tat.MessageJSON{}
	ctx.Bind(messageIn)
	out, code, err := m.createSingle(ctx, messageIn)
	if err != nil {
		ctx.JSON(code, gin.H{"error": err})
		return
	}
	ctx.JSON(code, out)
}

func (m *MessagesController) createSingle(ctx *gin.Context, messageIn *tat.MessageJSON) (*tat.MessageJSONOut, int, error) {

	msg, topic, user, e := m.preCheckTopic(ctx, messageIn)
	if e != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("No RW Access to topic %s", messageIn.Topic)
	}

	if isRw, _ := topicDB.GetUserRights(&topic, user); !isRw {
		return nil, http.StatusForbidden, fmt.Errorf("No RW Access to topic %s", messageIn.Topic)
	}

	var message = tat.Message{}

	idRef := ""
	if msg.ID != "" {
		idRef = msg.ID
	}

	text := messageIn.Text
	if idRef != "" && messageIn.Text != "" && (len(messageIn.Replies) > 0 || len(messageIn.Messages) > 0) {
		text = ""
	}

	// New root message or reply
	err := messageDB.Insert(&message, *user, topic, text, idRef, messageIn.DateCreation, messageIn.Labels, messageIn.Replies, messageIn.Messages, false, nil)
	if err != nil {
		log.Errorf("%s", err.Error())
		return nil, http.StatusInternalServerError, err
	}
	info := fmt.Sprintf("Message created in %s", topic.Topic)
	out := &tat.MessageJSONOut{Message: message, Info: info}
	hook.SendHook(&tat.HookJSON{HookMessage: &tat.HookMessageJSON{MessageJSONOut: out, Action: tat.MessageActionCreate}}, topic)
	return out, http.StatusCreated, nil
}

// Update a message : like, unlike, add label, etc...
func (m *MessagesController) Update(ctx *gin.Context) {
	messageIn := &tat.MessageJSON{}
	ctx.Bind(messageIn)
	messageReference, topic, user, e := m.preCheckTopic(ctx, messageIn)
	if e != nil {
		return
	}

	if messageIn.Action == "like" || messageIn.Action == "unlike" {
		m.likeOrUnlike(ctx, messageIn.Action, messageReference, topic, *user)
		return
	}

	isRW, isAdminOnTopic := topicDB.GetUserRights(&topic, user)
	if !isRW {
		ctx.AbortWithError(http.StatusForbidden, errors.New("No RW Access to topic : "+messageIn.Topic))
		return
	}

	if messageIn.Action == tat.MessageActionLabel || messageIn.Action == tat.MessageActionUnlabel ||
		messageIn.Action == tat.MessageActionRelabel || messageIn.Action == tat.MessageActionRelabelOrCreate {
		m.addOrRemoveLabel(ctx, messageIn, messageReference, *user, topic)
		return
	}

	if messageIn.Action == tat.MessageActionVoteup || messageIn.Action == tat.MessageActionVotedown ||
		messageIn.Action == tat.MessageActionUnvoteup || messageIn.Action == tat.MessageActionUnvotedown {
		m.voteMessage(ctx, messageIn, messageReference, *user, topic)
		return
	}

	if messageIn.Action == tat.MessageActionTask || messageIn.Action == tat.MessageActionUntask {
		m.addOrRemoveTask(ctx, messageIn, messageReference, *user, topic)
		return
	}

	if messageIn.Action == tat.MessageActionUpdate || messageIn.Action == tat.MessageActionConcat {
		m.updateMessage(ctx, messageIn, messageReference, *user, topic, isAdminOnTopic)
		return
	}

	if messageIn.Action == tat.MessageActionMove {
		// topic here is fromTopic
		m.moveMessage(ctx, messageIn, messageReference, *user, topic)
		return
	}

	ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Action"})
}

// Delete a message
func (m *MessagesController) Delete(ctx *gin.Context) {
	m.messageDelete(ctx, false, false)
}

// DeleteCascade deletes a message and its replies
func (m *MessagesController) DeleteCascade(ctx *gin.Context) {
	m.messageDelete(ctx, true, false)
}

// DeleteCascadeForce deletes a message and its replies, event if a msg is in a
// tasks topic of one user
func (m *MessagesController) DeleteCascadeForce(ctx *gin.Context) {
	m.messageDelete(ctx, true, true)
}

func (m *MessagesController) messageDelete(ctx *gin.Context, cascade, force bool) {
	idMessageIn, err := GetParam(ctx, "idMessage")
	if err != nil {
		return
	}

	topicIn, err := GetParam(ctx, "topic")
	if err != nil {
		return
	}

	user, e := PreCheckUser(ctx)
	if e != nil {
		return
	}

	topic, errf := topicDB.FindByTopic(topicIn, true, false, false, &user)
	if errf != nil {
		log.Errorf("messageDelete> err:%s", errf)
		e := fmt.Sprintf("Topic '%s' does not exist", topicIn)
		ctx.JSON(http.StatusNotFound, gin.H{"error": e})
		return
	}

	message := tat.Message{}
	if err = messageDB.FindByID(&message, idMessageIn, *topic); err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Message %s does not exist", idMessageIn)})
		return
	}

	err = m.checkBeforeDelete(ctx, message, user, force, *topic)
	if err != nil {
		// ctx writes in checkBeforeDelete
		return
	}

	c := &tat.MessageCriteria{
		InReplyOfID: message.ID,
		TreeView:    tat.TreeViewOneTree,
		Topic:       topic.Topic,
	}

	msgs, err := messageDB.ListMessages(c, "", *topic)
	if err != nil {
		log.Errorf("Error while list Messages in Delete %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error while list Messages in Delete"})
		return
	}

	if cascade {
		for _, r := range msgs {
			errCheck := m.checkBeforeDelete(ctx, r, user, force, *topic)
			if errCheck != nil {
				// ctx writes in checkBeforeDelete
				return
			}
		}
	} else if len(msgs) > 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Could not delete this message, this message have replies"})
		return
	}

	if err = messageDB.Delete(&message, cascade, *topic); err != nil {
		log.Errorf("Error while delete a message %s", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"info": fmt.Sprintf("Message deleted from %s", topic.Topic)})
}

// checkBeforeDelete checks
// - if user is RW on topic
// - if topic is Private OR is CanDeleteMsg or CanDeleteAllMsg
func (m *MessagesController) checkBeforeDelete(ctx *gin.Context, message tat.Message, user tat.User, force bool, topic tat.Topic) error {

	isRW, isTopicAdmin := topicDB.GetUserRights(&topic, &user)
	if !isRW {
		e := fmt.Sprintf("No RW Access to topic %s", message.Topic)
		ctx.JSON(http.StatusForbidden, gin.H{"error": e})
		return fmt.Errorf(e)
	}

	if topic.AdminCanDeleteAllMsg && isTopicAdmin {
		return nil
	}

	if !strings.HasPrefix(message.Topic, "/Private/"+user.Username) && !topic.CanDeleteMsg && !topic.CanDeleteAllMsg {
		if !topic.CanDeleteMsg && !topic.CanDeleteAllMsg {
			e := fmt.Sprintf("You can't delete a message from topic %s", topic.Topic)
			ctx.JSON(http.StatusForbidden, gin.H{"error": e})
			return fmt.Errorf(e)
		}
		e := fmt.Sprintf("Could not delete a message in topic %s", message.Topic)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": e})
		return fmt.Errorf(e)
	}

	if !topic.CanDeleteAllMsg && message.Author.Username != user.Username && !strings.HasPrefix(message.Topic, "/Private/"+user.Username) {
		// if it's a reply and force true, allow delete it.
		if !force || (force && message.InReplyOfIDRoot == "") {
			e := fmt.Sprintf("Could not delete a message from another user %s than you %s", message.Author.Username, user.Username)
			ctx.JSON(http.StatusBadRequest, gin.H{"error": e})
			return fmt.Errorf(e)
		}
	}

	// if label done on msg, can delete it
	if !force && message.IsDoing() {
		e := fmt.Sprintf("Could not delete a message with a doing label")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": e})
		return fmt.Errorf(e)
	}
	return nil
}

func (m *MessagesController) likeOrUnlike(ctx *gin.Context, action string, message tat.Message, topic tat.Topic, user tat.User) {

	info := ""
	if action == tat.MessageActionLike {
		if err := messageDB.Like(&message, user, topic); err != nil {
			log.Errorf("Error while like a message %s", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		info = "like added"
	} else if action == tat.MessageActionUnlike {
		if err := messageDB.Unlike(&message, user, topic); err != nil {
			log.Errorf("Error while unlike a message %s", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		info = "like removed"
	} else {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid action: " + action)})
		return
	}
	out := &tat.MessageJSONOut{Info: info, Message: message}
	hook.SendHook(&tat.HookJSON{HookMessage: &tat.HookMessageJSON{MessageJSONOut: out, Action: action}}, topic)
	ctx.JSON(http.StatusCreated, out)
}

func (m *MessagesController) addOrRemoveLabel(ctx *gin.Context, messageIn *tat.MessageJSON, message tat.Message, user tat.User, topic tat.Topic) {
	if messageIn.Text == "" && messageIn.Action != tat.MessageActionRelabel {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Invalid Text for label"))
		return
	}
	out := &tat.MessageJSONOut{}
	if messageIn.Action == tat.MessageActionLabel {
		addedLabel, err := messageDB.AddLabel(&message, topic, messageIn.Text, messageIn.Option)
		if err != nil {
			errInfo := fmt.Sprintf("Error while adding a label to a message %s", err.Error())
			log.Errorf(errInfo)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": errInfo})
			return
		}
		out = &tat.MessageJSONOut{Info: fmt.Sprintf("label %s added to message", addedLabel.Text), Message: message}
	} else if messageIn.Action == tat.MessageActionUnlabel {
		if err := messageDB.RemoveLabel(&message, messageIn.Text, topic); err != nil {
			errInfo := fmt.Sprintf("Error while removing a label from a message %s", err.Error())
			log.Errorf(errInfo)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": errInfo})
			return
		}
		out = &tat.MessageJSONOut{Info: fmt.Sprintf("label %s removed from message", messageIn.Text), Message: message}
	} else if messageIn.Action == tat.MessageActionRelabelOrCreate && len(messageIn.Options) == 0 {
		if message.ID != "" {
			if err := messageDB.RemoveAllAndAddNewLabel(&message, messageIn.Labels, topic); err != nil {
				errInfo := fmt.Sprintf("Error while removing all labels and add new ones for a message %s", err.Error())
				log.Errorf(errInfo)
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": errInfo})
				return
			}
			out = &tat.MessageJSONOut{Info: fmt.Sprintf("all labels removed and new labels %s added to message", messageIn.Text), Message: message}
		} else {
			// create new message
			var code int
			var errCreate error
			out, code, errCreate = m.createSingle(ctx, messageIn)
			if errCreate != nil {
				ctx.JSON(code, gin.H{"error": errCreate})
				return
			}
		}
	} else if messageIn.Action == tat.MessageActionRelabel && len(messageIn.Options) == 0 {
		if err := messageDB.RemoveAllAndAddNewLabel(&message, messageIn.Labels, topic); err != nil {
			errInfo := fmt.Sprintf("Error while removing all labels and add new ones for a message %s", err.Error())
			log.Errorf(errInfo)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": errInfo})
			return
		}
		out = &tat.MessageJSONOut{Info: fmt.Sprintf("all labels removed and new labels %s added to message", messageIn.Text), Message: message}
	} else if messageIn.Action == tat.MessageActionRelabel && len(messageIn.Options) > 0 {
		if err := messageDB.RemoveSomeAndAddNewLabel(&message, messageIn.Labels, messageIn.Options, topic); err != nil {
			errInfo := fmt.Sprintf("Error while removing some labels and add new ones for a message %s", err.Error())
			log.Errorf(errInfo)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": errInfo})
			return
		}
		out = &tat.MessageJSONOut{Info: fmt.Sprintf("Some labels removed and new labels %s added to message", messageIn.Text), Message: message}

	} else {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Invalid action: "+messageIn.Action))
		return
	}
	hook.SendHook(&tat.HookJSON{HookMessage: &tat.HookMessageJSON{MessageJSONOut: out, Action: messageIn.Action}}, topic)
	ctx.JSON(http.StatusCreated, out)
}

func (m *MessagesController) voteMessage(ctx *gin.Context, messageIn *tat.MessageJSON, message tat.Message, user tat.User, topic tat.Topic) {
	info := ""
	errInfo := ""
	if messageIn.Action == tat.MessageActionVoteup {
		if err := messageDB.VoteUP(&message, user, topic); err != nil {
			errInfo = fmt.Sprintf("Error while vote up a message %s", err.Error())
		}
		info = "Vote UP added to message"
	} else if messageIn.Action == tat.MessageActionVotedown {
		if err := messageDB.VoteDown(&message, user, topic); err != nil {
			errInfo = fmt.Sprintf("Error while vote down a message %s", err.Error())
		}
		info = "Vote Down added to message"
	} else if messageIn.Action == tat.MessageActionUnvoteup {
		if err := messageDB.UnVoteUP(&message, user, topic); err != nil {
			errInfo = fmt.Sprintf("Error while remove vote up from message %s", err.Error())
		}
		info = "Vote UP removed from message"
	} else if messageIn.Action == tat.MessageActionUnvotedown {
		if err := messageDB.UnVoteDown(&message, user, topic); err != nil {
			errInfo = fmt.Sprintf("Error while remove vote down from message %s", err.Error())
		}
		info = "Vote Down removed from message"
	} else {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("Invalid action: "+messageIn.Action))
		return
	}
	if errInfo != "" {
		log.Errorf(errInfo)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": errInfo})
		return
	}
	if err := messageDB.FindByID(&message, messageIn.IDReference, topic); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error while fetching message after voting"})
		return
	}

	out := &tat.MessageJSONOut{Info: info, Message: message}
	hook.SendHook(&tat.HookJSON{HookMessage: &tat.HookMessageJSON{MessageJSONOut: out, Action: messageIn.Action}}, topic)
	ctx.JSON(http.StatusCreated, out)
}

func (m *MessagesController) addOrRemoveTask(ctx *gin.Context, messageIn *tat.MessageJSON, message tat.Message, user tat.User, topic tat.Topic) {
	info := ""
	if messageIn.Action == tat.MessageActionTask {
		if message.InReplyOfIDRoot != "" {
			log.Warnf("This message is a reply, you can't task it (%s)", message.ID)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "This message is a reply, you can't task it"})
			return
		}
		if err := messageDB.AddToTasks(&message, user, topic); err != nil {
			log.Errorf("Error while adding a message to tasks %s", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error while adding a message to tasks"})
			return
		}
		info = fmt.Sprintf("New Task created")
	} else if messageIn.Action == tat.MessageActionUntask {
		if err := messageDB.RemoveFromTasks(&message, user, topic); err != nil {
			log.Errorf("Error while removing a message from tasks %s", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		info = fmt.Sprintf("Task removed")
	} else {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action: " + messageIn.Action})
		return
	}
	out := &tat.MessageJSONOut{Info: info, Message: message}
	hook.SendHook(&tat.HookJSON{HookMessage: &tat.HookMessageJSON{MessageJSONOut: out, Action: messageIn.Action}}, topic)
	ctx.JSON(http.StatusCreated, out)
}

func (m *MessagesController) updateMessage(ctx *gin.Context, messageIn *tat.MessageJSON, message tat.Message, user tat.User, topic tat.Topic, isAdminOnTopic bool) {
	var info string

	if isAdminOnTopic && topic.CanUpdateAllMsg {
		// ok, user is admin on topic, and admin can update all msg
	} else {
		if !topic.CanUpdateMsg && !topic.CanUpdateAllMsg {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("You can't update a message on topic %s", topic.Topic)})
			return
		}

		if !topic.CanUpdateAllMsg && message.Author.Username != user.Username {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Could not update a message from another user %s than you %s", message.Author.Username, user.Username)})
			return
		}
	}

	if err := messageDB.Update(&message, user, topic, messageIn.Text, messageIn.Action); err != nil {
		log.Errorf("Error while update a message %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	info = fmt.Sprintf("Message updated in %s", topic.Topic)
	out := &tat.MessageJSONOut{Info: info, Message: message}
	hook.SendHook(&tat.HookJSON{HookMessage: &tat.HookMessageJSON{MessageJSONOut: out, Action: messageIn.Action}}, topic)
	ctx.JSON(http.StatusOK, out)
}

func (m *MessagesController) moveMessage(ctx *gin.Context, messageIn *tat.MessageJSON, message tat.Message, user tat.User, fromTopic tat.Topic) {

	// Check if user can delete msg on from topic
	if err := m.checkBeforeDelete(ctx, message, user, true, fromTopic); err != nil {
		// ctx writes in checkBeforeDelete
		return
	}

	toTopic, err := topicDB.FindByTopic(messageIn.Option, true, false, false, &user)
	if err != nil {
		e := fmt.Sprintf("Topic destination %s does not exist", message.Topic)
		ctx.JSON(http.StatusNotFound, gin.H{"error": e})
		return
	}

	// Check if user can write msg from dest topic
	if isRW, _ := topicDB.GetUserRights(toTopic, &user); !isRW {
		ctx.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("No RW Access to topic %s", toTopic.Topic)})
		return
	}

	// check if message is a reply -> not possible
	if message.InReplyOfIDRoot != "" {
		ctx.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("You can't move a reply message")})
		return
	}

	info := ""
	if messageIn.Action == tat.MessageActionMove {
		err := messageDB.Move(&message, user, fromTopic, *toTopic)
		if err != nil {
			log.Errorf("Error while move a message to topic: %s err: %s", toTopic.Topic, err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error while move a message to topic %s", toTopic.Topic)})
			return
		}
		info = fmt.Sprintf("Message move to %s", toTopic.Topic)
	} else {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action: " + messageIn.Action})
		return
	}
	out := &tat.MessageJSONOut{Info: info, Message: message}
	hook.SendHook(&tat.HookJSON{HookMessage: &tat.HookMessageJSON{MessageJSONOut: out, Action: messageIn.Action}}, *toTopic)
	ctx.JSON(http.StatusCreated, out)
}

func (m *MessagesController) getTopicNameFromAction(username, action string) string {
	return "/Private/" + username + "/" + strings.Title(action) + "s"
}

func (m *MessagesController) inverseIfDMTopic(ctx *gin.Context, topicName string) string {
	if !strings.HasPrefix(topicName, "/Private/") {
		return topicName
	}
	if !strings.HasSuffix(topicName, "/DM/"+getCtxUsername(ctx)) {
		return topicName
	}

	// /Private/usernameFrom/DM/usernameTO
	part := strings.Split(topicName, "/")
	if len(part) != 5 {
		return topicName
	}
	return "/Private/" + getCtxUsername(ctx) + "/DM/" + part[2]
}

func checkDMTopic(ctx *gin.Context, topicName string) (*tat.Topic, string, error) {
	var topic = tat.Topic{}

	if !strings.HasPrefix(topicName, "/Private/"+getCtxUsername(ctx)+"/DM/") {
		log.Debugf("wrong topic name for DM:" + topicName)
		return &topic, "", errors.New("Wrong topic name for DM:" + topicName)
	}

	// /Private/usernameFrom/DM/usernameTO
	part := strings.Split(topicName, "/")
	if len(part) != 5 {
		log.Debugf("wrong topic name for DM")
		return &topic, "", errors.New("Wrong topic name for DM:" + topicName)
	}

	var userFrom = tat.User{}
	found, err := userDB.FindByUsername(&userFrom, getCtxUsername(ctx))
	if !found {
		return &topic, "", errors.New("User unknown")
	} else if err != nil {
		return &topic, "", errors.New("Error while fetching user.")
	}
	var userTo = tat.User{}
	usernameTo := part[4]
	found2, err2 := userDB.FindByUsername(&userTo, usernameTo)
	if !found2 {
		return &topic, "", errors.New("user unknown")
	} else if err2 != nil {
		return &topic, "", errors.New("Error while fetching user.")
	}

	if err3 := checkTopicParentDM(userFrom); err3 != nil {
		return &topic, "", errors.New(err3.Error())
	}

	if err4 := checkTopicParentDM(userTo); err4 != nil {
		return &topic, "", errors.New(err4.Error())
	}

	topic, err5 := insertTopicDM(userFrom, userTo)
	if err5 != nil {
		return &topic, "", errors.New(err5.Error())
	}

	if _, err6 := insertTopicDM(userTo, userFrom); err6 != nil {
		return &topic, "", errors.New(err6.Error())
	}

	topicCriteria := topicName + "," + "/Private/" + usernameTo + "/DM/" + userFrom.Username
	return &topic, topicCriteria, nil
}

func insertTopicDM(userFrom, userTo tat.User) (tat.Topic, error) {
	var topic = tat.Topic{}
	topicName := "/Private/" + userFrom.Username + "/DM/" + userTo.Username
	topic.Topic = topicName
	topic.Description = userTo.Fullname
	if err := topicDB.Insert(&topic, &userFrom); err != nil {
		log.Errorf("Error while InsertTopic %s", err)
		return topic, err
	}
	return topic, nil
}

func checkTopicParentDM(user tat.User) error {
	topicName := "/Private/" + user.Username + "/DM"
	topicParent, err := topicDB.FindByTopic(topicName, false, false, false, nil)
	if err != nil {
		topicParent.Topic = topicName
		topicParent.Description = "DM Topics"
		if err := topicDB.Insert(topicParent, &user); err != nil {
			log.Errorf("Error while InsertTopic Parent %s", err)
			return err
		}
	}
	return nil
}

// DeleteBulkCascade deletes messages and its replies, with criterias
func (m *MessagesController) DeleteBulkCascade(ctx *gin.Context) {
	m.messagesDeleteBulk(ctx, true, false)
}

// DeleteBulkCascadeForce deletes message and replies, event if a msg is in a
// tasks topic of one user, messages selected with criterias
func (m *MessagesController) DeleteBulkCascadeForce(ctx *gin.Context) {
	m.messagesDeleteBulk(ctx, true, true)
}

// DeleteBulk deletes messages matching criterias
func (m *MessagesController) DeleteBulk(ctx *gin.Context) {
	m.messagesDeleteBulk(ctx, false, false)
}

func (m *MessagesController) messagesDeleteBulk(ctx *gin.Context, cascade, force bool) {
	out, user, topic, criteria, httpCode, err := m.innerList(ctx)
	if criteria.TreeView == "" {
		criteria.TreeView = tat.TreeViewOneTree
	}

	if err != nil {
		ctx.JSON(httpCode, gin.H{"error": err.Error()})
		return
	}

	messages, err := messageDB.ListMessages(criteria, user.Username, topic)
	if err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	out.Messages = messages

	// check all before
	for _, msg := range out.Messages {
		errCheck := m.checkBeforeDelete(ctx, msg, user, force, topic)
		if errCheck != nil {
			// ctx writes in checkBeforeDelete
			return
		}
		if cascade {
			for _, r := range msg.Replies {
				errCheck := m.checkBeforeDelete(ctx, r, user, force, topic)
				if errCheck != nil {
					// ctx writes in checkBeforeDelete
					return
				}
			}
		} else if len(msg.Replies) > 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Could not delete a message, message have replies"})
			return
		}
	}

	nbDelete := 0
	for _, msg := range out.Messages {
		if err = messageDB.Delete(&msg, cascade, topic); err != nil {
			log.Errorf("Error while delete a message %s", err)
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		nbDelete++
	}

	ctx.JSON(http.StatusOK, gin.H{"info": fmt.Sprintf("%d messages (cascade:%t) deleted from %s, limit criteria to %d messages root",
		nbDelete, cascade, topic.Topic, criteria.Limit)})
}
