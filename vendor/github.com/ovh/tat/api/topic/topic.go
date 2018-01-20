package topic

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/ovh/tat"
	"github.com/ovh/tat/api/cache"
	"github.com/ovh/tat/api/group"
	"github.com/ovh/tat/api/store"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// InitDB creates /Private topic if necessary
func InitDB() {
	nbTopics, err := CountTopics()
	if err != nil {
		log.Fatalf("Error with getting Mongodb.Instance %s", err)
		return
	}

	if nbTopics == 0 {
		// Create /Private topic
		InitPrivateTopic()
	}
}

func buildTopicCriteria(criteria *tat.TopicCriteria, user *tat.User) (bson.M, error) {
	var query = []bson.M{}

	if criteria.IDTopic != "" {
		queryIDTopics := bson.M{}
		queryIDTopics["$or"] = []bson.M{}
		for _, val := range strings.Split(criteria.IDTopic, ",") {
			queryIDTopics["$or"] = append(queryIDTopics["$or"].([]bson.M), bson.M{"_id": val})
		}
		query = append(query, queryIDTopics)
	}
	if criteria.Topic != "" || criteria.OnlyFavorites == tat.True {
		queryTopics := bson.M{}
		queryTopics["$or"] = []bson.M{}
		for _, val := range strings.Split(criteria.Topic, ",") {
			queryTopics["$or"] = append(queryTopics["$or"].([]bson.M), bson.M{"topic": val})
		}
		query = append(query, queryTopics)
	}
	if criteria.TopicPath != "" {
		query = append(query, bson.M{"topic": bson.RegEx{Pattern: "^" + regexp.QuoteMeta(criteria.TopicPath) + ".*$", Options: "im"}})
	}
	if criteria.Description != "" {
		queryDescriptions := bson.M{}
		queryDescriptions["$or"] = []bson.M{}
		for _, val := range strings.Split(criteria.Description, ",") {
			queryDescriptions["$or"] = append(queryDescriptions["$or"].([]bson.M), bson.M{"description": val})
		}
		query = append(query, queryDescriptions)
	}
	if criteria.Group != "" {
		queryGroups := bson.M{}
		queryGroups["$or"] = []bson.M{}
		queryGroups["$or"] = append(queryGroups["$or"].([]bson.M), bson.M{"adminGroups": bson.M{"$in": strings.Split(criteria.Group, ",")}})
		queryGroups["$or"] = append(queryGroups["$or"].([]bson.M), bson.M{"roGroups": bson.M{"$in": strings.Split(criteria.Group, ",")}})
		queryGroups["$or"] = append(queryGroups["$or"].([]bson.M), bson.M{"rwGroups": bson.M{"$in": strings.Split(criteria.Group, ",")}})
		query = append(query, queryGroups)
	}

	var bsonDate = bson.M{}

	if criteria.DateMinCreation != "" {
		i, err := strconv.ParseInt(criteria.DateMinCreation, 10, 64)
		if err != nil {
			return bson.M{}, fmt.Errorf("Error while parsing dateMinCreation %s", err)
		}
		tm := time.Unix(i, 0)
		bsonDate["$gte"] = tm.Unix()
	}
	if criteria.DateMaxCreation != "" {
		i, err := strconv.ParseInt(criteria.DateMaxCreation, 10, 64)
		if err != nil {
			return bson.M{}, fmt.Errorf("Error while parsing dateMaxCreation %s", err)
		}
		tm := time.Unix(i, 0)
		bsonDate["$lte"] = tm.Unix()
	}
	if len(bsonDate) > 0 {
		query = append(query, bson.M{"dateCreation": bsonDate})
	}

	if user != nil {
		if criteria.GetForAllTasksTopics {
			query = append(query, bson.M{
				"topic": bson.RegEx{Pattern: "^\\/Private\\/.*/Tasks", Options: "i"},
			})
		} else if criteria.GetForTatAdmin == tat.True && user.IsAdmin {
			// requester is tat Admin and wants all topics, except /Private/* topics
			query = append(query, bson.M{
				"topic": bson.M{"$not": bson.RegEx{Pattern: "^\\/Private\\/.*", Options: "i"}},
			})
		} else if criteria.GetForTatAdmin == tat.True && !user.IsAdmin {
			log.Warnf("User %s (not a TatAdmin) try to list all topics as an admin", user.Username)
		} else {
			bsonUser := []bson.M{}
			bsonUser = append(bsonUser, bson.M{"roUsers": bson.M{"$in": [1]string{user.Username}}})
			bsonUser = append(bsonUser, bson.M{"rwUsers": bson.M{"$in": [1]string{user.Username}}})
			bsonUser = append(bsonUser, bson.M{"adminUsers": bson.M{"$in": [1]string{user.Username}}})
			userGroups, err := group.GetUserGroupsOnlyName(user.Username)
			if err != nil {
				log.Errorf("Error with getting groups for user %s", err)
			} else {
				bsonUser = append(bsonUser, bson.M{"roGroups": bson.M{"$in": userGroups}})
				bsonUser = append(bsonUser, bson.M{"rwGroups": bson.M{"$in": userGroups}})
				bsonUser = append(bsonUser, bson.M{"adminGroups": bson.M{"$in": userGroups}})
			}
			query = append(query, bson.M{"$or": bsonUser})
		}
	}

	if len(query) > 0 {
		return bson.M{"$and": query}, nil
	} else if len(query) == 1 {
		return query[0], nil
	}
	return bson.M{}, nil
}

