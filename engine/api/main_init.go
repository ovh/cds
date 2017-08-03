package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const (
	viperURLAPI                         = "url.api"
	viperURLUI                          = "url.ui"
	viperServerHTTPPort                 = "server.http.port"
	viperServerSessionTTL               = "server.http.sessionTTL"
	viperServerGRPCPort                 = "server.grpc.port"
	viperServerSecretKey                = "server.secrets.key"
	viperLogLevel                       = "log.level"
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
	viperAuthLDAPEnable                 = "auth.ldap.enable"
	viperAuthLDAPHost                   = "auth.ldap.host"
	viperAuthLDAPPort                   = "auth.ldap.port"
	viperAuthLDAPSSL                    = "auth.ldap.ssl"
	viperAuthLDAPBase                   = "auth.ldap.base"
	viperAuthLDAPDN                     = "auth.ldap.dn"
	viperAuthLDAPFullname               = "auth.ldap.fullname"
	viperAuthDefaultGroup               = "auth.defaultgroup"
	viperAuthSharedInfraToken           = "auth.sharedinfra.token"
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
	viperArtifactOSContainerPrefix      = "artifact.openstack.containerprefix"
	viperEventsKafkaEnabled             = "events.kafka.enabled"
	viperEventsKafkaBroker              = "events.kafka.broker"
	viperEventsKafkaTopic               = "events.kafka.topic"
	viperEventsKafkaUser                = "events.kafka.user"
	viperEventsKafkaPassword            = "events.kafka.password"
	viperSchedulersDisabled             = "schedulers.disabled"
	viperVCSPollingDisabled             = "vcs.polling.disabled"
	viperVCSRepoGithubStatusDisabled    = "vcs.repositories.github.statuses_disabled"
	viperVCSRepoGithubStatusURLDisabled = "vcs.repositories.github.statuses_url_disabled"
	viperVCSRepoGithubSecret            = "vcs.repositories.github.clientsecret"
	viperVCSRepoBitbucketStatusDisabled = "vcs.repositories.bitbucket.statuses_disabled"
	viperVCSRepoBitbucketConsumerKey    = "vcs.repositories.bitbucket.consumerkey"
	viperVCSRepoBitbucketPrivateKey     = "vcs.repositories.bitbucket.privatekey"
	vaultConfKey                        = "/secret/cds/conf"
	vaultDBKey                          = "/secret/cds/db-password"
)

var (
	cfgFile      string
	remoteCfg    string
	remoteCfgKey string
	vaultAddr    string
	vaultToken   string
)

func init() {
	mainCmd.Flags().StringVar(&cfgFile, "config", "", "config file")
	mainCmd.Flags().StringVar(&remoteCfg, "remote-config", "", "(optional) consul configuration store")
	mainCmd.Flags().StringVar(&remoteCfgKey, "remote-config-key", "cds/config.api.toml", "(optional) consul configuration store key")
	mainCmd.Flags().StringVar(&vaultAddr, "vault-addr", "", "(optional) Vault address to fetch secrets from vault (example: https://vault.mydomain.net:8200)")
	mainCmd.Flags().StringVar(&vaultToken, "vault-token", "", "(optional) Vault token to fetch secrets from vault")
	//Database command
	mainCmd.AddCommand(database.DBCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		//If the config file doesn't exists, let's create it
		if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
			generateConfigTemplate()
		}
		fmt.Println("Reading configuration file", cfgFile)

		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err != nil {
			sdk.Exit(err.Error())
		}

	} else if remoteCfg != "" {
		fmt.Println("Reading configuration from consul @", remoteCfg)
		viper.AddRemoteProvider("consul", remoteCfg, remoteCfgKey)
		viper.SetConfigType("toml")

		if err := viper.ReadRemoteConfig(); err != nil {
			sdk.Exit(err.Error())
		}
	}

	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvPrefix("cds")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")) // Replace "." and "-" by "_" for env variable lookup
}

type defaultValues struct {
	ServerSecretsKey     string
	AuthSharedInfraToken string
	DBPassword           string
	// For LDAP Client
	LDAPBase  string
	GivenName string
	SN        string
}

