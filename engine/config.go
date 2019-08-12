package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/symmecrypt/keyloader"

	"github.com/fsamin/go-dump"
	defaults "github.com/mcuadros/go-defaults"
	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/observability"
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

func configSetDefaults() {
	if conf.Debug != nil {
		defaults.SetDefaults(conf.Debug)
	}
	if conf.Tracing != nil {
		defaults.SetDefaults(conf.Tracing)
	}
	if conf.API != nil {
		defaults.SetDefaults(conf.API)
	}
	if conf.DatabaseMigrate != nil {
		defaults.SetDefaults(conf.DatabaseMigrate)
	}
	if conf.Hatchery != nil && conf.Hatchery.Local != nil {
		defaults.SetDefaults(conf.Hatchery.Local)
	}
	if conf.Hatchery != nil && conf.Hatchery.Kubernetes != nil {
		defaults.SetDefaults(conf.Hatchery.Kubernetes)
	}
	if conf.Hatchery != nil && conf.Hatchery.Marathon != nil {
		defaults.SetDefaults(conf.Hatchery.Marathon)
	}
	if conf.Hatchery != nil && conf.Hatchery.Openstack != nil {
		defaults.SetDefaults(conf.Hatchery.Openstack)
	}
	if conf.Hatchery != nil && conf.Hatchery.Swarm != nil {
		defaults.SetDefaults(conf.Hatchery.Swarm)
	}
	if conf.Hatchery != nil && conf.Hatchery.VSphere != nil {
		defaults.SetDefaults(conf.Hatchery.VSphere)
	}
	if conf.Hooks != nil {
		defaults.SetDefaults(conf.Hooks)
	}
	if conf.VCS != nil {
		defaults.SetDefaults(conf.VCS)
	}
	if conf.VCS != nil && conf.VCS.Servers == nil {
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
	}
	if conf.Repositories != nil {
		defaults.SetDefaults(conf.Repositories)
	}
	if conf.ElasticSearch != nil {
		defaults.SetDefaults(conf.ElasticSearch)
	}
}

// config reads in config file and ENV variables if set.
func configBootstrap(args []string) {
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
			if conf.API == nil {
				conf.API = &api.Configuration{}
				key, _ := keyloader.GenerateKey("hmac", gorpmapping.KeySignIdentifier, false, time.Now())
				conf.API.Database.SignatureKey = database.RollingKeyConfig{
					Cipher: "hmac",
				}
				conf.API.Database.SignatureKey.Keys = append(conf.API.Database.SignatureKey.Keys, database.KeyConfig{
					Key:       key.Key,
					Timestamp: key.Timestamp,
				})

				key, _ = keyloader.GenerateKey("xchacha20-poly1305", gorpmapping.KeyEcnryptionIdentifier, false, time.Now())
				conf.API.Database.EncryptionKey = database.RollingKeyConfig{
					Cipher: "xchacha20-poly1305",
				}
				conf.API.Database.EncryptionKey.Keys = append(conf.API.Database.EncryptionKey.Keys,
					database.KeyConfig{
						Key:       key.Key,
						Timestamp: key.Timestamp,
					})
			}
		case "migrate":
			if conf.DatabaseMigrate == nil {
				conf.DatabaseMigrate = &migrateservice.Configuration{}
			}
		case "hatchery:local":
			if conf.Hatchery.Local == nil {
				conf.Hatchery.Local = &local.HatcheryConfiguration{}
			}
		case "hatchery:kubernetes":
			if conf.Hatchery.Kubernetes == nil {
				conf.Hatchery.Kubernetes = &kubernetes.HatcheryConfiguration{}
			}
		case "hatchery:marathon":
			if conf.Hatchery.Marathon == nil {
				conf.Hatchery.Marathon = &marathon.HatcheryConfiguration{}
			}
		case "hatchery:openstack":
			if conf.Hatchery.Openstack == nil {
				conf.Hatchery.Openstack = &openstack.HatcheryConfiguration{}
			}
		case "hatchery:swarm":
			if conf.Hatchery.Swarm == nil {
				conf.Hatchery.Swarm = &swarm.HatcheryConfiguration{}
			}
		case "hatchery:vsphere":
			if conf.Hatchery.VSphere == nil {
				conf.Hatchery.VSphere = &vsphere.HatcheryConfiguration{}
			}
		case "hooks":
			if conf.Hooks == nil {
				conf.Hooks = &hooks.Configuration{}
			}
		case "vcs":
			if conf.VCS == nil {
				conf.VCS = &vcs.Configuration{}
			}
		case "repositories":
			if conf.Repositories == nil {
				conf.Repositories = &repositories.Configuration{}
			}
		case "elasticsearch":
			if conf.ElasticSearch == nil {
				conf.ElasticSearch = &elasticsearch.Configuration{}
			}
		default:
			fmt.Printf("Error: service '%s' unknown\n", a)
			os.Exit(1)
		}
	}

	if len(args) == 0 {
		conf.Debug = &DebugConfiguration{}
		conf.Tracing = &observability.Configuration{}
		conf.API = &api.Configuration{}
		key, _ := keyloader.GenerateKey("hmac", gorpmapping.KeySignIdentifier, false, time.Now())
		conf.API.Database.SignatureKey = database.RollingKeyConfig{
			Cipher: "hmac",
		}
		conf.API.Database.SignatureKey.Keys = append(conf.API.Database.SignatureKey.Keys,
			database.KeyConfig{
				Key:       key.Key,
				Timestamp: key.Timestamp,
			})
		key, _ = keyloader.GenerateKey("xchacha20-poly1305", gorpmapping.KeyEcnryptionIdentifier, false, time.Now())
		conf.API.Database.EncryptionKey = database.RollingKeyConfig{
			Cipher: "xchacha20-poly1305",
		}
		conf.API.Database.EncryptionKey.Keys = append(conf.API.Database.EncryptionKey.Keys,
			database.KeyConfig{
				Key:       key.Key,
				Timestamp: key.Timestamp,
			})
		conf.DatabaseMigrate = &migrateservice.Configuration{}
		conf.Hatchery = &HatcheryConfiguration{}
		conf.Hatchery.Local = &local.HatcheryConfiguration{}
		conf.Hatchery.Kubernetes = &kubernetes.HatcheryConfiguration{}
		conf.Hatchery.Marathon = &marathon.HatcheryConfiguration{}
		conf.Hatchery.Openstack = &openstack.HatcheryConfiguration{}
		conf.Hatchery.Swarm = &swarm.HatcheryConfiguration{}
		conf.Hatchery.VSphere = &vsphere.HatcheryConfiguration{}
		conf.Hooks = &hooks.Configuration{}
		conf.VCS = &vcs.Configuration{}
		conf.Repositories = &repositories.Configuration{}
		conf.ElasticSearch = &elasticsearch.Configuration{}
	}
}