// GetTopicSelectedFields return allowed selected field on mongo
func GetTopicSelectedFields(isAdmin, withTags, withLabels, oneTopic bool) bson.M {
	var b bson.M

	if isAdmin {
		b = bson.M{
			"_id":                  1,
			"collection":           1,
			"topic":                1,
			"description":          1,
			"roGroups":             1,
			"rwGroups":             1,
			"roUsers":              1,
			"rwUsers":              1,
			"adminUsers":           1,
			"adminGroups":          1,
			"maxlength":            1,
			"maxreplies":           1,
			"canForceDate":         1,
			"canUpdateMsg":         1,
			"canDeleteMsg":         1,
			"canUpdateAllMsg":      1,
			"canDeleteAllMsg":      1,
			"adminCanUpdateAllMsg": 1,
			"adminCanDeleteAllMsg": 1,
			"isAutoComputeTags":    1,
			"isAutoComputeLabels":  1,
			"dateModificationn":    1,
			"dateCreation":         1,
			"dateLastMessage":      1,
			"parameters":           1,
		}
		if oneTopic {
			b["history"] = 1
		}
	} else {
		b = bson.M{
			"collection":           1,
			"topic":                1,
			"description":          1,
			"roGroups":             1,
			"rwGroups":             1,
			"roUsers":              1,
			"rwUsers":              1,
			"adminUsers":           1,
			"adminGroups":          1,
			"canForceDate":         1,
			"canUpdateMsg":         1,
			"canDeleteMsg":         1,
			"canUpdateAllMsg":      1,
			"canDeleteAllMsg":      1,
			"adminCanUpdateAllMsg": 1,
			"adminCanDeleteAllMsg": 1,
			"isAutoComputeTags":    1,
			"isAutoComputeLabels":  1,
			"maxlength":            1,
			"maxreplies":           1,
			"dateLastMessage":      1,
			"parameters":           1,
		}
	}
	if oneTopic {
		b["filters"] = 1
	}
	if withTags {
		b["tags"] = 1
	}
	if withLabels {
		b["labels"] = 1
	}
	return b
}

// CountTopics returns the total number of topics in db
func CountTopics() (int, error) {
	return store.Tat().CTopics.Count()
}

// FindAllTopicsWithCollections returns the total number of topics in db
func FindAllTopicsWithCollections() ([]tat.Topic, error) {
	var topics []tat.Topic
	err := store.Tat().CTopics.Find(bson.M{"collection": bson.M{"$exists": true, "$ne": ""}}).
		Select(bson.M{"_id": 1, "collection": 1, "topic": 1}).
		All(&topics)
	return topics, err
}

// ListTopics returns list of topics, matching criterias
// /!\ user arg could be nil
func ListTopics(criteria *tat.TopicCriteria, u *tat.User, isAdmin, withTags, withLabels bool) (int, []tat.Topic, error) {
	var topics []tat.Topic

	username := "internal"
	if u != nil {
		username = u.Username
	}
	k := cache.CriteriaKey(criteria, "tat", "users", username, "topics", "list_topics", "isAdmin", strconv.FormatBool(isAdmin), "withTags", strconv.FormatBool(withTags), "withLabels", strconv.FormatBool(withLabels))
	kcount := cache.CriteriaKey(criteria, "tat", "users", username, "topics", "count_topics")

	bytes, _ := cache.Client().Get(k).Bytes()
	if len(bytes) > 0 {
		json.Unmarshal(bytes, &topics)
	}

	ccount, _ := cache.Client().Get(kcount).Int64()
	if len(topics) > 0 && ccount > 0 {
		log.Debugf("ListTopics: topics (%s) loaded from cache", k)
		return int(ccount), topics, nil
	}

	cursor, errl := listTopicsCursor(criteria, u)
	if errl != nil {
		return -1, nil, errl
	}
	count, errc := cursor.Count()
	if errc != nil {
		return -1, nil, fmt.Errorf("Error while count Topics %s", errc)
	}
	oneTopic := false
	if criteria.Topic != "" {
		oneTopic = true
	}

	sortBy := criteria.SortBy
	if sortBy == "" {
		sortBy = "topic"
	}
	err := cursor.Select(GetTopicSelectedFields(isAdmin, withTags, withLabels, oneTopic)).
		Sort(sortBy).
		Skip(criteria.Skip).
		Limit(criteria.Limit).
		All(&topics)

	if err != nil {
		log.Errorf("Error while Find Topics %s", err)
		return -1, nil, err
	}

	cache.Client().Set(kcount, count, time.Hour)
	bytes, _ = json.Marshal(topics)
	if len(bytes) > 0 {
		log.Debugf("ListTopics: Put %s in cache", k)
		cache.Client().Set(k, string(bytes), time.Hour)
	}
	ku := cache.Key("tat", "users", username, "topics")
	cache.Client().SAdd(ku, k, kcount)
	cache.Client().SAdd(cache.Key(cache.TatTopicsKeys()...), ku, k, kcount)
	return count, topics, err
}

func listTopicsCursor(criteria *tat.TopicCriteria, user *tat.User) (*mgo.Query, error) {
	c, err := buildTopicCriteria(criteria, user)
	if err != nil {
		return nil, err
	}
	return store.Tat().CTopics.Find(c), nil
}

