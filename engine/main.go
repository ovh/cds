package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/spf13/viper/remote"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/database"

)

type Configuration struct {
	API      api.Configuration
	Hatchery []struct{}
}

type Service interface {
	Init(cfg interface{}) error
	Serve(ctx context.Context) error
}

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
func config() {
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

		if err := viper.Unmarshal(&Configuration{}) {
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

func main() {
	//Initliaze context
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// Gracefully shutdown all
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	defer func() {
		signal.Stop(c)
		cancel()
	}()
	go func() {
		select {
		case <-c:
			cancel()
			os.Exit(0)
		case <-ctx.Done():
		}
	}()
}

var mainCmd = &cobra.Command{
	Use:   "api",
	Short: "CDS Engine",
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
			LDAPBase:             "{{.ldapBase}}",
			SN:                   "{{.sn}}",
			GivenName:            "{{.givenName}}",
		}
		tmplContent = tmpl
	} else { // Generate config with vault
		s, errS := secret.New(vaultToken, vaultAddr)
		if errS != nil {
			log.Warning("Error when creating vault config")
			os.Exit(1)
		}
		// Get raw config file from vault
		cfgFileContent, errV := s.GetFromVault(vaultConfKey)
		if errV != nil {
			log.Warning("Error when fetch secret %s from vault", vaultConfKey)
			os.Exit(1)
		}
		tmplContent = cfgFileContent

		v = defaultValues{
			AuthSharedInfraToken: token,
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

	fmt.Printf("You can now launch: 'api --config %s' to run CDS API\n", cfgFile)
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
password = "cds"
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
    password = "<OS_PASSWORD>"
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
    password = "<Kafka password>"

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
