package main

import (
	"fmt"
	"io/ioutil"
	"os/user"
	"strings"

	"github.com/spf13/viper"

	"os"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/sdk"
)

const (
	viperURLAPI                         = "url.api"
	viperURLUI                          = "url.ui"
	viperServerHTTPPort                 = "server.http.port"
	viperServerSessionTTL               = "server.http.sessionTTL"
	viperServerGRPCPort                 = "server.grpc.port"
	viperServerSecretKey                = "server.secrets.key"
	viperServerSecretBackend            = "server.secrets.backend"
	viperServerSecretBackendOption      = "server.secrets.backend.option"
	viperLogLevel                       = "log.level"
	viperLogDBLogging                   = "log.db"
	viperDBUser                         = "db.user"
	viperDBPassword                     = "db.password"
	viperDBName                         = "db.name"
	viperDBHost                         = "db.host"
	viperDBPort                         = "db.port"
	viperDBSSLMode                      = "db.sslmode"
	viperDBMaxConn                      = "db.maxconn"
	viperDBTimeout                      = "db.timeout"
	viperDBSecret                       = "db.secret"
	viperCacheMode                      = "cache.mode"
	viperCacheTTL                       = "cache.ttl"
	viperCacheRedisHost                 = "cache.redis.host"
	viperCacheRedisPassword             = "cache.redis.password"
	viperDownloadDirectory              = "directories.download"
	viperKeysDirectory                  = "directories.keys"
	viperAuthMode                       = "auth.localmode"
	viperAuthLDAPEnable                 = "auth.ldap.enable"
	viperAuthLDAPHost                   = "auth.ldap.host"
	viperAuthLDAPPort                   = "auth.ldap.port"
	viperAuthLDAPSSL                    = "auth.ldap.ssl"
	viperAuthLDAPBase                   = "auth.ldap.base"
	viperAuthLDAPDN                     = "auth.ldap.dn"
	viperAuthLDAPFullname               = "auth.ldap.fullname"
	viperAuthDefaultGroup               = "auth.defaultgroup"
	viperSMTPDisable                    = "smtp.disable"
	viperSMTPHost                       = "smtp.host"
	viperSMTPPort                       = "smtp.port"
	viperSMTPTLS                        = "smtp.tls"
	viperSMTPUser                       = "smtp.user"
	viperSMTPPassword                   = "smtp.password"
	viperSMTPFrom                       = "smtp.from"
	viperArtifactMode                   = "artifact.mode"
	viperArtifactLocalBasedir           = "artifact.local.basedir"
	viperArtifactOSURL                  = "artifact.openstack.url"
	viperArtifactOSUsername             = "artifact.openstack.username"
	viperArtifactOSPassword             = "artifact.openstack.password"
	viperArtifactOSTenant               = "artifact.openstack.tenant"
	viperArtifactOSRegion               = "artifact.openstack.region"
	viperEventsKafkaEnabled             = "events.kafka.enabled"
	viperEventsKafkaBroker              = "events.kafka.broker"
	viperEventsKafkaTopic               = "events.kafka.topic"
	viperEventsKafkaUser                = "events.kafka.user"
	viperEventsKafkaPassword            = "events.kafka.password"
	viperSchedulersDisabled             = "schedulers.disabled"
	viperVCSPollingDisabled             = "vcs.polling.disabled"
	viperVCSRepoCacheLoaderDisabled     = "vcs.repositories.cacherloader.disabled"
	viperVCSRepoGithubStatusDisabled    = "vcs.repositories.github.statuses_disabled"
	viperVCSRepoGithubStatusURLDisabled = "vcs.repositories.github.statuses_url_disabled"
	viperVCSRepoGithubSecret            = "vcs.repositories.github.clientsecret"
	viperVCSRepoBitbucketStatusDisabled = "vcs.repositories.bitbucket.statuses_disabled"
	viperVCSRepoBitbucketPrivateKey     = "vcs.repositories.bitbucket.privatekey"
)

var (
	cfgFile string
)

func init() {
	//Config file
	us, err := user.Current()
	if err != nil {
		sdk.Exit(err.Error())
	}
	mainCmd.PersistentFlags().StringVar(&cfgFile, "config", us.HomeDir+"/.cds/api.config.toml", "config file")

	//Database command
	mainCmd.AddCommand(database.DBCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		//If the config file doesn't exists, let's create it
		if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
			generateConfigTemplate()
		}
		fmt.Println("Reading config file", cfgFile)
		viper.SetConfigFile(cfgFile)
	}

	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvPrefix("cds")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")) // Replace "." and "-" by "_" for env variable lookup

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		sdk.Exit(err.Error())
	}
}

func generateConfigTemplate() {
	fmt.Println("Generating default config file", cfgFile)
	if err := ioutil.WriteFile(cfgFile, []byte(tmpl), os.FileMode(0600)); err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}

const tmpl = `# CDS Configuration file template

# Set the URLs from the user's point of view
[url]
api = "http://localhost:8081"
ui = "http://localhost:8080"

[directories]
download = "/app"
keys = "/app/keys"

# Server part
[server]

    [server.http]
    port = 8081
    sessionTTL = 60

    [server.grpc]
    port = 8082

    [server.secrets]
    key = ""
    # uncomment this two lines to 
    # backend = "secret-backend-vault"
    # backendoptions = []

[log]
level = "notice"
db=false

[db]
user = "cds"
password = "cds"
name = "cds"
host = "localhost"
port = 5432
sslmode = "disable"
maxconn = 20
timeout = 3000
# uncomment this to retreive database credentials from secret-backend
# secret = ""

[cache]
mode = "local"
ttl = 60

    [cache.redis]
    host = "localhost:6379"
    password = ""

[auth]
defaultgroup = ""
localmode = "session"

[auth.ldap]
enable = false
host = "ldap-internal.ovh.net"
port = 636
ssl = true
base = ""
dn = "uid=%s,ou=people,{{.ldap-base}}"
fullname = "{{.givenName}} {{.sn}}"

[smtp]

disable = true
host = ""
port = 23
tls = false
user = ""
password = ""
from = "no-reply@cds.org"

[artifact]
mode = "local"

    [artifact.local]
    basedir = "/tmp/cds"

    [artifact.openstack]
    url = ""
    username = ""
    password = ""
    tenant = ""
    region = ""

[events]
    [events.kafka]
    enabled = false
    broker = ""
    topic = ""
    user = ""
    password = ""

[schedulers]
disabled = true

[vcs]
    [vcs.polling]
    disabled = true

    [vcs.repositories]
    cacheloader_disabled = true

    [vcs.repositories.github]
    statuses_disabled = true
    statuses_url_disabled = true
    clientsecret = ""

    [vcs.repositories.bitbucket]
    statuses_disabled = true
    privatekey = ""
`