// InitPrivateTopic insert topic "/Private"
func InitPrivateTopic() {
	topic := &tat.Topic{
		ID:                   bson.NewObjectId().Hex(),
		Topic:                "/Private",
		Description:          "Private Topics",
		DateCreation:         time.Now().Unix(),
		MaxLength:            tat.DefaultMessageMaxSize,
		MaxReplies:           tat.DefaultMessageMaxReplies,
		CanForceDate:         false,
		CanUpdateMsg:         false,
		CanDeleteMsg:         false,
		CanUpdateAllMsg:      false,
		CanDeleteAllMsg:      false,
		AdminCanUpdateAllMsg: false,
		AdminCanDeleteAllMsg: false,
		IsAutoComputeTags:    true,
		IsAutoComputeLabels:  true,
	}
	log.Infof("Initialize /Private Topic")
	if err := store.Tat().CTopics.Insert(topic); err != nil {
		log.Fatalf("Error while initialize /Private Topic %s", err)
	}
}

// Insert creates a new topic. User is read write on topic
func Insert(topic *tat.Topic, u *tat.User) error {
	if err := CheckAndFixName(topic); err != nil {
		return err
	}

	isParentRootTopic, parentTopic, err := getParentTopic(topic)
	if !isParentRootTopic {
		if err != nil {
			return tat.NewError(http.StatusNotFound, "Parent Topic not found %s", topic.Topic)
		}
		// If user create a Topic in /Private/username, no check or RW to create
		if !strings.HasPrefix(topic.Topic, "/Private/"+u.Username) {
			// check if user can create topic in /topic
			hasRW := IsUserAdmin(parentTopic, u)
			if !hasRW {
				return tat.NewError(http.StatusUnauthorized, "No RW access to parent topic %s", parentTopic.Topic)
			}
		}
	} else if !u.IsAdmin { // no parent topic, check admin
		return tat.NewError(http.StatusUnauthorized, "No write access to create parent topic %s", topic.Topic)
	}
	if _, err = FindByTopic(topic.Topic, true, false, false, nil); err == nil {
		return tat.NewError(http.StatusConflict, "Topic Already Exists : %s", topic.Topic)
	}

	topic.ID = bson.NewObjectId().Hex()
	topic.DateCreation = time.Now().Unix()
	topic.MaxLength = tat.DefaultMessageMaxSize     // topic MaxLenth messages
	topic.MaxReplies = tat.DefaultMessageMaxReplies // topic max replies on a message
	topic.CanForceDate = false
	topic.IsAutoComputeLabels = true
	topic.IsAutoComputeTags = true
	topic.Collection = "messages" + topic.ID

	if !isParentRootTopic {
		topic.ROGroups = parentTopic.ROGroups
		topic.RWGroups = parentTopic.RWGroups
		topic.ROUsers = parentTopic.ROUsers
		topic.RWUsers = parentTopic.RWUsers
		topic.AdminUsers = parentTopic.AdminUsers
		topic.AdminGroups = parentTopic.AdminGroups
		topic.MaxLength = parentTopic.MaxLength
		topic.CanForceDate = parentTopic.CanForceDate
		// topic.CanUpdateMsg can be set by user.createTopics for new users
		// with CanUpdateMsg=true
		if !topic.CanUpdateMsg {
			topic.CanUpdateMsg = parentTopic.CanUpdateMsg
		}
		// topic.CanDeleteMsg can be set by user.createTopics for new users
		// with CanDeleteMsg=true
		if !topic.CanDeleteMsg {
			topic.CanDeleteMsg = parentTopic.CanDeleteMsg
		}
		topic.CanUpdateAllMsg = parentTopic.CanUpdateAllMsg
		topic.CanDeleteAllMsg = parentTopic.CanDeleteAllMsg
		topic.AdminCanUpdateAllMsg = parentTopic.AdminCanUpdateAllMsg
		topic.AdminCanDeleteAllMsg = parentTopic.AdminCanDeleteAllMsg
		topic.IsAutoComputeTags = parentTopic.IsAutoComputeTags
		topic.IsAutoComputeLabels = parentTopic.IsAutoComputeLabels
		topic.Parameters = parentTopic.Parameters
	}

	if err = store.Tat().CTopics.Insert(topic); err != nil {
		log.Errorf("Error while inserting new topic %s", err)
	}

	if errC := store.Tat().Session.DB(store.DatabaseName).C(topic.Collection).Create(&mgo.CollectionInfo{ForceIdIndex: true}); errC != nil {
		log.Errorf("Error while create new collection %s", topic.Collection)
	}

	store.EnsureIndexesMessages(topic.Collection)

	h := fmt.Sprintf("create a new topic :%s", topic.Topic)
	err = addToHistory(topic, bson.M{"_id": topic.ID}, u.Username, h)
	if err != nil {
		log.Errorf("Error while inserting history for new topic %s", err)
	}

	log.Debugf("Insert: Clean topics cache for user %s", u.Username)
	cache.CleanAllTopicsLists()

	return AddRwUser(topic, u.Username, u.Username, false)
}

// Delete deletes a topic from database
func Delete(topic *tat.Topic, u *tat.User) error {
	log.Debugf("Delete: Clean topics cache for user %s", u.Username)
	if topic.Collection != "" {
		err := store.Tat().CTopics.Update(
			bson.M{"_id": topic.ID},
			bson.M{"$set": bson.M{"description": topic.Description + " TODEL"}})
		if err != nil {
			log.Errorf("Error while update description before delete topic %s err:%s", topic.Topic, err)
		}
		store.Tat().Session.SetMode(mgo.Strong, true)
		defer store.Tat().Session.SetMode(mgo.SecondaryPreferred, true)
		if err := store.Tat().Session.DB(store.DatabaseName).C(topic.Collection).DropCollection(); err != nil {
			return fmt.Errorf("Error while drop collection for topic %s err: %s", topic.Topic, err)
		}
	}

	if err := store.Tat().CTopics.Remove(bson.M{"_id": topic.ID}); err != nil {
		return fmt.Errorf("Error while remove topic from topics collection: %s", err)
	}
	cache.CleanAllTopicsLists()
	return nil
}