// asEnvVariables returns the object attributes as env variables. It used for configuration structs
func asEnvVariables(o interface{}) map[string]string {
	dumper := dump.NewDefaultEncoder()
	dumper.DisableTypePrefix = true
	dumper.Separator = "_"
	dumper.Prefix = "CDS"
	dumper.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultUpperCaseFormatter()}
	envs, _ := dumper.ToStringMap(o)
	for k := range envs {
		viper.BindEnv(dumper.ViperKey(k), k) // nolint
	}
	return envs
}

func config(args []string) {
	if conf.Debug == nil {
		conf.Debug = &DebugConfiguration{}
	}

	if conf.Tracing == nil {
		conf.Tracing = &observability.Configuration{}
	}

	asEnvVariables(conf)

	var setDefault bool
	switch {
	case remoteCfg != "":
		fmt.Println("Reading configuration from consul @", remoteCfg)
		viper.AddRemoteProvider("consul", remoteCfg, remoteCfgKey)
		viper.SetConfigType("toml")

		if err := viper.ReadRemoteConfig(); err != nil {
			sdk.Exit(err.Error())
		}
	case vaultAddr != "" && vaultToken != "":
		//I hope one day viper will be a standard viper remote provider
		fmt.Println("Reading configuration from vault @", vaultAddr)

		s, errS := secret.New(vaultToken, vaultAddr)
		if errS != nil {
			sdk.Exit("Error when getting config from vault: %v", errS)
		}
		// Get raw config file from vault
		cfgFileContent, errV := s.GetFromVault(vaultConfKey)
		if errV != nil {
			sdk.Exit("Error when fetching config from vault: %v", errV)
		}

		// Put the content in a buffer and ask viper to read the buffer
		cfgBuffer := bytes.NewBufferString(cfgFileContent)
		viper.SetConfigType("toml")
		if err := viper.ReadConfig(cfgBuffer); err != nil {
			sdk.Exit("Unable to read config: %v", err.Error())
		}
	case cfgFile != "":
		//If the config file doesn't exists, let's exit
		if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
			sdk.Exit("File %s doesn't exist", cfgFile)
		}
		fmt.Println("Reading configuration file", cfgFile)

		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err != nil {
			sdk.Exit(err.Error())
		}
	default:
		setDefault = true
	}

	if err := viper.Unmarshal(conf); err != nil {
		sdk.Exit("Unable to parse config: %v", err.Error())
	}

	if setDefault {
		configSetDefaults()
	}
}
