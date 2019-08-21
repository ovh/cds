package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fsamin/go-dump"
	defaults "github.com/mcuadros/go-defaults"
	"github.com/ovh/symmecrypt/keyloader"
	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/engine/elasticsearch"
	"github.com/ovh/cds/engine/hatchery/kubernetes"
	"github.com/ovh/cds/engine/hatchery/local"
	"github.com/ovh/cds/engine/hatchery/marathon"
	"github.com/ovh/cds/engine/hatchery/openstack"
	"github.com/ovh/cds/engine/hatchery/swarm"
	"github.com/ovh/cds/engine/hatchery/vsphere"
	"github.com/ovh/cds/engine/hooks"
	"github.com/ovh/cds/engine/migrateservice"
	"github.com/ovh/cds/engine/repositories"
	"github.com/ovh/cds/engine/vcs"
	"github.com/ovh/cds/sdk"
)

const (
	vaultConfKey = "/secret/cds/conf"
)

func configBootstrap(args []string) *Configuration {
	conf := &Configuration{}
	defaults.SetDefaults(&conf.Debug)
	defaults.SetDefaults(&conf.Tracing)

	// Default config if nothing is given
	if len(args) == 0 {
		args = []string{
			"api", "migrate", "hooks", "vcs", "repositories", "elasticsearch",
			"hatchery:local", "hatchery:kubernetes", "hatchery:marathon", "hatchery:openstack", "hatchery:swarm", "hatchery:vsphere",
		}
	}

	for _, a := range args {
		if strings.HasPrefix(a, "hatchery:") {
			if conf.Hatchery == nil {
				conf.Hatchery = &HatcheryConfiguration{}
				break
			}
		}
	}
	for _, a := range args {
		switch a {
		case "api":
			conf.API = &api.Configuration{}
			defaults.SetDefaults(conf.API)

			key, _ := keyloader.GenerateKey("hmac", gorpmapping.KeySignIdentifier, false, time.Now())
			conf.API.Database.SignatureKey = database.RollingKeyConfig{Cipher: "hmac"}
			conf.API.Database.SignatureKey.Keys = append(conf.API.Database.SignatureKey.Keys, database.KeyConfig{
				Key:       key.Key,
				Timestamp: key.Timestamp,
			})

			key, _ = keyloader.GenerateKey("xchacha20-poly1305", gorpmapping.KeyEcnryptionIdentifier, false, time.Now())
			conf.API.Database.EncryptionKey = database.RollingKeyConfig{Cipher: "xchacha20-poly1305"}
			conf.API.Database.EncryptionKey.Keys = append(conf.API.Database.EncryptionKey.Keys, database.KeyConfig{
				Key:       key.Key,
				Timestamp: key.Timestamp,
			})
		case "migrate":
			conf.DatabaseMigrate = &migrateservice.Configuration{}
			defaults.SetDefaults(conf.DatabaseMigrate)
		case "hatchery:local":
			conf.Hatchery.Local = &local.HatcheryConfiguration{}
			defaults.SetDefaults(conf.Hatchery.Local)
		case "hatchery:kubernetes":
			conf.Hatchery.Kubernetes = &kubernetes.HatcheryConfiguration{}
			defaults.SetDefaults(conf.Hatchery.Kubernetes)
		case "hatchery:marathon":
			conf.Hatchery.Marathon = &marathon.HatcheryConfiguration{}
			defaults.SetDefaults(conf.Hatchery.Marathon)
		case "hatchery:openstack":
			conf.Hatchery.Openstack = &openstack.HatcheryConfiguration{}
			defaults.SetDefaults(conf.Hatchery.Openstack)
		case "hatchery:swarm":
			conf.Hatchery.Swarm = &swarm.HatcheryConfiguration{}
			defaults.SetDefaults(conf.Hatchery.Swarm)
		case "hatchery:vsphere":
			conf.Hatchery.VSphere = &vsphere.HatcheryConfiguration{}
			defaults.SetDefaults(conf.Hatchery.VSphere)
		case "hooks":
			conf.Hooks = &hooks.Configuration{}
			defaults.SetDefaults(conf.Hooks)
		case "vcs":
			conf.VCS = &vcs.Configuration{}
			defaults.SetDefaults(conf.VCS)
			var github vcs.GithubServerConfiguration
			defaults.SetDefaults(&github)
			var bitbucket vcs.BitbucketServerConfiguration
			defaults.SetDefaults(&bitbucket)
			var bitbucketcloud vcs.BitbucketCloudConfiguration
			defaults.SetDefaults(&bitbucketcloud)
			var gitlab vcs.GitlabServerConfiguration
			defaults.SetDefaults(&gitlab)
			var gerrit vcs.GerritServerConfiguration
			defaults.SetDefaults(&gerrit)
			conf.VCS.Servers = map[string]vcs.ServerConfiguration{
				"Github":         vcs.ServerConfiguration{URL: "https://github.com", Github: &github},
				"Bitbucket":      vcs.ServerConfiguration{URL: "https://mybitbucket.com", Bitbucket: &bitbucket},
				"bitbucketcloud": vcs.ServerConfiguration{BitbucketCloud: &bitbucketcloud},
				"Gitlab":         vcs.ServerConfiguration{URL: "https://gitlab.com", Gitlab: &gitlab},
				"Gerrit":         vcs.ServerConfiguration{URL: "http://localhost:8080", Gerrit: &gerrit},
			}
		case "repositories":
			conf.Repositories = &repositories.Configuration{}
			defaults.SetDefaults(conf.Repositories)
		case "elasticsearch":
			conf.ElasticSearch = &elasticsearch.Configuration{}
			defaults.SetDefaults(conf.ElasticSearch)
		default:
			sdk.Exit("Error service '%s' is unknown", a)
		}
	}

	return conf
}