// Truncate removes all messages in a topic
func Truncate(topic *tat.Topic) (int, error) {
	var changeInfo *mgo.ChangeInfo
	var err error
	if topic.Collection != "" && topic.Collection != store.CollectionDefaultMessages {
		changeInfo, err = store.GetCMessages(topic.Collection).RemoveAll(bson.M{})
	} else {
		changeInfo, err = store.GetCMessages(topic.Collection).RemoveAll(bson.M{"topic": topic.Topic})
		// TODO remove this after remove defaultMessagesCollection
		changeInfoOld, errOld := store.GetCMessages(topic.Collection).RemoveAll(bson.M{"topics": bson.M{"$in": [1]string{topic.Topic}}})
		if errOld != nil {
			log.Warnf("Error while removing message with topics attribute: %s", errOld)
		} else {
			log.Infof("Remove %d message with old way, select on topics attribute", changeInfoOld.Removed)
		}
	}

	if err != nil {
		return 0, err
	}
	cache.CleanMessagesLists(topic.Topic)
	return changeInfo.Removed, err
}

// TruncateTags clears "cached" tags in topic
func TruncateTags(topic *tat.Topic) error {
	err := store.Tat().CTopics.Update(
		bson.M{"_id": topic.ID},
		bson.M{"$unset": bson.M{"tags": ""}})

	cache.CleanTopicByName(topic.Topic)
	return err
}

// TruncateLabels clears "cached" labels on a topic
func TruncateLabels(topic *tat.Topic) error {
	err := store.Tat().CTopics.Update(
		bson.M{"_id": topic.ID},
		bson.M{"$unset": bson.M{"labels": ""}})

	cache.CleanTopicByName(topic.Topic)
	return err
}

var topicsLastMsgUpdate map[string]int64
var syncLastMsgUpdate sync.Mutex

func init() {
	topicsLastMsgUpdate = make(map[string]int64)
	go updateLastMessageTopics()
}

func updateLastMessageTopics() {
	for {
		syncLastMsgUpdate.Lock()
		if len(topicsLastMsgUpdate) > 0 {
			workOnTopicsLastMsgUpdate()
		}
		syncLastMsgUpdate.Unlock()
		time.Sleep(10 * time.Second)
	}
}

func workOnTopicsLastMsgUpdate() {
	for topic, dateUpdate := range topicsLastMsgUpdate {
		err := store.Tat().CTopics.Update(
			bson.M{"topic": topic},
			bson.M{"$set": bson.M{"dateLastMessage": dateUpdate}})
		if err != nil {
			log.Errorf("Error while update last date message on topic %s, err:%s", topic, err)
		}
	}
	topicsLastMsgUpdate = make(map[string]int64)
}

// UpdateTopicLastMessage updates tags on topic
func UpdateTopicLastMessage(topic *tat.Topic, dateUpdateLastMsg time.Time) {
	syncLastMsgUpdate.Lock()
	topicsLastMsgUpdate[topic.Topic] = dateUpdateLastMsg.Unix()
	syncLastMsgUpdate.Unlock()
}

// UpdateTopicTags updates tags on topic
func UpdateTopicTags(topic *tat.Topic, tags []string) {
	if !topic.IsAutoComputeTags || len(tags) == 0 {
		return
	}

	update := false
	newTags := topic.Tags
	for _, tag := range tags {
		if !tat.ArrayContains(topic.Tags, tag) {
			update = true
			newTags = append(newTags, tag)
		}
	}

	if update {
		err := store.Tat().CTopics.Update(
			bson.M{"_id": topic.ID},
			bson.M{"$set": bson.M{"tags": newTags}})

		if err != nil {
			log.Errorf("UpdateTopicTags> Error while updating tags on topic")
		} else {
			log.Debugf("UpdateTopicTags> Topic %s ", topic.Topic)
		}
		cache.CleanTopicByName(topic.Topic)
	}
}

// UpdateTopicLabels updates labels on topic
func UpdateTopicLabels(topic *tat.Topic, labels []tat.Label) {
	if !topic.IsAutoComputeLabels || len(labels) == 0 {
		return
	}

	update := false
	newLabels := topic.Labels
	for _, label := range labels {
		find := false
		for _, tlabel := range topic.Labels {
			if label.Text == tlabel.Text {
				find = true
				continue
			}
		}
		if !find {
			newLabels = append(newLabels, label)
			update = true
		}
	}

	if update {
		err := store.Tat().CTopics.Update(
			bson.M{"_id": topic.ID},
			bson.M{"$set": bson.M{"labels": newLabels}})

		if err != nil {
			log.Errorf("UpdateTopicLabels> Error while updating labels on topic")
		} else {
			log.Debugf("UpdateTopicLabels> Topic %s ", topic.Topic)
		}
		cache.CleanTopicByName(topic.Topic)
	}
}

// ListTags returns all tags on one topic
func ListTags(topic tat.Topic) ([]string, error) {
	var tags []string
	err := store.GetCMessages(topic.Collection).
		Find(bson.M{"topic": topic.Topic}).
		Distinct("tags", &tags)
	if err != nil {
		log.Errorf("Error while getting tags on topic %s, err:%s", topic.Topic, err.Error())
	}
	return tags, err
}

