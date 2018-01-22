package presence

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/ovh/tat"
	"github.com/ovh/tat/api/store"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func buildPresenceCriteria(criteria *tat.PresenceCriteria) (bson.M, error) {
	var query = []bson.M{}

	if criteria.Status != "" {
		queryStatus := bson.M{}
		queryStatus["$or"] = []bson.M{}
		for _, val := range strings.Split(criteria.Status, ",") {
			queryStatus["$or"] = append(queryStatus["$or"].([]bson.M), bson.M{"status": bson.RegEx{Pattern: "^.*" + regexp.QuoteMeta(val) + ".*$", Options: "i"}})
		}
		query = append(query, queryStatus)
	}
	if criteria.Username != "" {
		queryUsernames := bson.M{}
		queryUsernames["$or"] = []bson.M{}
		for _, val := range strings.Split(criteria.Username, ",") {
			queryUsernames["$or"] = append(queryUsernames["$or"].([]bson.M), bson.M{"userPresence.username": val})
		}
		query = append(query, queryUsernames)
	}
	if criteria.Topic != "" {
		queryTopics := bson.M{}
		queryTopics["$or"] = []bson.M{}
		for _, val := range strings.Split(criteria.Topic, ",") {
			queryTopics["$or"] = append(queryTopics["$or"].([]bson.M), bson.M{"topic": val})
		}
		query = append(query, queryTopics)
	}

	var bsonDate = bson.M{}

	if criteria.DateMinPresence != "" {
		i, err := strconv.ParseInt(criteria.DateMinPresence, 10, 64)
		if err != nil {
			return bson.M{}, fmt.Errorf("Error while parsing dateMinPresence %s", err)
		}
		tm := time.Unix(i, 0)
		bsonDate["$gte"] = tm.Unix()
	}
	if criteria.DateMaxPresence != "" {
		i, err := strconv.ParseInt(criteria.DateMaxPresence, 10, 64)
		if err != nil {
			return bson.M{}, fmt.Errorf("Error while parsing dateMaxPresence %s", err)
		}
		tm := time.Unix(i, 0)
		bsonDate["$lte"] = tm.Unix()
	}
	if len(bsonDate) > 0 {
		query = append(query, bson.M{"datePresence": bsonDate})
	}

	if len(query) > 0 {
		return bson.M{"$and": query}, nil
	} else if len(query) == 1 {
		return query[0], nil
	}
	return bson.M{}, nil
}

func getFieldsPresence(allFields bool) bson.M {

	if allFields {
		return bson.M{}
	}

	return bson.M{"_id": 0,
		"status":           1,
		"dateTimePresence": 1,
		"datePresence":     1,
		"userPresence":     1,
		"topic":            1,
	}
}

// ListPresencesAllFields returns list of presences, with given criteria
func ListPresencesAllFields(criteria *tat.PresenceCriteria) (int, []tat.Presence, error) {
	return listPresencesInternal(criteria, true)
}

// ListPresences returns list of presences, but only field status, dateTimePresence,datePresence,userPresence
func ListPresences(criteria *tat.PresenceCriteria) (int, []tat.Presence, error) {
	return listPresencesInternal(criteria, false)
}

func listPresencesInternal(criteria *tat.PresenceCriteria, allFields bool) (int, []tat.Presence, error) {
	var presences []tat.Presence

	cursor, errl := listPresencesCursor(criteria, allFields)
	if errl != nil {
		return -1, presences, errl
	}
	count, err := cursor.Count()
	if err != nil {
		return -1, presences, fmt.Errorf("Error while count Presences %s", err)
	}

	sortBy := criteria.SortBy
	if sortBy == "" {
		sortBy = "-datePresence"
	}
	err = cursor.Select(getFieldsPresence(allFields)).
		Sort(sortBy).
		Skip(criteria.Skip).
		Limit(criteria.Limit).
		All(&presences)

	if err != nil {
		log.Errorf("Error while Find All Presences %s", err)
	}
	return count, presences, err
}

func listPresencesCursor(criteria *tat.PresenceCriteria, allFields bool) (*mgo.Query, error) {
	c, err := buildPresenceCriteria(criteria)
	if err != nil {
		return nil, err
	}
	return store.Tat().CPresences.Find(c), nil
}

// Upsert insert of update a presence (user / topic)
func Upsert(presence *tat.Presence, user tat.User, topic tat.Topic, status string) error {
	presence.Status = status
	if err := checkAndFixStatus(presence); err != nil {
		return err
	}
	//	presence.ID = bson.NewObjectId().Hex()
	presence.Topic = topic.Topic
	var userPresence = tat.UserPresence{}
	userPresence.Username = user.Username
	userPresence.Fullname = user.Fullname
	presence.UserPresence = userPresence
	now := time.Now()
	presence.DatePresence = now.Unix()
	presence.DateTimePresence = time.Now()

	//selector := ]bson.M{}
	//selector = append(selector, bson.M{"userpresence.username": userPresence.Username})
	//selector = append(selector, bson.M{"topic": topic.Topic})
	selector := bson.M{"topic": topic.Topic, "userPresence.username": userPresence.Username}
	if _, err := store.Tat().CPresences.Upsert(selector, presence); err != nil {
		log.Errorf("Error while inserting new presence for %s err:%s", userPresence.Username, err)
	}
	return nil
}

// truncate to 140 characters
// if len < 1, return error
func checkAndFixStatus(presence *tat.Presence) error {
	status := strings.TrimSpace(presence.Status)
	if len(status) < 1 {
		return fmt.Errorf("Invalid Status:%s", presence.Status)
	}

	validStatus := [...]string{"online", "offline", "busy"}
	find := false
	for _, s := range validStatus {
		if s == presence.Status {
			find = true
			break
		}
	}

	if !find {
		return fmt.Errorf("Invalid Status, should be online or offline or busy :%s", presence.Status)
	}
	presence.Status = status
	return nil
}

// ChangeAuthorUsernameOnPresences changes username on presences collection
func ChangeAuthorUsernameOnPresences(oldUsername, newUsername string) error {
	_, err := store.Tat().CPresences.UpdateAll(
		bson.M{"userPresence.username": oldUsername},
		bson.M{"$set": bson.M{"userPresence.username": newUsername}})

	if err != nil {
		log.Errorf("Error while update username from %s to %s on Presences %s", oldUsername, newUsername, err)
	}

	return err
}

// CountPresences returns the total number of presences in db
func CountPresences() (int, error) {
	return store.Tat().CPresences.Count()
}

// Delete all presences of one user on one topic
func Delete(user tat.User, topic tat.Topic) error {
	_, err := store.Tat().CPresences.RemoveAll(bson.M{"userPresence.username": user.Username, "topic": topic.Topic})
	return err
}

// CheckAllPresences detects duplicate
func CheckAllPresences() ([]bson.M, error) {
	pipeline := []bson.M{
		{"$group": bson.M{"_id": bson.M{"username": "$userPresence.username", "topic": "$topic"}, "count": bson.M{"$sum": 1}}},
		{"$match": bson.M{"count": bson.M{"$gt": 1}}},
	}

	pipe := store.Tat().CPresences.Pipe(pipeline)
	results := []bson.M{}

	err := pipe.All(&results)
	return results, err
}
