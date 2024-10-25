package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fsamin/go-dump"
	defaults "github.com/mcuadros/go-defaults"
	"github.com/ovh/symmecrypt"
	"github.com/ovh/symmecrypt/ciphers/aesgcm"
	"github.com/ovh/symmecrypt/convergent"
	"github.com/ovh/symmecrypt/keyloader"
	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/driver/builtin"
	"github.com/ovh/cds/engine/cdn"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/database"
	"github.com/ovh/cds/engine/elasticsearch"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/hatchery/kubernetes"
	"github.com/ovh/cds/engine/hatchery/local"
	"github.com/ovh/cds/engine/hatchery/openstack"
	"github.com/ovh/cds/engine/hatchery/swarm"
	"github.com/ovh/cds/engine/hatchery/vsphere"
	"github.com/ovh/cds/engine/hooks"
	"github.com/ovh/cds/engine/migrateservice"
	"github.com/ovh/cds/engine/repositories"
	"github.com/ovh/cds/engine/ui"
	"github.com/ovh/cds/engine/vcs"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/namesgenerator"
)

var (
	vaultConfKey = "/secret/cds/conf"
)

func configBootstrap(args []string) Configuration {
	var conf Configuration
	defaults.SetDefaults(&conf.Telemetry)

	// Default config if nothing is given
	if len(args) == 0 {
		args = []string{
			"api", "ui", "migrate", "hooks", "vcs", "repositories", "elasticsearch", "cdn",
			"hatchery:local", "hatchery:kubernetes", "hatchery:openstack", "hatchery:swarm", "hatchery:vsphere",
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
		case sdk.TypeAPI:
			conf.API = &api.Configuration{}
			conf.API.Name = "cds-api-" + namesgenerator.GetRandomNameCDS()
			defaults.SetDefaults(conf.API)
			conf.API.Database.Schema = "public"
			conf.API.HTTP.Port = 8081
			conf.API.Auth.AllowedOrganizations = []string{"default"}
			conf.API.Workflow.CustomServiceJobBookDelay = map[string]int64{
				"my-service": 120,
			}
		case sdk.TypeUI:
			conf.UI = &ui.Configuration{}
			conf.UI.Name = "cds-ui-" + namesgenerator.GetRandomNameCDS()
			defaults.SetDefaults(conf.UI)
			conf.UI.HTTP.Port = 8080
		case "migrate":
			conf.DatabaseMigrate = &migrateservice.Configuration{}
			defaults.SetDefaults(conf.DatabaseMigrate)
			conf.DatabaseMigrate.Name = "cds-migrate-" + namesgenerator.GetRandomNameCDS()
			conf.DatabaseMigrate.ServiceAPI.DB.Schema = "public"
			conf.DatabaseMigrate.ServiceCDN.DB.Schema = "cdn"
			conf.DatabaseMigrate.HTTP.Port = 8087
		case sdk.TypeHatchery + ":local":
			conf.Hatchery.Local = &local.HatcheryConfiguration{}
			defaults.SetDefaults(conf.Hatchery.Local)
			conf.Hatchery.Local.Name = "cds-hatchery-local-" + namesgenerator.GetRandomNameCDS()
			conf.Hatchery.Local.HTTP.Port = 8086
		case sdk.TypeHatchery + ":kubernetes":
			conf.Hatchery.Kubernetes = &kubernetes.HatcheryConfiguration{}
			defaults.SetDefaults(conf.Hatchery.Kubernetes)
			conf.Hatchery.Kubernetes.Name = "cds-hatchery-kubernetes-" + namesgenerator.GetRandomNameCDS()
			conf.Hatchery.Kubernetes.HTTP.Port = 8086
		case sdk.TypeHatchery + ":openstack":
			conf.Hatchery.Openstack = &openstack.HatcheryConfiguration{}
			defaults.SetDefaults(conf.Hatchery.Openstack)
			conf.Hatchery.Openstack.Name = "cds-hatchery-openstack-" + namesgenerator.GetRandomNameCDS()
			conf.Hatchery.Openstack.HTTP.Port = 8086
		case sdk.TypeHatchery + ":swarm":
			conf.Hatchery.Swarm = &swarm.HatcheryConfiguration{}
			defaults.SetDefaults(conf.Hatchery.Swarm)
			host := swarm.DockerEngineConfiguration{
				Host: "unix:///var/run/docker.sock",
			}
			defaults.SetDefaults(&host)
			conf.Hatchery.Swarm.DockerEngines = map[string]swarm.DockerEngineConfiguration{
				"default": host,
			}
			conf.Hatchery.Swarm.Name = "cds-hatchery-swarm-" + namesgenerator.GetRandomNameCDS()
			conf.Hatchery.Swarm.HTTP.Port = 8086
			conf.Hatchery.Swarm.RegistryCredentials = []swarm.RegistryCredential{{
				Domain: "docker.io",
			}}
		case sdk.TypeHatchery + ":vsphere":
			conf.Hatchery.VSphere = &vsphere.HatcheryConfiguration{}
			defaults.SetDefaults(conf.Hatchery.VSphere)
			conf.Hatchery.VSphere.Name = "cds-hatchery-vsphere-" + namesgenerator.GetRandomNameCDS()
			conf.Hatchery.VSphere.HTTP.Port = 8086
			conf.Hatchery.VSphere.WorkerProvisioning = []vsphere.WorkerProvisioningConfig{{
				ModelPath: "my/model",
			}}
			conf.Hatchery.VSphere.GuestCredentials = []vsphere.GuestCredential{{
				ModelPath: "my/model",
			}}
		case sdk.TypeHooks:
			conf.Hooks = &hooks.Configuration{}
			defaults.SetDefaults(conf.Hooks)
			conf.Hooks.Name = "cds-hooks-" + namesgenerator.GetRandomNameCDS()
			conf.Hooks.HTTP.Port = 8083
		case sdk.TypeRepositories:
			conf.Repositories = &repositories.Configuration{}
			defaults.SetDefaults(conf.Repositories)
			conf.Repositories.Name = "cds-repositories-" + namesgenerator.GetRandomNameCDS()
			conf.Repositories.Basedir = "/var/lib/cds-engine/repositories"
			conf.Repositories.HTTP.Port = 8085
		case sdk.TypeVCS:
			conf.VCS = &vcs.Configuration{}
			defaults.SetDefaults(conf.VCS)
			conf.VCS.Name = "cds-vcs-" + namesgenerator.GetRandomNameCDS()
			conf.VCS.HTTP.Port = 8084
		case sdk.TypeCDN:
			conf.CDN = &cdn.Configuration{}
			defaults.SetDefaults(conf.CDN)
			conf.CDN.HTTP.Port = 8089
			conf.CDN.Database.Schema = "cdn"
			conf.CDN.Units.HashLocatorSalt = sdk.RandomString(8)

			kc := keyloader.KeyConfig{
				Identifier: "cdn-buffer-local",
				Timestamp:  time.Now().Unix(),
				Cipher:     aesgcm.CipherName,
				Sealed:     false,
			}

			key, err := symmecrypt.NewRandomKey(kc.Cipher)
			if err != nil {
				sdk.Exit("Error generate cdn buffer key'%v'", err)
			}

			keyStr, err := key.String()
			if err != nil {
				sdk.Exit("Error getting cdn buffer key: '%v'", err)
			}

			kc.Key = keyStr

			conf.CDN.Units.Buffers = map[string]storage.BufferConfiguration{
				"redis": {
					BufferType: storage.CDNBufferTypeLog,
					Redis: &sdk.RedisConf{
						Host: "localhost:6379",
					},
				},
				"local-buffer": {
					BufferType: storage.CDNBufferTypeFile,
					Local: &storage.LocalBufferConfiguration{
						Path:       "/var/lib/cds-engine/cdn-buffer",
						Encryption: []*keyloader.KeyConfig{&kc},
					},
				},
			}
			conf.CDN.Units.Storages = map[string]storage.StorageConfiguration{
				"local": {
					SyncBandwidth: 128,
					SyncParallel:  2,
					Local: &storage.LocalStorageConfiguration{
						Path: "/var/lib/cds-engine/cdn",
						Encryption: []convergent.ConvergentEncryptionConfig{{
							Cipher:      "aes-gcm",
							Identifier:  "cdn-storage-local",
							LocatorSalt: sdk.RandomString(9),
							SecretValue: sdk.RandomString(17),
							Timestamp:   time.Now().Unix(),
						}},
					},
				},
			}
		case sdk.TypeElasticsearch:
			conf.ElasticSearch = &elasticsearch.Configuration{}
			defaults.SetDefaults(conf.ElasticSearch)
			conf.ElasticSearch.HTTP.Port = 8088
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

func configPrintToEnv(c Configuration, w io.Writer) {
	m := configToEnvVariables(c)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		// Print the export command and escape all \n in value (useful for keys)
		fmt.Fprintf(w, "export %s=\"%s\"\n", k, strings.ReplaceAll(m[k], "\n", "\\n"))
	}
}

// Generates a config
func configImport(args []string, cfgFile, remoteCfg, remoteCfgKey, vaultAddr, vaultToken string, silent bool) Configuration {
	// Generate a default bootstraped config for given args to get ENV variables keys.
	defaultConfig := configBootstrap(args)

	// Convert the default generated config to envs to setup binding in viper.
	_ = configToEnvVariables(defaultConfig)

	switch {
	case remoteCfg != "":
		if !silent {
			fmt.Println("Reading configuration from consul @", remoteCfg)
		}

		viper.AddRemoteProvider("consul", remoteCfg, remoteCfgKey)
		viper.SetConfigType("toml")
		if err := viper.ReadRemoteConfig(); err != nil {
			sdk.Exit(err.Error())
		}
	case vaultAddr != "" && vaultToken != "":
		// I hope one day vault will be a standard viper remote provider
		if !silent {
			fmt.Println("Reading configuration from vault @", vaultAddr)
		}

		s, err := VaultNewSecret(vaultToken, vaultAddr)
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
		if !silent {
			fmt.Println("Reading configuration file @", cfgFile)
		}

		// If the config file doesn't exists, let's exit
		if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
			sdk.Exit("Error file %s doesn't exist", cfgFile)
		}

		viper.SetConfigFile(cfgFile)
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

func configSetStartupData(conf *Configuration) (string, error) {
	apiPrivateKey, err := jws.NewRandomRSAKey()
	if err != nil {
		return "", err
	}

	apiPrivateKeyPEM, err := jws.ExportPrivateKey(apiPrivateKey)
	if err != nil {
		return "", err
	}

	validityPediod := sdk.NewAuthConsumerValidityPeriod(time.Now(), 0)
	startupCfg := api.StartupConfig{IAT: validityPediod.Latest().IssuedAt.Unix()}

	if err := authentication.Init(context.TODO(), "cds-api", []authentication.KeyConfig{{Key: string(apiPrivateKeyPEM)}}); err != nil {
		return "", err
	}

	if conf.API != nil {
		conf.API.Auth.RSAPrivateKeys = []authentication.KeyConfig{
			{
				Timestamp: time.Now().Unix(),
				Key:       string(apiPrivateKeyPEM),
			},
		}

		key, _ := keyloader.GenerateKey("hmac", gorpmapper.KeySignIdentifier, false, time.Now())
		conf.API.Database.SignatureKey = &database.RollingKeyConfig{Cipher: "hmac"}
		conf.API.Database.SignatureKey.Keys = append(conf.API.Database.SignatureKey.Keys, database.KeyConfig{
			Key:       key.Key,
			Timestamp: key.Timestamp,
		})

		key, _ = keyloader.GenerateKey("xchacha20-poly1305", gorpmapper.KeyEcnryptionIdentifier, false, time.Now())
		conf.API.Database.EncryptionKey = &database.RollingKeyConfig{Cipher: "xchacha20-poly1305"}
		conf.API.Database.EncryptionKey.Keys = append(conf.API.Database.EncryptionKey.Keys, database.KeyConfig{
			Key:       key.Key,
			Timestamp: key.Timestamp,
		})
	}

	if conf.CDN != nil {
		key, _ := keyloader.GenerateKey("hmac", gorpmapper.KeySignIdentifier, false, time.Now())
		conf.CDN.Database.SignatureKey = &database.RollingKeyConfig{Cipher: "hmac"}
		conf.CDN.Database.SignatureKey.Keys = append(conf.CDN.Database.SignatureKey.Keys, database.KeyConfig{
			Key:       key.Key,
			Timestamp: key.Timestamp,
		})

		key, _ = keyloader.GenerateKey("xchacha20-poly1305", gorpmapper.KeyEcnryptionIdentifier, false, time.Now())
		conf.CDN.Database.EncryptionKey = &database.RollingKeyConfig{Cipher: "xchacha20-poly1305"}
		conf.CDN.Database.EncryptionKey.Keys = append(conf.CDN.Database.EncryptionKey.Keys, database.KeyConfig{
			Key:       key.Key,
			Timestamp: key.Timestamp,
		})
	}

	if conf.UI != nil {
		var cfg = api.StartupConfigConsumer{
			ID:          sdk.UUID(),
			Name:        conf.UI.Name,
			Description: "Autogenerated configuration for ui service",
			Type:        api.StartupConfigConsumerTypeUI,
		}
		var c = sdk.AuthUserConsumer{
			AuthConsumer: sdk.AuthConsumer{
				ID:              cfg.ID,
				Name:            cfg.Name,
				Description:     cfg.Description,
				Type:            sdk.ConsumerBuiltin,
				ValidityPeriods: validityPediod,
			},
			AuthConsumerUser: sdk.AuthUserConsumerData{
				Data: map[string]string{},
			},
		}
		conf.UI.API.Token, err = builtin.NewSigninConsumerToken(&c)
		if err != nil {
			return "", err
		}
		startupCfg.Consumers = append(startupCfg.Consumers, cfg)
	}

	if h := conf.Hatchery; h != nil {
		if h.Local != nil {
			var cfg = api.StartupConfigConsumer{
				ID:          sdk.UUID(),
				Name:        h.Local.Name,
				Description: "Autogenerated configuration for local hatchery",
				Type:        api.StartupConfigConsumerTypeHatchery,
			}
			var c = sdk.AuthUserConsumer{
				AuthConsumer: sdk.AuthConsumer{
					ID:              cfg.ID,
					Name:            cfg.Name,
					Description:     cfg.Description,
					Type:            sdk.ConsumerBuiltin,
					ValidityPeriods: validityPediod,
				},
				AuthConsumerUser: sdk.AuthUserConsumerData{
					Data: map[string]string{},
				},
			}
			h.Local.API.Token, err = builtin.NewSigninConsumerToken(&c)
			if err != nil {
				return "", err
			}
			startupCfg.Consumers = append(startupCfg.Consumers, cfg)
			privateKey, _ := jws.NewRandomRSAKey()
			privateKeyPEM, _ := jws.ExportPrivateKey(privateKey)
			h.Local.RSAPrivateKey = string(privateKeyPEM)
		}

		if h.Openstack != nil {
			var cfg = api.StartupConfigConsumer{
				ID:          sdk.UUID(),
				Name:        h.Openstack.Name,
				Description: "Autogenerated configuration for openstack hatchery",
				Type:        api.StartupConfigConsumerTypeHatchery,
			}
			var c = sdk.AuthUserConsumer{
				AuthConsumer: sdk.AuthConsumer{
					ID:              cfg.ID,
					Name:            cfg.Name,
					Description:     cfg.Description,
					Type:            sdk.ConsumerBuiltin,
					ValidityPeriods: validityPediod,
				},
				AuthConsumerUser: sdk.AuthUserConsumerData{
					Data: map[string]string{},
				},
			}
			h.Openstack.API.Token, err = builtin.NewSigninConsumerToken(&c)
			if err != nil {
				return "", err
			}
			startupCfg.Consumers = append(startupCfg.Consumers, cfg)
			privateKey, _ := jws.NewRandomRSAKey()
			privateKeyPEM, _ := jws.ExportPrivateKey(privateKey)
			h.Openstack.RSAPrivateKey = string(privateKeyPEM)
		}

		if h.VSphere != nil {
			var cfg = api.StartupConfigConsumer{
				ID:          sdk.UUID(),
				Name:        h.VSphere.Name,
				Description: "Autogenerated configuration for vsphere hatchery",
				Type:        api.StartupConfigConsumerTypeHatchery,
			}
			var c = sdk.AuthUserConsumer{
				AuthConsumer: sdk.AuthConsumer{
					ID:              cfg.ID,
					Name:            cfg.Name,
					Description:     cfg.Description,
					Type:            sdk.ConsumerBuiltin,
					ValidityPeriods: validityPediod,
				},
				AuthConsumerUser: sdk.AuthUserConsumerData{
					Data: map[string]string{},
				},
			}
			h.VSphere.API.Token, err = builtin.NewSigninConsumerToken(&c)
			if err != nil {
				return "", err
			}
			startupCfg.Consumers = append(startupCfg.Consumers, cfg)
			privateKey, _ := jws.NewRandomRSAKey()
			privateKeyPEM, _ := jws.ExportPrivateKey(privateKey)
			h.VSphere.RSAPrivateKey = string(privateKeyPEM)
		}

		if h.Swarm != nil {
			var cfg = api.StartupConfigConsumer{
				ID:          sdk.UUID(),
				Name:        h.Swarm.Name,
				Description: "Autogenerated configuration for swarm hatchery",
				Type:        api.StartupConfigConsumerTypeHatchery,
			}
			var c = sdk.AuthUserConsumer{
				AuthConsumer: sdk.AuthConsumer{
					ID:              cfg.ID,
					Name:            cfg.Name,
					Description:     cfg.Description,
					Type:            sdk.ConsumerBuiltin,
					ValidityPeriods: validityPediod,
				},
				AuthConsumerUser: sdk.AuthUserConsumerData{
					Data: map[string]string{},
				},
			}
			h.Swarm.API.Token, err = builtin.NewSigninConsumerToken(&c)
			if err != nil {
				return "", err
			}
			startupCfg.Consumers = append(startupCfg.Consumers, cfg)
			privateKey, _ := jws.NewRandomRSAKey()
			privateKeyPEM, _ := jws.ExportPrivateKey(privateKey)
			h.Swarm.RSAPrivateKey = string(privateKeyPEM)
		}

		if h.Kubernetes != nil {
			var cfg = api.StartupConfigConsumer{
				ID:          sdk.UUID(),
				Name:        h.Kubernetes.Name,
				Description: "Autogenerated configuration for kubernetes hatchery",
				Type:        api.StartupConfigConsumerTypeHatchery,
			}
			var c = sdk.AuthUserConsumer{
				AuthConsumer: sdk.AuthConsumer{
					ID:              cfg.ID,
					Name:            cfg.Name,
					Description:     cfg.Description,
					Type:            sdk.ConsumerBuiltin,
					ValidityPeriods: validityPediod,
				},
				AuthConsumerUser: sdk.AuthUserConsumerData{
					Data: map[string]string{},
				},
			}
			conf.Hatchery.Kubernetes.API.Token, err = builtin.NewSigninConsumerToken(&c)
			if err != nil {
				return "", err
			}
			startupCfg.Consumers = append(startupCfg.Consumers, cfg)
			privateKey, _ := jws.NewRandomRSAKey()
			privateKeyPEM, _ := jws.ExportPrivateKey(privateKey)
			h.Kubernetes.RSAPrivateKey = string(privateKeyPEM)
			var a kubernetes.CustomAnnotation
			defaults.SetDefaults(&a)
			conf.Hatchery.Kubernetes.CustomAnnotations = append(conf.Hatchery.Kubernetes.CustomAnnotations, a)
		}
	}

	if conf.Hooks != nil {
		var cfg = api.StartupConfigConsumer{
			ID:          sdk.UUID(),
			Name:        conf.Hooks.Name,
			Description: "Autogenerated configuration for hooks service",
			Type:        api.StartupConfigConsumerTypeHooks,
		}
		var c = sdk.AuthUserConsumer{
			AuthConsumer: sdk.AuthConsumer{
				ID:              cfg.ID,
				Name:            cfg.Name,
				Description:     cfg.Description,
				Type:            sdk.ConsumerBuiltin,
				ValidityPeriods: validityPediod,
			},
			AuthConsumerUser: sdk.AuthUserConsumerData{
				Data: map[string]string{},
			},
		}
		conf.Hooks.API.Token, err = builtin.NewSigninConsumerToken(&c)
		if err != nil {
			return "", err
		}
		startupCfg.Consumers = append(startupCfg.Consumers, cfg)
	}

	if conf.Repositories != nil {
		var cfg = api.StartupConfigConsumer{
			ID:          sdk.UUID(),
			Name:        conf.Repositories.Name,
			Description: "Autogenerated configuration for repositories service",
			Type:        api.StartupConfigConsumerTypeRepositories,
		}
		var c = sdk.AuthUserConsumer{
			AuthConsumer: sdk.AuthConsumer{
				ID:              cfg.ID,
				Name:            cfg.Name,
				Description:     cfg.Description,
				Type:            sdk.ConsumerBuiltin,
				ValidityPeriods: validityPediod,
			},
			AuthConsumerUser: sdk.AuthUserConsumerData{
				Data: map[string]string{},
			},
		}
		conf.Repositories.API.Token, err = builtin.NewSigninConsumerToken(&c)
		if err != nil {
			return "", err
		}
		startupCfg.Consumers = append(startupCfg.Consumers, cfg)
	}

	if conf.DatabaseMigrate != nil {
		var cfg = api.StartupConfigConsumer{
			ID:          sdk.UUID(),
			Name:        conf.DatabaseMigrate.Name,
			Description: "Autogenerated configuration for migrate service",
			Type:        api.StartupConfigConsumerTypeDBMigrate,
		}
		var c = sdk.AuthUserConsumer{
			AuthConsumer: sdk.AuthConsumer{
				ID:              cfg.ID,
				Name:            cfg.Name,
				Description:     cfg.Description,
				Type:            sdk.ConsumerBuiltin,
				ValidityPeriods: validityPediod,
			},
			AuthConsumerUser: sdk.AuthUserConsumerData{
				Data: map[string]string{},
			},
		}
		conf.DatabaseMigrate.API.Token, err = builtin.NewSigninConsumerToken(&c)
		if err != nil {
			return "", err
		}
		startupCfg.Consumers = append(startupCfg.Consumers, cfg)
	}

	if conf.VCS != nil {
		var cfg = api.StartupConfigConsumer{
			ID:          sdk.UUID(),
			Name:        conf.VCS.Name,
			Description: "Autogenerated configuration for vcs service",
			Type:        api.StartupConfigConsumerTypeVCS,
		}
		var c = sdk.AuthUserConsumer{
			AuthConsumer: sdk.AuthConsumer{
				ID:              cfg.ID,
				Name:            cfg.Name,
				Description:     cfg.Description,
				Type:            sdk.ConsumerBuiltin,
				ValidityPeriods: validityPediod,
			},
			AuthConsumerUser: sdk.AuthUserConsumerData{
				Data: map[string]string{},
			},
		}
		conf.VCS.API.Token, err = builtin.NewSigninConsumerToken(&c)
		if err != nil {
			return "", err
		}
		startupCfg.Consumers = append(startupCfg.Consumers, cfg)
	}

	if conf.CDN != nil {
		var cfg = api.StartupConfigConsumer{
			ID:          sdk.UUID(),
			Name:        conf.CDN.Name,
			Description: "Autogenerated configuration for cdn service",
			Type:        api.StartupConfigConsumerTypeCDN,
		}
		var c = sdk.AuthUserConsumer{
			AuthConsumer: sdk.AuthConsumer{
				ID:              cfg.ID,
				Name:            cfg.Name,
				Description:     cfg.Description,
				Type:            sdk.ConsumerBuiltin,
				ValidityPeriods: validityPediod,
			},
			AuthConsumerUser: sdk.AuthUserConsumerData{
				Data: map[string]string{},
			},
		}
		conf.CDN.API.Token, err = builtin.NewSigninConsumerToken(&c)
		if err != nil {
			return "", err
		}
		startupCfg.Consumers = append(startupCfg.Consumers, cfg)
	}

	if conf.ElasticSearch != nil {
		var cfg = api.StartupConfigConsumer{
			ID:          sdk.UUID(),
			Name:        conf.ElasticSearch.Name,
			Description: "Autogenerated configuration for elasticSearch service",
			Type:        api.StartupConfigConsumerTypeElasticsearch,
		}
		var c = sdk.AuthUserConsumer{
			AuthConsumer: sdk.AuthConsumer{
				ID:              cfg.ID,
				Name:            cfg.Name,
				Description:     cfg.Description,
				Type:            sdk.ConsumerBuiltin,
				ValidityPeriods: validityPediod,
			},
			AuthConsumerUser: sdk.AuthUserConsumerData{
				Data: map[string]string{},
			},
		}
		conf.ElasticSearch.API.Token, err = builtin.NewSigninConsumerToken(&c)
		if err != nil {
			return "", err
		}
		startupCfg.Consumers = append(startupCfg.Consumers, cfg)
	}

	return authentication.SignJWS(startupCfg, time.Now(), time.Hour)
}

func getInitTokenFromExistingConfiguration(conf Configuration) (string, error) {
	if conf.API == nil {
		return "", fmt.Errorf("cannot load configuration")
	}

	now := time.Now()
	globalIAT := now.Unix()
	startupCfg := api.StartupConfig{}

	var RSAKeyConfigs []authentication.KeyConfig
	if conf.API.Auth.RSAPrivateKey != "" {
		RSAKeyConfigs = append(RSAKeyConfigs, authentication.KeyConfig{
			Key:       conf.API.Auth.RSAPrivateKey,
			Timestamp: 0,
		})
	}
	if len(conf.API.Auth.RSAPrivateKeys) > 0 {
		RSAKeyConfigs = append(RSAKeyConfigs, conf.API.Auth.RSAPrivateKeys...)
	}

	if err := authentication.Init(context.TODO(), "cds-api", RSAKeyConfigs); err != nil {
		return "", err
	}

	if conf.UI != nil {
		consumerID, iat, err := builtin.CheckSigninConsumerToken(conf.UI.API.Token)
		if err != nil {
			return "", fmt.Errorf("cannot parse ui signin token: %v", err)
		}
		if iat < globalIAT {
			globalIAT = iat
		}
		var cfg = api.StartupConfigConsumer{
			ID:          consumerID,
			Name:        conf.UI.Name,
			Description: "Autogenerated configuration for ui service",
			Type:        api.StartupConfigConsumerTypeUI,
		}
		startupCfg.Consumers = append(startupCfg.Consumers, cfg)
	}

	if h := conf.Hatchery; h != nil {
		if h.Local != nil {
			consumerID, iat, err := builtin.CheckSigninConsumerToken(h.Local.API.Token)
			if err != nil {
				return "", fmt.Errorf("cannot parse hatchery:local signin token: %v", err)
			}
			if iat < globalIAT {
				globalIAT = iat
			}
			var cfg = api.StartupConfigConsumer{
				ID:          consumerID,
				Name:        h.Local.Name,
				Description: "Autogenerated configuration for local hatchery",
				Type:        api.StartupConfigConsumerTypeHatchery,
			}
			startupCfg.Consumers = append(startupCfg.Consumers, cfg)
		}

		if h.Openstack != nil {
			consumerID, iat, err := builtin.CheckSigninConsumerToken(h.Openstack.API.Token)
			if err != nil {
				return "", fmt.Errorf("cannot parse hatchery:openstack signin token: %v", err)
			}
			if iat < globalIAT {
				globalIAT = iat
			}
			var cfg = api.StartupConfigConsumer{
				ID:          consumerID,
				Name:        h.Openstack.Name,
				Description: "Autogenerated configuration for openstack hatchery",
				Type:        api.StartupConfigConsumerTypeHatchery,
			}
			startupCfg.Consumers = append(startupCfg.Consumers, cfg)
		}

		if h.VSphere != nil {
			consumerID, iat, err := builtin.CheckSigninConsumerToken(h.VSphere.API.Token)
			if err != nil {
				return "", fmt.Errorf("cannot parse hatchery:vsphere signin token: %v", err)
			}
			if iat < globalIAT {
				globalIAT = iat
			}
			var cfg = api.StartupConfigConsumer{
				ID:          consumerID,
				Name:        h.VSphere.Name,
				Description: "Autogenerated configuration for vsphere hatchery",
				Type:        api.StartupConfigConsumerTypeHatchery,
			}
			startupCfg.Consumers = append(startupCfg.Consumers, cfg)
		}

		if h.Swarm != nil {
			consumerID, iat, err := builtin.CheckSigninConsumerToken(h.Swarm.API.Token)
			if err != nil {
				return "", fmt.Errorf("cannot parse hatchery:swarm signin token: %v", err)
			}
			if iat < globalIAT {
				globalIAT = iat
			}
			var cfg = api.StartupConfigConsumer{
				ID:          consumerID,
				Name:        h.Swarm.Name,
				Description: "Autogenerated configuration for swarm hatchery",
				Type:        api.StartupConfigConsumerTypeHatchery,
			}
			startupCfg.Consumers = append(startupCfg.Consumers, cfg)
		}

		if h.Kubernetes != nil {
			consumerID, iat, err := builtin.CheckSigninConsumerToken(h.Kubernetes.API.Token)
			if err != nil {
				return "", fmt.Errorf("cannot parse hatchery:kubernetes signin token: %v", err)
			}
			if iat < globalIAT {
				globalIAT = iat
			}
			var cfg = api.StartupConfigConsumer{
				ID:          consumerID,
				Name:        h.Kubernetes.Name,
				Description: "Autogenerated configuration for kubernetes hatchery",
				Type:        api.StartupConfigConsumerTypeHatchery,
			}
			startupCfg.Consumers = append(startupCfg.Consumers, cfg)
		}
	}

	if conf.Hooks != nil {
		consumerID, iat, err := builtin.CheckSigninConsumerToken(conf.Hooks.API.Token)
		if err != nil {
			return "", fmt.Errorf("cannot parse hooks signin token: %v", err)
		}
		if iat < globalIAT {
			globalIAT = iat
		}
		var cfg = api.StartupConfigConsumer{
			ID:          consumerID,
			Name:        conf.Hooks.Name,
			Description: "Autogenerated configuration for hooks service",
			Type:        api.StartupConfigConsumerTypeHooks,
		}
		startupCfg.Consumers = append(startupCfg.Consumers, cfg)
	}

	if conf.Repositories != nil {
		consumerID, iat, err := builtin.CheckSigninConsumerToken(conf.Repositories.API.Token)
		if err != nil {
			return "", fmt.Errorf("cannot parse hooks repositories token: %v", err)
		}
		if iat < globalIAT {
			globalIAT = iat
		}
		var cfg = api.StartupConfigConsumer{
			ID:          consumerID,
			Name:        conf.Repositories.Name,
			Description: "Autogenerated configuration for repositories service",
			Type:        api.StartupConfigConsumerTypeRepositories,
		}
		startupCfg.Consumers = append(startupCfg.Consumers, cfg)
	}

	if conf.DatabaseMigrate != nil {
		consumerID, iat, err := builtin.CheckSigninConsumerToken(conf.DatabaseMigrate.API.Token)
		if err != nil {
			return "", fmt.Errorf("cannot parse database migrate token: %v", err)
		}
		if iat < globalIAT {
			globalIAT = iat
		}
		var cfg = api.StartupConfigConsumer{
			ID:          consumerID,
			Name:        conf.DatabaseMigrate.Name,
			Description: "Autogenerated configuration for migrate service",
			Type:        api.StartupConfigConsumerTypeDBMigrate,
		}
		startupCfg.Consumers = append(startupCfg.Consumers, cfg)
	}

	if conf.VCS != nil {
		consumerID, iat, err := builtin.CheckSigninConsumerToken(conf.VCS.API.Token)
		if err != nil {
			return "", fmt.Errorf("cannot parse vcs token: %v", err)
		}
		if iat < globalIAT {
			globalIAT = iat
		}
		var cfg = api.StartupConfigConsumer{
			ID:          consumerID,
			Name:        conf.VCS.Name,
			Description: "Autogenerated configuration for vcs service",
			Type:        api.StartupConfigConsumerTypeVCS,
		}
		startupCfg.Consumers = append(startupCfg.Consumers, cfg)
	}

	if conf.CDN != nil {
		consumerID, iat, err := builtin.CheckSigninConsumerToken(conf.CDN.API.Token)
		if err != nil {
			return "", fmt.Errorf("cannot parse cdn token: %v", err)
		}
		if iat < globalIAT {
			globalIAT = iat
		}
		var cfg = api.StartupConfigConsumer{
			ID:          consumerID,
			Name:        conf.CDN.Name,
			Description: "Autogenerated configuration for cdn service",
			Type:        api.StartupConfigConsumerTypeCDN,
		}
		startupCfg.Consumers = append(startupCfg.Consumers, cfg)
	}

	if conf.ElasticSearch != nil {
		consumerID, iat, err := builtin.CheckSigninConsumerToken(conf.ElasticSearch.API.Token)
		if err != nil {
			return "", fmt.Errorf("cannot parse elasticsearch token: %v", err)
		}
		if iat < globalIAT {
			globalIAT = iat
		}
		var cfg = api.StartupConfigConsumer{
			ID:          consumerID,
			Name:        conf.ElasticSearch.Name,
			Description: "Autogenerated configuration for elasticSearch service",
			Type:        api.StartupConfigConsumerTypeElasticsearch,
		}
		startupCfg.Consumers = append(startupCfg.Consumers, cfg)
	}

	startupCfg.IAT = globalIAT

	return authentication.SignJWS(startupCfg, now, time.Hour)
}