// ComputeTags computes "cached" tags in topic
// initialize tags, one entry per tag (unique)
func ComputeTags(topic *tat.Topic) (int, error) {
	tags, err := ListTags(*topic)
	if err != nil {
		return 0, err
	}

	err = store.Tat().CTopics.Update(
		bson.M{"_id": topic.ID},
		bson.M{"$set": bson.M{"tags": tags}})

	cache.CleanTopicByName(topic.Topic)
	return len(tags), err
}

// ListLabels returns all labels on one topic
func ListLabels(topic tat.Topic) ([]tat.Label, error) {
	var labels []tat.Label
	err := store.GetCMessages(topic.Collection).
		Find(bson.M{"topic": topic.Topic}).
		Distinct("labels", &labels)
	if err != nil {
		log.Errorf("Error while getting labels on topic %s, err:%s", topic.Topic, err.Error())
	}
	return labels, err
}

// ComputeLabels computes "cached" labels on a topic
// initialize labels, one entry per label (unicity with text & color)
func ComputeLabels(topic *tat.Topic) (int, error) {
	labels, err := ListLabels(*topic)
	if err != nil {
		return 0, err
	}

	err = store.Tat().CTopics.Update(
		bson.M{"_id": topic.ID},
		bson.M{"$set": bson.M{"labels": labels}})

	cache.CleanTopicByName(topic.Topic)
	return len(labels), err
}

// AllTopicsComputeLabels computes Labels on all topics
func AllTopicsComputeLabels() (string, error) {
	var topics []tat.Topic
	err := store.Tat().CTopics.Find(bson.M{}).
		Select(GetTopicSelectedFields(true, false, false, false)).
		All(&topics)

	if err != nil {
		log.Errorf("Error while getting all topics for compute labels")
		return "", err
	}

	errTxt := ""
	infoTxt := ""
	for _, topic := range topics {
		if topic.IsAutoComputeLabels {
			n, err := ComputeLabels(&topic)
			if err != nil {
				log.Errorf("Error while compute labels on topic %s: %s", topic.Topic, err.Error())
				errTxt += fmt.Sprintf(" Error compute labels on topic %s", topic.Topic)
			} else {
				infoTxt += fmt.Sprintf(" %d labels computed on topic %s", n, topic.Topic)
				log.Infof(infoTxt)
			}
		}
	}

	if errTxt != "" {
		return infoTxt, fmt.Errorf(errTxt)
	}
	return infoTxt, nil
}

// AllTopicsComputeTags computes Tags on all topics
func AllTopicsComputeTags() (string, error) {
	var topics []tat.Topic
	err := store.Tat().CTopics.Find(bson.M{}).
		Select(GetTopicSelectedFields(true, false, false, false)).
		All(&topics)

	if err != nil {
		log.Errorf("Error while getting all topics for compute tags")
		return "", err
	}

	errTxt := ""
	infoTxt := ""
	for _, topic := range topics {
		if topic.IsAutoComputeTags {
			n, err := ComputeTags(&topic)
			if err != nil {
				log.Errorf("Error while compute tags on topic %s: %s", topic.Topic, err.Error())
				errTxt += fmt.Sprintf(" Error compute tags on topic %s", topic.Topic)
			} else {
				infoTxt += fmt.Sprintf(" %d tags computed on topic %s", n, topic.Topic)
				log.Infof(infoTxt)
			}
		}
	}

	if errTxt != "" {
		return infoTxt, fmt.Errorf(errTxt)
	}
	return infoTxt, nil
}

// AllTopicsSetParam computes Tags on all topics
func AllTopicsSetParam(key, value string) (string, error) {
	var topics []tat.Topic
	err := store.Tat().CTopics.Find(bson.M{}).
		Select(GetTopicSelectedFields(true, false, false, false)).
		All(&topics)

	if err != nil {
		log.Errorf("Error while getting all topics for set a param")
		return "", err
	}

	errTxt := ""
	nOk := 1
	for _, topic := range topics {
		if err := setAParam(&topic, key, value); err != nil {
			log.Errorf("Error while set param %s on topic %s: %s", key, topic.Topic, err.Error())
			errTxt += fmt.Sprintf(" Error set param %s on topic %s", key, topic.Topic)
		} else {
			log.Infof(" %s param setted on topic %s", key, topic.Topic)
			nOk++
		}
	}

	if errTxt != "" {
		return "", fmt.Errorf(errTxt)
	}

	return fmt.Sprintf("Param setted on %d topics", nOk), nil
}

// setAParam sets a param on one topic. Limited only of some attributes
func setAParam(topic *tat.Topic, key, value string) error {
	if key == "isAutoComputeTags" || key == "isAutoComputeLabels" {
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("Error while set param %s with value %s", key, value)
		}
		return setParamInDB(topic, key, v)
	} else if key == "maxreplies" {
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("Error while set param %s with value %s", key, value)
		}
		return setParamInDB(topic, key, v)
	}
	return fmt.Errorf("set param %s is an invalid action", key)
}

func setParamInDB(topic *tat.Topic, key string, value interface{}) error {
	if key != "maxreplies" && key != "isAutoComputeTags" && key != "isAutoComputeLabels" {
		return fmt.Errorf("set param %s is an invalid action", key)
	}

	err := store.Tat().CTopics.Update(
		bson.M{"_id": topic.ID},
		bson.M{"$set": bson.M{key: value}},
	)
	if err != nil {
		log.Errorf("Error while update topic %s, param %s with new value %s", topic.Topic, key, value)
	}
	cache.CleanTopicByName(topic.Topic)
	return nil
}

