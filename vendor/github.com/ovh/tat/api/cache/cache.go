package cache

import (
	"strings"

	"gopkg.in/redis.v4"

	log "github.com/Sirupsen/logrus"
	"github.com/ovh/tat"
	"github.com/spf13/viper"
)

var instance Cache
var needToFlush bool

//Client returns Cache interface
func Client() Cache {
	redisHosts := viper.GetString("redis_hosts")
	redisMaster := viper.GetString("redis_master")
	redisSentinels := viper.GetString("redis_sentinels")
	redisPassword := viper.GetString("redis_password")
	redisHostsArray := strings.Split(redisHosts, ",")
	redisSentinelsArray := strings.Split(redisSentinels, ",")

	if instance != nil {
		goto testInstance
	}

	if redisHosts == "" && redisSentinels == "" {
		//Mode in memory
		log.Warningf("Configuring fake redis client. You should consider to start at least a standalone redis client.")
		instance = &LocalCache{}
		goto testInstance
	}

	if len(redisHostsArray) > 1 {
		//Mode cluster
		log.Infof("Configuring Redis Cluster client for %s", redisHosts)
		opts := &redis.ClusterOptions{
			Addrs:    redisHostsArray,
			Password: redisPassword,
		}
		instance = redis.NewClusterClient(opts)
		FlushDB()
		goto testInstance
	}

	if len(redisHostsArray) == 1 && redisHosts != "" {
		//Mode master
		log.Infof("Configuring Redis client for %s", redisHosts)
		opts := &redis.Options{
			Addr:     redisHosts,
			Password: redisPassword,
		}
		instance = redis.NewClient(opts)
		FlushDB()
		goto testInstance
	}

	if len(redisSentinelsArray) > 1 && redisMaster != "" {
		//Mode sentinels
		log.Infof("Configuring Failover Redis client for master %s on sentinels %s", redisMaster, redisSentinels)
		opts := &redis.FailoverOptions{
			MasterName:    redisMaster,
			Password:      redisPassword,
			SentinelAddrs: redisSentinelsArray,
		}
		instance = redis.NewFailoverClient(opts)
		FlushDB()
		goto testInstance
	}

	log.Errorf("Invalid Redis configuration. For Redis Cluster: use --redis-hosts=my-redis-host1.local:6379,my-redis-host2.local:6379. For Redis Sentinels : use --redis-master=mymaster --redis-sentinels=my-redis-host1.local:26379,my-redis-host2.local:26379. For Standalone Redis:  --redis-hosts=my-redis-host1.local:6379")
	log.Errorf("Configuring fake Redis client. You should consider to fix your configuration.")
	instance = &LocalCache{}

testInstance:
	if needToFlush {
		FlushDB()
	}

	if err := instance.Ping().Err(); err != nil {
		log.Errorf("Unable to ping Redis at %s", err)
		needToFlush = true
	} else {
		needToFlush = false
	}

	return instance
}

// TestInstanceAtStartup pings redis and display error log if no redis, and Info
// log is redis is here
func TestInstanceAtStartup() {
	if viper.GetString("redis_hosts") == "" && viper.GetString("redis_sentinels") == "" {
		log.Infof("TAT is NOT linked to a redis")
		return
	}
	Client()
}

//CriteriaKey returns the Redis Key
func CriteriaKey(i tat.CacheableCriteria, s ...string) string {
	k := i.CacheKey()
	return Key(s...) + ":" + Key(k...)
}

//Key convert string array in redis key
func Key(s ...string) string {
	var escape = func(s string) string {
		r := strings.Replace(s, ":", "_", -1)
		return strings.Replace(r, " ", "", -1)
	}

	for i := range s {
		s[i] = escape(s[i])
	}

	return strings.Join(s, ":")
}

func removeSomeMembers(pipeline *redis.Pipeline, key string, members ...string) {

	m := make([]interface{}, len(members))
	for i, member := range members {
		m[i] = member
	}
	pipeline.SRem(key, m...)
}
