package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/ovh/tat"
	presenceDB "github.com/ovh/tat/api/presence"
	topicDB "github.com/ovh/tat/api/topic"
	userDB "github.com/ovh/tat/api/user"
)

// PresencesController contains all methods about presences manipulation
type PresencesController struct{}

func (*PresencesController) buildCriteria(ctx *gin.Context) *tat.PresenceCriteria {
	c := tat.PresenceCriteria{}
	skip, e := strconv.Atoi(ctx.DefaultQuery("skip", "0"))
	if e != nil {
		skip = 0
	}
	c.Skip = skip
	limit, e2 := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	if e2 != nil {
		limit = 10
	}
	c.Limit = limit
	c.IDPresence = ctx.Query("idPresence")
	c.Status = ctx.Query("status")
	c.Username = ctx.Query("username")
	c.DateMinPresence = ctx.Query("dateMinPresence")
	c.DateMaxPresence = ctx.Query("dateMaxPresence")
	if c.SortBy == "" {
		c.SortBy = "-datePresence"
	}
	return &c
}

// List list presences with given criteria
func (m *PresencesController) List(ctx *gin.Context) {
	criteria := m.buildCriteria(ctx)
	topicIn, found := ctx.Params.Get("topic")
	if found {
		criteria.Topic = topicIn
	}
	m.listWithCriteria(ctx, criteria)
}

func (m *PresencesController) listWithCriteria(ctx *gin.Context, criteria *tat.PresenceCriteria) {
	user, e := m.preCheckUser(ctx)
	if e != nil {
		return
	}

	if criteria.Topic != "" {
		_, err := topicDB.FindByTopic(criteria.Topic, true, false, false, user)
		if err != nil {
			ctx.AbortWithError(http.StatusBadRequest, errors.New("topic "+criteria.Topic+" does not exist or you have no Read Access on it"))
			return
		}

		// add / if search on topic
		// as topic is in path, it can't start with a /
		if criteria.Topic != "" && string(criteria.Topic[0]) != "/" {
			criteria.Topic = "/" + criteria.Topic
		}

		topicDM := "/Private/" + getCtxUsername(ctx) + "/DM/"
		if strings.HasPrefix(criteria.Topic, topicDM) {
			part := strings.Split(criteria.Topic, "/")
			if len(part) != 5 {
				log.Errorf("wrong topic name for DM")
				ctx.AbortWithError(http.StatusInternalServerError, errors.New("Wrong topic name for DM:"+criteria.Topic))
				return
			}
			topicInverse := "/Private/" + part[4] + "/DM/" + getCtxUsername(ctx)
			criteria.Topic = criteria.Topic + "," + topicInverse
		}
	}

	count, presences, err := presenceDB.ListPresences(criteria)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	out := &tat.PresencesJSON{
		Count:     count,
		Presences: presences,
	}
	ctx.JSON(http.StatusOK, out)
}

func (m *PresencesController) preCheckTopic(ctx *gin.Context) (tat.PresenceJSON, tat.Topic, *tat.User, error) {
	var presenceIn tat.PresenceJSON
	ctx.Bind(&presenceIn)

	topicIn, err := GetParam(ctx, "topic")
	if err != nil {
		return presenceIn, tat.Topic{}, nil, err
	}
	presenceIn.Topic = topicIn

	user, e := m.preCheckUser(ctx)
	if e != nil {
		return presenceIn, tat.Topic{}, nil, err
	}

	topic, err := topicDB.FindByTopic(presenceIn.Topic, true, false, false, user)
	if err != nil {
		e := errors.New("Topic " + presenceIn.Topic + " does not exist")
		ctx.AbortWithError(http.StatusInternalServerError, e)
		return presenceIn, tat.Topic{}, nil, e
	}
	return presenceIn, *topic, user, nil
}

func (*PresencesController) preCheckUser(ctx *gin.Context) (*tat.User, error) {
	var user = &tat.User{}
	found, err := userDB.FindByUsername(user, getCtxUsername(ctx))
	var e error
	if !found {
		e = errors.New("User unknown")
	} else if err != nil {
		e = errors.New("Error while fetching user")
	}
	if e != nil {
		ctx.AbortWithError(http.StatusInternalServerError, e)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})
		return nil, e
	}
	return user, nil
}

func (m *PresencesController) create(ctx *gin.Context) {
	presenceIn, topic, user, e := m.preCheckTopic(ctx)
	if e != nil {
		return
	}

	var presence = tat.Presence{}
	if err := presenceDB.Upsert(&presence, *user, topic, presenceIn.Status); err != nil {
		log.Errorf("Error while InsertPresence %s", err)
		ctx.AbortWithError(http.StatusInternalServerError, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}

// CreateAndGet creates a presence and get presences on current topic
func (m *PresencesController) CreateAndGet(ctx *gin.Context) {
	m.create(ctx)
	if ctx.IsAborted() {
		return
	}

	fiften := strconv.FormatInt(time.Now().Unix()-15, 10)

	topicIn, _ := GetParam(ctx, "topic") // no error possible here
	criteria := &tat.PresenceCriteria{
		Skip:            0,
		Limit:           1000,
		Topic:           topicIn,
		DateMinPresence: fiften,
	}

	m.listWithCriteria(ctx, criteria)
}

// Delete deletes all presences of one user, on one topic
func (m *PresencesController) Delete(ctx *gin.Context) {
	presenceIn, topic, userAction, e := m.preCheckTopic(ctx)
	if e != nil {
		return
	}

	userToDelete := &tat.User{}
	if userAction.IsAdmin {
		found, err := userDB.FindByUsername(userToDelete, presenceIn.Username)
		if !found {
			e := errors.New("User unknown while fetching user " + presenceIn.Username + " for delete presence")
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})
			return
		} else if err != nil {
			e := errors.New("Error while fetching user " + presenceIn.Username + " for delete presence")
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})
			return
		}
	}

	if err := presenceDB.Delete(*userToDelete, topic); err != nil {
		log.Errorf("Error while DeletePresence %s", err)
		ctx.AbortWithError(http.StatusInternalServerError, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, nil)
}

// CheckAllPresences checks presences, delete double
func (m *PresencesController) CheckAllPresences(ctx *gin.Context) {
	// admin check in route
	statsPresences, err := presenceDB.CheckAllPresences()
	if err != nil {
		log.Errorf("Error while get models.CheckAllPresences %s", err)
	}

	now := time.Now()
	ctx.JSON(http.StatusOK, gin.H{
		"date":           now.Unix(),
		"dateHuman":      now,
		"statsPresences": statsPresences,
	})
}