// asEnvVariables returns the object attributes as env variables.
func configToEnvVariables(o interface{}) map[string]string {
	dumper := dump.NewDefaultEncoder()
	dumper.DisableTypePrefix = true
	dumper.Separator = "_"
	dumper.Prefix = "CDS"
	dumper.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultUpperCaseFormatter()}
	envs, _ := dumper.ToStringMap(o)
	for key := range envs {
		_ = viper.BindEnv(dumper.ViperKey(key), key)
	}
	return envs
}

// Generates a config
func configImport(args []string, cfgFile, remoteCfg, remoteCfgKey, vaultAddr, vaultToken string) Configuration {
	// Generate a default bootstraped config for given args to get ENV variables keys.
	defaultConfig := configBootstrap(args)

	// Convert the default generated config to envs to setup binding in viper.
	_ = configToEnvVariables(defaultConfig)

	switch {
	case remoteCfg != "":
		fmt.Println("Reading configuration from consul @", remoteCfg)

		viper.AddRemoteProvider("consul", remoteCfg, remoteCfgKey)
		viper.SetConfigType("toml")
		if err := viper.ReadRemoteConfig(); err != nil {
			sdk.Exit(err.Error())
		}
	case vaultAddr != "" && vaultToken != "":
		// I hope one day vault will be a standard viper remote provider
		fmt.Println("Reading configuration from vault @", vaultAddr)

		s, err := secret.New(vaultToken, vaultAddr)
		if err != nil {
			sdk.Exit("Error when getting config from vault: %v", err)
		}

		// Get raw config file from vault
		cfgFileContent, err := s.GetFromVault(vaultConfKey)
		if err != nil {
			sdk.Exit("Error when fetching config from vault: %v", err)
		}

		// Put the content in a buffer and ask viper to read the buffer
		viper.SetConfigType("toml")
		if err := viper.ReadConfig(bytes.NewBufferString(cfgFileContent)); err != nil {
			sdk.Exit("Unable to read config: %v", err)
		}
	case cfgFile != "":
		fmt.Println("Reading configuration file @", cfgFile)

		// If the config file doesn't exists, let's exit
		if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
			sdk.Exit("Error file %s doesn't exist", cfgFile)
		}

		viper.SetConfigFile(cfgFile)
		viper.SetConfigType("toml")
		if err := viper.ReadInConfig(); err != nil {
			sdk.Exit(err.Error())
		}
	}

	var conf Configuration
	if err := viper.Unmarshal(&conf); err != nil {
		sdk.Exit("Unable to parse config: %v", err.Error())
	}
	return conf
}
