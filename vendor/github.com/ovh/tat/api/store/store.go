package store

import (
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	// DatabaseName is DatabaseName on mongoBD
	DatabaseName     = "tat"
	collectionGroups = "groups"
	// CollectionDefaultMessages is default names for collections message, if
	// topic doesn't have "collection" attribute setted
	CollectionDefaultMessages = "messages"
	collectionPresences       = "presences"
	collectionTopics          = "topics"
	collectionUsers           = "users"
	collectionSockets         = "sockets"
)

// MongoStore stores MongoDB Session and collections
type MongoStore struct {
	Session          *mgo.Session
	CGroups          *mgo.Collection
	CDefaultMessages *mgo.Collection
	CPresences       *mgo.Collection
	CTopics          *mgo.Collection
	CUsers           *mgo.Collection
	CSockets         *mgo.Collection
}

var _instance *MongoStore

// Tat returns mongoDB instance for Tat
func Tat() *MongoStore {
	return _instance
}

// NewStore initializes a new MongoDB Store
func NewStore() error {
	log.Info("Mongodb : create new instance")
	var session *mgo.Session
	var err error

	username := getDbParameter("db_user")
	password := getDbParameter("db_password")
	replicaSetHostnamesTags := getDbParameter("db_rs_tags")

	address := viper.GetString("db_addr")
	if username != "" && password != "" {
		session, err = mgo.Dial("mongodb://" + username + ":" + password + "@" + address)
	} else {
		session, err = mgo.Dial("mongodb://" + address)
	}

	session.SetSocketTimeout(time.Duration(viper.GetInt("db_socket_timeout")) * time.Second)

	if err != nil {
		log.Errorf("Error with mgo.Dial %s", err.Error())
		return err
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Errorf("Error with getting hostname: %s", err.Error())
	}

	session.Refresh()
	session.SetMode(mgo.SecondaryPreferred, true)
	if viper.GetInt("db_ensure_safe_db_write") > 0 {
		session.EnsureSafe(&mgo.Safe{W: viper.GetInt("db_ensure_safe_db_write"), FSync: true})
	}

	if replicaSetHostnamesTags != "" && hostname != "" {
		log.Warnf("SelectServers try selectServer for %s with values %s", hostname, replicaSetHostnamesTags)
		tuples := strings.Split(replicaSetHostnamesTags, ",")
		for _, tuple := range tuples {
			t := strings.Split(tuple, ":")
			tupleHostname := t[0]
			if tupleHostname == hostname {
				tupleTagName := t[1]
				tupleTagValue := t[2]
				log.Warnf("SelectServers attach %s on replicaSet with tagName %s and value %s and %s", hostname, tupleTagName, tupleTagValue)
				session.SelectServers(bson.D{{Name: tupleTagName, Value: tupleTagValue}})
				break
			}
		}
	} else {
		log.Debugf("SelectServers No prefered server to select : %s", replicaSetHostnamesTags)
	}

	if err != nil {
		log.Errorf("Error with getting Mongodb.Instance on address %s : %s", address, err)
		return err
	}

	_instance = &MongoStore{
		Session:          session,
		CGroups:          session.DB(DatabaseName).C(collectionGroups),
		CDefaultMessages: session.DB(DatabaseName).C(CollectionDefaultMessages),
		CPresences:       session.DB(DatabaseName).C(collectionPresences),
		CTopics:          session.DB(DatabaseName).C(collectionTopics),
		CUsers:           session.DB(DatabaseName).C(collectionUsers),
		CSockets:         session.DB(DatabaseName).C(collectionSockets),
	}

	EnsureIndexes()
	return nil
}

// getDbParameter gets value of tat parameter
// return values if not "" AND not "false"
// used by db_user, db_password and db_rs_tags
func getDbParameter(key string) string {
	value := ""
	if viper.GetString(key) != "" && viper.GetString(key) != "false" {
		value = viper.GetString(key)
	}
	return value
}

// EnsureIndexes fixes index at startup
func EnsureIndexes() {
	//listIndex(_instance.CTopics, false)
	//listIndex(_instance.CGroups, false)
	//listIndex(_instance.CUsers, false)
	//listIndex(_instance.CPresences, false)

	// messages
	EnsureIndexesMessages(CollectionDefaultMessages)

	// topics
	ensureIndex(_instance.CTopics, mgo.Index{Key: []string{"topic"}, Unique: true})

	// groups
	ensureIndex(_instance.CGroups, mgo.Index{Key: []string{"name"}, Unique: true})

	// users
	ensureIndex(_instance.CUsers, mgo.Index{Key: []string{"username"}, Unique: true})
	ensureIndex(_instance.CUsers, mgo.Index{Key: []string{"email"}, Unique: true})

	// presences
	ensureIndex(_instance.CPresences, mgo.Index{Key: []string{"topic", "-dateTimePresence"}})
	ensureIndex(_instance.CPresences, mgo.Index{Key: []string{"userPresence.username", "-datePresence"}})
	ensureIndex(_instance.CPresences, mgo.Index{Key: []string{"topic", "userPresence.username"}, Unique: true})
}