func generateConfigTemplate() {
	var v defaultValues
	var tmplContent string

	token, err := token.GenerateToken()
	if err != nil {
		log.Error("generateConfigTemplate> cannot generate token")
		os.Exit(1)
	}

	// Generate config with local
	if vaultAddr == "" {
		v = defaultValues{
			ServerSecretsKey:     sdk.RandomString(32),
			AuthSharedInfraToken: token,
			DBPassword:           "cds",
			LDAPBase:             "{{.ldapBase}}",
			SN:                   "{{.sn}}",
			GivenName:            "{{.givenName}}",
		}
		tmplContent = tmpl
	} else { // Generate config with vault
		s, err := secret.New(vaultToken, vaultAddr)
		if err != nil {
			log.Warning("Error when creating vault config")
			os.Exit(1)
		}
		// Get raw config file from vault
		cfgFileContent, err := s.GetFromVault(vaultConfKey)
		if err != nil {
			log.Warning("Error when fetch secret %s from vault", vaultConfKey)
			os.Exit(1)
		}
		tmplContent = cfgFileContent

		// Get database password from vault
		dbPassword, err := s.GetFromVault(vaultDBKey)
		if err != nil {
			log.Warning("Error when fetch secret %s from vault", vaultDBKey)
			os.Exit(1)
		}

		v = defaultValues{
			AuthSharedInfraToken: token,
			DBPassword:           dbPassword,
			LDAPBase:             "{{.ldapBase}}",
			SN:                   "{{.sn}}",
			GivenName:            "{{.givenName}}",
		}
	}

	tmplI, err := template.New("test").Parse(tmplContent)
	if err != nil {
		fmt.Println("Error new: ", err)
		os.Exit(1)
	}
	var tpl bytes.Buffer
	if err := tmplI.Execute(&tpl, v); err != nil {
		fmt.Println("Error execute: ", err)
		os.Exit(1)
	}

	fmt.Println("Generating default config file", cfgFile)
	if err := ioutil.WriteFile(cfgFile, tpl.Bytes(), os.FileMode(0600)); err != nil {
		fmt.Println("Error write file: ", err)
		os.Exit(1)
	}
	os.Exit(0)
}