// Get parent topic
// If it is a "root topic", like /myTopic, return true, nil, nil
func getParentTopic(topic *tat.Topic) (bool, *tat.Topic, error) {
	index := strings.LastIndex(topic.Topic, "/")
	if index == 0 {
		return true, nil, nil
	}
	var nameParent = topic.Topic[0:index]
	parentTopic, err := FindByTopic(nameParent, true, false, false, nil)
	if err != nil {
		log.Errorf("Error while fetching parent topic %s", err)
	}
	return false, parentTopic, err
}

// FindByTopic returns topic by topicName.
func FindByTopic(topicIn string, isAdmin, withTags, withLabels bool, user *tat.User) (*tat.Topic, error) {
	topic := &tat.Topic{
		Topic: topicIn,
	}
	if err := CheckAndFixName(topic); err != nil {
		return nil, err
	}
	criteria := &tat.TopicCriteria{
		Topic: topic.Topic,
	}
	nb, topics, err := ListTopics(criteria, user, isAdmin, withTags, withLabels)
	if err != nil {
		return nil, err
	}
	if nb != 1 && len(topics) != 1 {
		return nil, fmt.Errorf("Invalid Request. Get many topics instead one")
	}
	if topics[0].MaxReplies == 0 {
		topics[0].MaxReplies = tat.DefaultMessageMaxReplies
	}
	return &topics[0], nil
}

// IsTopicExists return true if topic exists, false otherwise
func IsTopicExists(topicName string) bool {
	_, err := FindByTopic(topicName, false, false, false, nil)
	return err == nil // no error, return true
}

// SetParam update param maxLength, maxReplies, canForceDate, canUpdateMsg, canDeleteMsg,
// canUpdateAllMsg, canDeleteAllMsg, adminCanUpdateAllMsg, adminCanDeleteAllMsg, parameters on topic
func SetParam(topic *tat.Topic, username string, recursive bool, maxLength, maxReplies int,
	canForceDate, canUpdateMsg, canDeleteMsg, canUpdateAllMsg, canDeleteAllMsg, adminCanUpdateAllMsg, adminCanDeleteAllMsg,
	isAutoComputeTags, isAutoComputeLabels bool, parameters []tat.TopicParameter) error {

	var selector bson.M

	if recursive {
		selector = bson.M{"topic": bson.RegEx{Pattern: "^" + topic.Topic + ".*$"}}
	} else {
		selector = bson.M{"_id": topic.ID}
	}

	if maxLength <= 0 {
		maxLength = tat.DefaultMessageMaxSize
	}

	update := bson.M{
		"maxlength":            maxLength,
		"maxreplies":           maxReplies,
		"canForceDate":         canForceDate,
		"canUpdateMsg":         canUpdateMsg,
		"canDeleteMsg":         canDeleteMsg,
		"canUpdateAllMsg":      canUpdateAllMsg,
		"canDeleteAllMsg":      canDeleteAllMsg,
		"adminCanUpdateAllMsg": adminCanUpdateAllMsg,
		"adminCanDeleteAllMsg": adminCanDeleteAllMsg,
		"isAutoComputeTags":    isAutoComputeTags,
		"isAutoComputeLabels":  isAutoComputeLabels,
	}

	if parameters != nil {
		update["parameters"] = parameters
	}
	_, err := store.Tat().CTopics.UpdateAll(selector, bson.M{"$set": update})

	if err != nil {
		log.Errorf("Error while updateAll parameters : %s", err.Error())
		return err
	}
	h := fmt.Sprintf("update param to maxlength:%d, maxreplies:%d, canForceDate:%t, canUpdateMsg:%t, canDeleteMsg:%t, canUpdateAllMsg:%t, canDeleteAllMsg:%t, adminCanDeleteAllMsg:%t isAutoComputeTags:%t, isAutoComputeLabels:%t",
		maxLength, maxReplies, canForceDate, canUpdateMsg, canDeleteMsg, canUpdateAllMsg, canDeleteAllMsg, adminCanDeleteAllMsg, isAutoComputeTags, isAutoComputeLabels)

	err = addToHistory(topic, selector, username, h)
	cache.CleanTopicByName(topic.Topic)
	return err
}

func actionOnSetParameter(topic *tat.Topic, operand, set, admin string, newParam tat.TopicParameter, recursive bool, history string) error {

	var selector bson.M

	if recursive {
		selector = bson.M{"topic": bson.RegEx{Pattern: "^" + topic.Topic + ".*$"}}
	} else {
		selector = bson.M{"_id": topic.ID}
	}

	var err error
	if operand == "$pull" {
		_, err = store.Tat().CTopics.UpdateAll(
			selector,
			bson.M{operand: bson.M{set: bson.M{"key": newParam.Key}}},
		)
	} else {
		_, err = store.Tat().CTopics.UpdateAll(
			selector,
			bson.M{operand: bson.M{set: bson.M{"key": newParam.Key, "value": newParam.Value}}},
		)
	}

	if err != nil {
		return err
	}
	return addToHistory(topic, selector, admin, history+" "+newParam.Key+":"+newParam.Value)
}

func actionOnSet(topic *tat.Topic, operand, set, username, admin string, recursive bool, history string) error {

	var selector bson.M

	if recursive {
		selector = bson.M{"topic": bson.RegEx{Pattern: "^" + topic.Topic + ".*$"}}
	} else {
		selector = bson.M{"_id": topic.ID}
	}

	_, err := store.Tat().CTopics.UpdateAll(
		selector,
		bson.M{operand: bson.M{set: username}},
	)

	if err != nil {
		return err
	}
	return addToHistory(topic, selector, admin, history+" "+username)
}

