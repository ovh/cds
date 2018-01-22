package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/ovh/tat"
	"github.com/ovh/tat/api/group"
	"github.com/ovh/tat/api/message"
	"github.com/ovh/tat/api/presence"
	"github.com/ovh/tat/api/store"
	"github.com/ovh/tat/api/topic"
	"github.com/ovh/tat/api/user"
)

// StatsController contains all methods about stats
type StatsController struct{}

// Count returns total number of messages
func (*StatsController) Count(ctx *gin.Context) {

	nbGroups, err := group.CountGroups()
	if err != nil {
		log.Errorf("Error while count all groups %s", err)
		nbGroups = -1
	}

	nbMessages, err := message.CountAllMessages()
	if err != nil {
		log.Errorf("Error while count all messages %s", err)
		nbMessages = -1
	}

	nbPresences, err := presence.CountPresences()
	if err != nil {
		log.Errorf("Error while count all presences %s", err)
		nbPresences = -1
	}
	nbTopics, err := topic.CountTopics()
	if err != nil {
		log.Errorf("Error while count all topics %s", err)
		nbTopics = -1
	}
	nbUsers, err := user.CountUsers()
	if err != nil {
		log.Errorf("Error while count all users %s", err)
		nbUsers = -1
	}

	now := time.Now()

	out := tat.StatsCountJSON{
		Date:      now.Unix(),
		DateHuman: now,
		Version:   tat.Version,
		Groups:    nbGroups,
		Messages:  nbMessages,
		Presences: nbPresences,
		Topics:    nbTopics,
		Users:     nbUsers,
	}
	ctx.JSON(http.StatusOK, out)
}

// Instance returns information about current engine
func (*StatsController) Instance(ctx *gin.Context) {

	hostname, err := os.Hostname()
	if err != nil {
		log.Errorf("Error while getting Hostname %s", err)
		hostname = fmt.Sprintf("Error while getting Hostname: %s", err.Error())
	}

	now := time.Now()
	ctx.JSON(http.StatusOK, gin.H{
		"date":      now.Unix(),
		"dateHuman": now,
		"version":   tat.Version,
		"hostname":  hostname,
		"ips":       externalIP(),
	})

}

func externalIP() string {
	ips := ""
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Errorf("Error while getting net.Interfaces %s", err.Error())
		return err.Error()
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			log.Errorf("Error while getting iface.Addrs %s", err.Error())
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			ips += iface.Name + ":" + ip.String() + ","
		}
	}
	if ips != "" {
		return ips
	}
	return "are you connected to the network?"
}

// DistributionTopics returns total number of messages
func (*StatsController) DistributionTopics(ctx *gin.Context) {
	c := &tat.TopicCriteria{}
	skip, e := strconv.Atoi(ctx.DefaultQuery("skip", "0"))
	if e != nil {
		skip = 0
	}
	c.Skip = skip
	limit, e2 := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	if e2 != nil {
		limit = 20
	}
	c.Limit = limit

	count, topics, err := topic.ListTopics(c, nil, true, false, false)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error while listing topics %s", err)})
		return
	}

	info := ""
	t := []tat.TopicDistributionJSON{}
	for _, topic := range topics {
		countMsg, err := message.CountMessages(&tat.MessageCriteria{Topic: topic.Topic}, topic)
		if err != nil {
			info += fmt.Sprintf("Error on topic %s: %s", topic.Topic, err)
		}
		t = append(t, tat.TopicDistributionJSON{
			ID:         topic.ID,
			Topic:      topic.Topic,
			Count:      countMsg,
			Dedicated:  topic.Collection != "",
			Collection: topic.Collection,
		})
	}

	out := tat.StatsDistributionTopicsJSON{
		Total:  count,
		Info:   info,
		Topics: t,
	}

	ctx.JSON(http.StatusOK, out)
}

// DBServerStatus returns stats of db : serverStatus
func (*StatsController) DBServerStatus(ctx *gin.Context) {
	serverStatus, err := store.DBServerStatus()
	if err != nil {
		log.Errorf("Error while get DBServerStatus of db server %s", err)
	}

	now := time.Now()
	ctx.JSON(http.StatusOK, gin.H{
		"date":         now.Unix(),
		"dateHuman":    now,
		"serverStatus": serverStatus,
	})
}

// DBStats returns stats of db : dbstats
func (*StatsController) DBStats(ctx *gin.Context) {
	dbstats, err := store.DBStats()
	if err != nil {
		log.Errorf("Error while get DBStats of db server %s", err)
	}

	now := time.Now()
	ctx.JSON(http.StatusOK, gin.H{
		"date":      now.Unix(),
		"dateHuman": now,
		"dbstats":   dbstats,
	})
}

// DBReplSetGetConfig returns rs.conf() mongo cmd
func (*StatsController) DBReplSetGetConfig(ctx *gin.Context) {
	replSetGetConfig, err := store.DBReplSetGetConfig()
	if err != nil {
		log.Errorf("Error while get DBReplSetGetConfig of db server %s", err)
	}

	now := time.Now()
	ctx.JSON(http.StatusOK, gin.H{
		"date":             now.Unix(),
		"dateHuman":        now,
		"replSetGetConfig": replSetGetConfig,
	})
}

// DBReplSetGetStatus returns stats of db : replSetGetStatus
func (*StatsController) DBReplSetGetStatus(ctx *gin.Context) {
	replSetGetStatus, err := store.DBReplSetGetStatus()
	if err != nil {
		log.Errorf("Error while get DBReplSetGetStatus of db server %s", err)
	}

	now := time.Now()
	ctx.JSON(http.StatusOK, gin.H{
		"date":         now.Unix(),
		"dateHuman":    now,
		"serverStatus": replSetGetStatus,
	})
}

// DBStatsCollections returns stats of each collections
func (*StatsController) DBStatsCollections(ctx *gin.Context) {

	collNames, err := store.GetCollectionNames()

	now := time.Now()
	g := gin.H{
		"date":      now.Unix(),
		"dateHuman": now,
		"version":   tat.Version,
	}
	if err != nil {
		log.Errorf("Error while getting collectionNames %s", err)
	}

	for _, collName := range collNames {
		v, err := store.DBStatsCollection(collName)
		if err != nil {
			g[collName] = "error"
			log.Errorf("Error while getting stats for collection %s, error: %s", collName, err)
		} else {
			g[collName] = v
		}
	}

	ctx.JSON(http.StatusOK, g)
}

// DBGetSlowestQueries returns the slowest queries
func (*StatsController) DBGetSlowestQueries(ctx *gin.Context) {
	queries, err := store.GetSlowestQueries()
	if err != nil {
		log.Errorf("Error while getting the slowest queries %s", err)
	}

	now := time.Now()
	ctx.JSON(http.StatusOK, gin.H{
		"date":           now.Unix(),
		"dateHuman":      now,
		"slowestQueries": queries,
	})
}

// CheckHeaders drop headers (admin route)
func (*StatsController) CheckHeaders(ctx *gin.Context) {
	g := gin.H{}
	for k, v := range ctx.Request.Header {
		g[k] = v
	}

	ctx.JSON(http.StatusOK, g)
}