const tmpl = `###################################
# CDS Configuration file template #
###################################
# Please update this file with your own settings
#
# Note that you can override the configuration file with environments variables
# CDS_URL_API
# CDS_URL_UI
# CDS_SERVER_HTTP_PORT
# CDS_SERVER_HTTP_SESSIONTTL
# CDS_SERVER_GRPC_PORT
# CDS_SERVER_SECRETS_KEY
# CDS_LOG_LEVEL
# CDS_DB_USER
# CDS_DB_PASSWORD
# CDS_DB_NAME
# CDS_DB_HOST
# CDS_DB_PORT
# CDS_DB_SSLMODE
# CDS_DB_MAXCONN
# CDS_DB_TIMEOUT
# CDS_DB_SECRET
# CDS_CACHE_MODE
# CDS_CACHE_TTL
# CDS_CACHE_REDIS_HOST
# CDS_CACHE_REDIS_PASSWORD
# CDS_DIRECTORIES_DOWNLOAD
# CDS_DIRECTORIES_KEYS
# CDS_AUTH_LDAP_ENABLE
# CDS_AUTH_LDAP_HOST
# CDS_AUTH_LDAP_PORT
# CDS_AUTH_LDAP_SSL
# CDS_AUTH_LDAP_BASE
# CDS_AUTH_LDAP_DN
# CDS_AUTH_LDAP_FULLNAME
# CDS_AUTH_DEFAULTGROUP
# CDS_AUTH_SHAREDINFRA_TOKEN
# CDS_SMTP_DISABLE
# CDS_SMTP_HOST
# CDS_SMTP_PORT
# CDS_SMTP_TLS
# CDS_SMTP_USER
# CDS_SMTP_PASSWORD
# CDS_SMTP_FROM
# CDS_ARTIFACT_MODE
# CDS_ARTIFACT_LOCAL_BASEDIR
# CDS_ARTIFACT_OPENSTACK_URL
# CDS_ARTIFACT_OPENSTACK_USERNAME
# CDS_ARTIFACT_OPENSTACK_PASSWORD
# CDS_ARTIFACT_OPENSTACK_TENANT
# CDS_ARTIFACT_OPENSTACK_REGION
# CDS_ARTIFACT_OPENSTACK_CONTAINERPREFIX
# CDS_EVENTS_KAFKA_ENABLED
# CDS_EVENTS_KAFKA_BROKER
# CDS_EVENTS_KAFKA_TOPIC
# CDS_EVENTS_KAFKA_USER
# CDS_EVENTS_KAFKA_PASSWORD
# CDS_SCHEDULERS_DISABLED
# CDS_VCS_POLLING_DISABLED
# CDS_VCS_REPOSITORIES_GITHUB_STATUSES_DISABLED
# CDS_VCS_REPOSITORIES_GITHUB_STATUSES_URL_DISABLED
# CDS_VCS_REPOSITORIES_GITHUB_CLIENTSECRET
# CDS_VCS_REPOSITORIES_BITBUCKET_STATUSES_DISABLED
# CDS_VCS_REPOSITORIES_BITBUCKET_CONSUMERKEY
# CDS_VCS_REPOSITORIES_BITBUCKET_PRIVATEKEY


#####################
# CDS URLs Settings #
#####################
# Set the URLs from the user's point of view. It may be URL of your reverse proxy if you use one.
[url]
api = "http://localhost:8081"
ui = "http://localhost:8080"

#####################
# CDS Logs Settings #
#####################
# Define log levels and hooks
[log]
# debug, info, warning or error
level = "info"

# CDS needs local directories to store temporary data (keys) and serve cds binaries such as hatcheries and workers (download)
[directories]
download = "/app"
keys = "/app/keys"

###########################
# General server settings #
###########################
[server]
    [server.http]
    port = 8081
    sessionTTL = 60

    [server.grpc]
    port = 8082

    [server.secrets]
		# AES Cypher key for database encryption. 32 char.
		# This is mandatory
    key = "{{.ServerSecretsKey}}"


################################
# Postgresql Database settings #
################################
[db]
user = "cds"
password = "{{.DBPassword}}" # set /secret/cds/db-password in vault if you use it
name = "cds"
host = "localhost"
port = 5432
# DB SSL Mode: require, verify-full, or disable
sslmode = "disable"
maxconn = 20
timeout = 3000


######################
# CDS Cache Settings #
######################
# If your CDS is made of a unique instance, a local cache if enough, but rememeber that all cached data will be lost on startup.
[cache]
#Uncomment to use redis as cache
#mode = "redis"
mode = "local"
ttl = 60

    # Connect CDS to a redis cache If you more than one CDS instance and to avoid losing data at startup
    [cache.redis]
    host = "localhost:6379" # If your want to use a redis-sentinel based cluster, follow this syntax ! <clustername>@sentinel1:26379,sentinel2:26379sentinel3:26379
    password = ""

##############################
# CDS Authentication Settings#
##############################
[auth]
# The default group is the group in which every new user will be granted at signup
defaultgroup = ""

	[auth.sharedinfra]
	# Token for shared.infra group. This value will be used when shared.infra will be created
	# at first CDS launch. This token can be used by CDS CLI, Hatchery, etc...
	# This is mandatory. 64 char
	token = "{{.AuthSharedInfraToken}}"

	[auth.ldap]
	enable = false
	host = "<LDAP-server>"
	port = 636
	ssl = true
	# LDAP Base
	base = ""
	# LDAP Bind DN
	dn = "uid=%s,ou=people,{{.LDAPBase}}"
	# Define CDS user fullname from LDAP attribute
	fullname = "{{.GivenName}} {{.SN}}"

#####################
# CDS SMTP Settings #
#####################
[smtp]
disable = true
host = ""
port = 23
tls = false
user = ""
password = ""
from = "no-reply@cds.org"

##########################
# CDS Artifacts Settings #
##########################
# Either filesystem local storage or Openstack Swift Storage are supported
[artifact]
# mode = "swift#
mode = "local"

    [artifact.local]
    basedir = "/tmp/cds"

    [artifact.openstack]
    url = "<OS_AUTH_URL>"
    username = "<OS_USERNAME>"
    password = ""
    tenant = "<OS_TENANT_NAME>"
    region = "<OS_REGION_NAME>"
    containerprefix = "" # Use if your want to prefix containers

#######################
# CDS Events Settings #
#######################
#For now, only Kafka is supported as a event broker
[events]
    [events.kafka]
    enabled = false
    broker = "<Kafka SASK/SSL addresses>"
    topic = "<Kafka topic>"
    user = "<Kafka username>"
    password = ""

###########################
# CDS Schedulers Settings #
###########################
[schedulers]
disabled = false #This is mainly for dev purpose, you should not have to change it

####################
# CDS VCS Settings #
####################
[vcs]
    [vcs.polling]
    disabled = false #This is mainly for dev purpose, you should not have to change it

    [vcs.repositories]

    [vcs.repositories.github]
    statuses_disabled = false # Set to true if you don't want CDS to push statuses on Github API
    statuses_url_disabled = false # Set to true if you don't want CDS to push CDS URL in statuses on Github API
    clientsecret = ""

    [vcs.repositories.bitbucket]
    statuses_disabled = false
    privatekey = ""
`