// AddRoUser add a read only user to topic
func AddRoUser(topic *tat.Topic, admin string, username string, recursive bool) error {
	err := actionOnSet(topic, "$addToSet", "roUsers", username, admin, recursive, "add to ro")
	cache.CleanAllTopicsLists()
	return err
}

// AddRwUser add a read write user to topic
func AddRwUser(topic *tat.Topic, admin string, username string, recursive bool) error {
	err := actionOnSet(topic, "$addToSet", "rwUsers", username, admin, recursive, "add to ro")
	cache.CleanAllTopicsLists()
	return err
}

// AddAdminUser add a read write user to topic
func AddAdminUser(topic *tat.Topic, admin string, username string, recursive bool) error {
	err := actionOnSet(topic, "$addToSet", "adminUsers", username, admin, recursive, "add to admin")
	cache.CleanAllTopicsLists()
	return err
}

// RemoveRoUser removes a read only user from topic
func RemoveRoUser(topic *tat.Topic, admin string, username string, recursive bool) error {
	err := actionOnSet(topic, "$pull", "roUsers", username, admin, recursive, "remove from ro")
	cache.CleanAllTopicsLists()
	return err
}

// RemoveAdminUser removes a read only user from topic
func RemoveAdminUser(topic *tat.Topic, admin string, username string, recursive bool) error {
	err := actionOnSet(topic, "$pull", "adminUsers", username, admin, recursive, "remove from admin")
	cache.CleanAllTopicsLists()
	return err
}

// RemoveRwUser removes a read write user from topic
func RemoveRwUser(topic *tat.Topic, admin string, username string, recursive bool) error {
	err := actionOnSet(topic, "$pull", "rwUsers", username, admin, recursive, "remove from rw")
	cache.CleanAllTopicsLists()
	return err
}

// AddRoGroup add a read only group to topic
func AddRoGroup(topic *tat.Topic, admin string, groupname string, recursive bool) error {
	err := actionOnSet(topic, "$addToSet", "roGroups", groupname, admin, recursive, "add to ro")
	cache.CleanTopicByName(topic.Topic)
	return err
}

// AddRwGroup add a read write group to topic
func AddRwGroup(topic *tat.Topic, admin string, groupname string, recursive bool) error {
	err := actionOnSet(topic, "$addToSet", "rwGroups", groupname, admin, recursive, "add to ro")
	cache.CleanTopicByName(topic.Topic)
	return err
}

// AddAdminGroup add a admin group to topic
func AddAdminGroup(topic *tat.Topic, admin string, groupname string, recursive bool) error {
	err := actionOnSet(topic, "$addToSet", "adminGroups", groupname, admin, recursive, "add to admin")
	cache.CleanTopicByName(topic.Topic)
	return err
}

// RemoveAdminGroup removes a read write group from topic
func RemoveAdminGroup(topic *tat.Topic, admin string, groupname string, recursive bool) error {
	err := actionOnSet(topic, "$pull", "adminGroups", groupname, admin, recursive, "remove from admin")
	cache.CleanTopicByName(topic.Topic)
	return err
}

// RemoveRoGroup removes a read only group from topic
func RemoveRoGroup(topic *tat.Topic, admin string, groupname string, recursive bool) error {
	err := actionOnSet(topic, "$pull", "roGroups", groupname, admin, recursive, "remove from ro")
	cache.CleanTopicByName(topic.Topic)
	return err
}

// RemoveRwGroup removes a read write group from topic
func RemoveRwGroup(topic *tat.Topic, admin string, groupname string, recursive bool) error {
	err := actionOnSet(topic, "$pull", "rwGroups", groupname, admin, recursive, "remove from rw")
	cache.CleanTopicByName(topic.Topic)
	return err
}

// AddFilter add a user filter to the topic
func AddFilter(topic *tat.Topic, user *tat.User, filter *tat.Filter) error {

	filter.ID = bson.NewObjectId().Hex()
	filter.UserID = user.ID
	filter.Username = user.Username

	for _, h := range filter.Hooks {
		h.ID = bson.NewObjectId().Hex()
	}

	err := store.Tat().CTopics.Update(
		bson.M{"_id": topic.ID},
		bson.M{"$addToSet": bson.M{"filters": filter}},
	)
	cache.CleanTopicByName(topic.Topic)
	return err
}

// RemoveFilter add a user filter to the topic
func RemoveFilter(topic *tat.Topic, filter *tat.Filter) error {
	err := store.Tat().CTopics.Update(
		bson.M{"_id": topic.ID},
		bson.M{"$pull": bson.M{"filters": bson.M{"_id": filter.ID}}},
	)
	cache.CleanTopicByName(topic.Topic)
	return err
}

// UpdateFilter add a user filter to the topic
func UpdateFilter(topic *tat.Topic, filter *tat.Filter) error {
	err := store.Tat().CTopics.Update(
		bson.M{"_id": topic.ID, "filters._id": filter.ID},
		bson.M{"$set": bson.M{"filters.$": filter}},
	)
	cache.CleanTopicByName(topic.Topic)
	return err
}

// AddParameter add a parameter to the topic
func AddParameter(topic *tat.Topic, admin string, parameterKey string, parameterValue string, recursive bool) error {
	return actionOnSetParameter(topic, "$addToSet", "parameters", admin, tat.TopicParameter{Key: parameterKey, Value: parameterValue}, recursive, "add to parameter")
}

// RemoveParameter removes a read only user from topic
func RemoveParameter(topic *tat.Topic, admin string, parameterKey string, parameterValue string, recursive bool) error {
	return actionOnSetParameter(topic, "$pull", "parameters", admin, tat.TopicParameter{Key: parameterKey, Value: ""}, recursive, "remove from parameters")
}