// EnsureIndexesMessages set indexes on a message collection
func EnsureIndexesMessages(collection string) {

	if collection != CollectionDefaultMessages {
		//listIndex(_instance.Session.DB(DatabaseName).C(collection), true)
		ensureIndex(_instance.Session.DB(DatabaseName).C(collection), mgo.Index{Key: []string{"-dateUpdate", "-dateCreation"}})
		ensureIndex(_instance.Session.DB(DatabaseName).C(collection), mgo.Index{Key: []string{"-dateCreation"}})
		ensureIndex(_instance.Session.DB(DatabaseName).C(collection), mgo.Index{Key: []string{"tags"}})
		ensureIndex(_instance.Session.DB(DatabaseName).C(collection), mgo.Index{Key: []string{"labels.text"}})
		ensureIndex(_instance.Session.DB(DatabaseName).C(collection), mgo.Index{Key: []string{"inReplyOfID"}})
		ensureIndex(_instance.Session.DB(DatabaseName).C(collection), mgo.Index{Key: []string{"inReplyOfIDRoot"}})
	} else {
		//listIndex(_instance.Session.DB(DatabaseName).C(collection), false)
		ensureIndex(_instance.Session.DB(DatabaseName).C(collection), mgo.Index{Key: []string{"topic", "-dateUpdate", "-dateCreation"}})
		ensureIndex(_instance.Session.DB(DatabaseName).C(collection), mgo.Index{Key: []string{"topic", "-dateCreation"}})
		ensureIndex(_instance.Session.DB(DatabaseName).C(collection), mgo.Index{Key: []string{"topic", "tags"}})
		ensureIndex(_instance.Session.DB(DatabaseName).C(collection), mgo.Index{Key: []string{"topic", "labels.text"}})
		ensureIndex(_instance.Session.DB(DatabaseName).C(collection), mgo.Index{Key: []string{"inReplyOfID"}})
		ensureIndex(_instance.Session.DB(DatabaseName).C(collection), mgo.Index{Key: []string{"inReplyOfIDRoot"}})
	}
}

func listIndex(col *mgo.Collection, drop bool) {
	indexes, err := col.Indexes()
	if err != nil {
		log.Warnf("Error while getting index: %s", err)
	}
	for _, index := range indexes {
		if strings.HasPrefix(index.Key[0], "_id") {
			continue
		}
		log.Warnf("Info Index : Col %s : %+v - toRemove %t", col.Name, index, drop)
		if drop {
			if err := col.DropIndex(index.Key...); err != nil {
				log.Warnf("Error while dropping index: %s", err)
			}
		}
	}
}

func ensureIndex(col *mgo.Collection, index mgo.Index) {
	if err := col.EnsureIndex(index); err != nil {
		log.Fatalf("Error while creating index on %s:%s", col.Name, err)
		return
	}
}

// RefreshStore calls Refresh on mongoDB Store, in order to avoid lost connection
func RefreshStore() {
	_instance.Session.Refresh()
}

// DBServerStatus returns serverStatus cmd
func DBServerStatus() (bson.M, error) {
	result := bson.M{}
	err := _instance.Session.Run(bson.D{{Name: "serverStatus", Value: 1}}, &result)
	return result, err
}

// DBStats returns dbstats cmd
func DBStats() (bson.M, error) {
	result := bson.M{}
	err := _instance.Session.DB("tat").Run(bson.D{{Name: "dbStats", Value: 1}, {Name: "scale", Value: 1024}}, &result)
	return result, err
}

// GetCollectionNames returns collection names
func GetCollectionNames() ([]string, error) {
	return _instance.Session.DB("tat").CollectionNames()
}

// DBStatsCollection returns stats for given collection
func DBStatsCollection(colName string) (bson.M, error) {
	result := bson.M{}
	err := _instance.Session.DB("tat").Run(bson.D{{Name: "collStats", Value: colName},
		{Name: "scale", Value: 1024},
		{Name: "indexDetails", Value: true},
	}, &result)
	return result, err
}

// DBReplSetGetStatus returns replSetGetStatus cmd
func DBReplSetGetStatus() (bson.M, error) {
	result := bson.M{}
	err := _instance.Session.Run(bson.D{{Name: "replSetGetStatus", Value: 1}}, &result)
	return result, err
}

// DBReplSetGetConfig returns replSetGetConfig cmd
func DBReplSetGetConfig() (bson.M, error) {
	result := bson.M{}
	err := _instance.Session.Run(bson.D{{Name: "replSetGetConfig", Value: 1}}, &result)
	return result, err
}

// GetSlowestQueries returns the slowest queries
func GetSlowestQueries() ([]map[string]interface{}, error) {
	col := _instance.Session.DB("tat").C("system.profile")
	var results []map[string]interface{}
	err := col.Find(bson.M{}).
		Sort("-millis").
		Skip(0).
		Limit(10).
		All(&results)
	return results, err
}

// GetCMessages return mgo collection
func GetCMessages(collection string) *mgo.Collection {
	if collection != "" {
		return _instance.Session.DB(DatabaseName).C(collection)
	}
	return _instance.CDefaultMessages
}