func addToHistory(topic *tat.Topic, selector bson.M, user string, historyToAdd string) error {
	toAdd := strconv.FormatInt(time.Now().Unix(), 10) + " " + user + " " + historyToAdd
	_, err := store.Tat().CTopics.UpdateAll(
		selector,
		bson.M{"$addToSet": bson.M{"history": toAdd}},
	)
	return err
}

// GetUserRights return isRW, isAdmin for user
// Check personal access to topic, and group access
func GetUserRights(topic *tat.Topic, user *tat.User) (bool, bool) {

	isUserAdmin := tat.ArrayContains(topic.AdminUsers, user.Username)
	if isUserAdmin {
		return true, true
	}

	userGroups, err := group.GetGroups(user.Username)
	if err != nil {
		log.Errorf("Error while fetching user groups")
		return false, false
	}

	var groups []string
	for _, g := range userGroups {
		groups = append(groups, g.Name)
	}

	isUserRW := tat.ArrayContains(topic.RWUsers, user.Username)
	isRW := isUserRW || tat.ItemInBothArrays(topic.RWGroups, groups)
	isAdmin := isUserAdmin || tat.ItemInBothArrays(topic.AdminUsers, groups)
	return isRW, isAdmin
}

// IsUserAdmin return true if user is Tat admin or is admin on this topic
// Check personal access to topic, and group access
func IsUserAdmin(topic *tat.Topic, user *tat.User) bool {

	if user.IsAdmin {
		return true
	}

	if tat.ArrayContains(topic.AdminUsers, user.Username) {
		return true
	}

	userGroups, err := group.GetGroups(user.Username)
	if err != nil {
		log.Errorf("Error while fetching user groups")
		return false
	}

	var groups []string
	for _, g := range userGroups {
		groups = append(groups, g.Name)
	}

	if tat.ItemInBothArrays(topic.AdminGroups, groups) {
		return true
	}

	// user is "Admin" on his /Private/usrname topics
	return strings.HasPrefix(topic.Topic, "/Private/"+user.Username)
}

// CheckAndFixName Add a / to topic name is it is not present
// return an error if length of name is < 4 or > 100
func CheckAndFixName(topic *tat.Topic) error {
	name, err := tat.CheckAndFixNameTopic(topic.Topic)
	if err != nil {
		return err
	}
	topic.Topic = name
	return nil
}

// ChangeUsernameOnTopics changes a username on topics, ro, rw, admin users
func ChangeUsernameOnTopics(oldUsername, newUsername string) {
	changeNameOnSet("username", "roUsers", oldUsername, newUsername)
	changeNameOnSet("username", "rwUsers", oldUsername, newUsername)
	changeNameOnSet("username", "adminUsers", oldUsername, newUsername)
	changeUsernameOnPrivateTopics(oldUsername, newUsername)
}

// ChangeGroupnameOnTopics updates group name on topics
func ChangeGroupnameOnTopics(oldGroupname, newGroupname string) error {
	if err := changeNameOnSet("groupname", "roGroups", oldGroupname, newGroupname); err != nil {
		return err
	}
	if err := changeNameOnSet("groupname", "rwGroups", oldGroupname, newGroupname); err != nil {
		return err
	}
	if err := changeNameOnSet("groupname", "adminGroups", oldGroupname, newGroupname); err != nil {
		return err
	}
	return nil
}

func changeNameOnSet(typeChange, set, oldname, newname string) error {
	_, err := store.Tat().CTopics.UpdateAll(
		bson.M{set: oldname},
		bson.M{"$set": bson.M{set + ".$": newname}})

	if err != nil {
		log.Errorf("Error while changes %s from %s to %s on Topics (%s) %s", typeChange, oldname, newname, set, err)
		return fmt.Errorf("Error while changes %s from %s to %s on Topics (%s)", typeChange, oldname, newname, set)
	}
	return nil
}

func changeUsernameOnPrivateTopics(oldUsername, newUsername string) error {
	var topics []tat.Topic

	err := store.Tat().CTopics.Find(
		bson.M{
			"topic": bson.RegEx{
				Pattern: "^/Private/" + oldUsername + ".*$", Options: "i",
			}}).All(&topics)

	if err != nil {
		log.Errorf("Error while getting topic with username %s for rename to %s on Topics %s", oldUsername, newUsername, err)
	}

	for _, topic := range topics {
		newTopicName := strings.Replace(topic.Topic, oldUsername, newUsername, 1)
		errUpdate := store.Tat().CTopics.Update(
			bson.M{"_id": topic.ID},
			bson.M{"$set": bson.M{"topic": newTopicName}},
		)
		if errUpdate != nil {
			log.Errorf("Error while update Topic name from %s to %s :%s", topic.Topic, newTopicName, errUpdate)
		}
	}

	return err
}

// MigrateToDedicatedTopic sets collection  attribute on topic
func MigrateToDedicatedTopic(topic *tat.Topic) error {

	if topic.Collection != "" {
		return fmt.Errorf("MigrateToDedicatedTopic> This topic is already dedicated on a collection")
	}

	topic.Collection = "messages" + topic.ID

	errUpdate := store.Tat().CTopics.Update(
		bson.M{"_id": topic.ID},
		bson.M{"$set": bson.M{"collection": topic.Collection}},
	)
	if errUpdate != nil {
		return fmt.Errorf("MigrateToDedicatedTopic> Error while update Topic collection:%s", errUpdate)
	}

	store.EnsureIndexesMessages(topic.Collection)

	return nil
}
