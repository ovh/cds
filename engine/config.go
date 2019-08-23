package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fsamin/go-dump"
	defaults "github.com/mcuadros/go-defaults"
	"github.com/ovh/symmecrypt/keyloader"
	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/engine/api/services"
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
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/namesgenerator"
)

const (
	vaultConfKey = "/secret/cds/conf"
)

func configBootstrap(args []string) Configuration {
	var conf Configuration
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
			conf.API.Services = append(conf.API.Services, api.ServiceConfiguration{
				Name:       "sample-service",
				URL:        "https://ovh.github.io",
				Port:       "443",
				Path:       "/cds",
				HealthPath: "/cds",
				HealthPort: "443",
				HealthURL:  "https://ovh.github.io",
				Type:       "doc",
			})
		case "migrate":
			conf.DatabaseMigrate = &migrateservice.Configuration{}
			defaults.SetDefaults(conf.DatabaseMigrate)
			conf.DatabaseMigrate.Name = "cds-migrate-" + namesgenerator.GetRandomNameCDS(0)
		case "hatchery:local":
			conf.Hatchery.Local = &local.HatcheryConfiguration{}
			defaults.SetDefaults(conf.Hatchery.Local)
			conf.Hatchery.Local.Name = "cds-hatchery-local-" + namesgenerator.GetRandomNameCDS(0)
		case "hatchery:kubernetes":
			conf.Hatchery.Kubernetes = &kubernetes.HatcheryConfiguration{}
			defaults.SetDefaults(conf.Hatchery.Kubernetes)
			conf.Hatchery.Kubernetes.Name = "cds-hatchery-kubernetes-" + namesgenerator.GetRandomNameCDS(0)
		case "hatchery:marathon":
			conf.Hatchery.Marathon = &marathon.HatcheryConfiguration{}
			defaults.SetDefaults(conf.Hatchery.Marathon)
			conf.Hatchery.Marathon.Name = "cds-hatchery-marathon-" + namesgenerator.GetRandomNameCDS(0)
		case "hatchery:openstack":
			conf.Hatchery.Openstack = &openstack.HatcheryConfiguration{}
			defaults.SetDefaults(conf.Hatchery.Openstack)
			conf.Hatchery.Openstack.Name = "cds-hatchery-openstack-" + namesgenerator.GetRandomNameCDS(0)
		case "hatchery:swarm":
			conf.Hatchery.Swarm = &swarm.HatcheryConfiguration{}
			defaults.SetDefaults(conf.Hatchery.Swarm)
			conf.Hatchery.Swarm.DockerEngines = map[string]swarm.DockerEngineConfiguration{
				"sample-docker-engine": {
					Host: "///var/run/docker.sock",
				},
			}
			conf.Hatchery.Swarm.Name = "cds-hatchery-swarm-" + namesgenerator.GetRandomNameCDS(0)
		case "hatchery:vsphere":
			conf.Hatchery.VSphere = &vsphere.HatcheryConfiguration{}
			defaults.SetDefaults(conf.Hatchery.VSphere)
			conf.Hatchery.VSphere.Name = "cds-hatchery-vsphere-" + namesgenerator.GetRandomNameCDS(0)
		case "hooks":
			conf.Hooks = &hooks.Configuration{}
			defaults.SetDefaults(conf.Hooks)
			conf.Hooks.Name = "cds-hooks-" + namesgenerator.GetRandomNameCDS(0)
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
			conf.VCS.Name = "cds-vcs-" + namesgenerator.GetRandomNameCDS(0)
		case "repositories":
			conf.Repositories = &repositories.Configuration{}
			defaults.SetDefaults(conf.Repositories)
			conf.Repositories.Name = "cds-repositories-" + namesgenerator.GetRandomNameCDS(0)
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

	var startupCfg api.StartupConfig

	if err := authentication.Init("cds-api", apiPrivateKeyPEM); err != nil {
		return "", err
	}

	if conf.API != nil {
		conf.API.Auth.RSAPrivateKey = string(apiPrivateKeyPEM)
		conf.API.Secrets.Key = sdk.RandomString(32)

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
	}

	if h := conf.Hatchery; h != nil {
		if h.Local != nil {
			var cfg = api.StartupConfigService{
				ID:          sdk.UUID(),
				Name:        "hatchery:local",
				Description: "Autogenerated configuration for local hatchery",
				ServiceType: services.TypeHatchery,
			}

			var c = sdk.AuthConsumer{
				ID:          cfg.ID,
				Name:        cfg.Name,
				Description: cfg.Description,

				Type:     sdk.ConsumerBuiltin,
				Data:     map[string]string{},
				GroupIDs: []int64{},
				Scopes:   []sdk.AuthConsumerScope{sdk.AuthConsumerScopeHatchery, sdk.AuthConsumerScopeRunExecution, sdk.AuthConsumerScopeService},
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
			var cfg = api.StartupConfigService{
				ID:          sdk.UUID(),
				Name:        "hatchery:openstack",
				Description: "Autogenerated configuration for openstack hatchery",
				ServiceType: services.TypeHatchery,
			}

			var c = sdk.AuthConsumer{
				ID:          cfg.ID,
				Name:        cfg.Name,
				Description: cfg.Description,
				Type:        sdk.ConsumerBuiltin,
				Data:        map[string]string{},
				GroupIDs:    []int64{},
				Scopes:      []sdk.AuthConsumerScope{sdk.AuthConsumerScopeHatchery, sdk.AuthConsumerScopeRunExecution, sdk.AuthConsumerScopeService},
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
			var cfg = api.StartupConfigService{
				ID:          sdk.UUID(),
				Name:        "hatchery:vsphere",
				Description: "Autogenerated configuration for vsphere hatchery",
				ServiceType: services.TypeHatchery,
			}

			var c = sdk.AuthConsumer{
				ID:          cfg.ID,
				Name:        cfg.Name,
				Description: cfg.Description,

				Type:     sdk.ConsumerBuiltin,
				Data:     map[string]string{},
				GroupIDs: []int64{},
				Scopes:   []sdk.AuthConsumerScope{sdk.AuthConsumerScopeHatchery, sdk.AuthConsumerScopeRunExecution, sdk.AuthConsumerScopeService},
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
			var cfg = api.StartupConfigService{
				ID:          sdk.UUID(),
				Name:        "hatchery:swarm",
				Description: "Autogenerated configuration for swarm hatchery",
				ServiceType: services.TypeHatchery,
			}

			var c = sdk.AuthConsumer{
				ID:          cfg.ID,
				Name:        cfg.Name,
				Description: cfg.Description,

				Type:     sdk.ConsumerBuiltin,
				Data:     map[string]string{},
				GroupIDs: []int64{},
				Scopes:   []sdk.AuthConsumerScope{sdk.AuthConsumerScopeHatchery, sdk.AuthConsumerScopeRunExecution, sdk.AuthConsumerScopeService},
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
		if h.Marathon != nil {
			var cfg = api.StartupConfigService{
				ID:          sdk.UUID(),
				Name:        "hatchery:marathon",
				Description: "Autogenerated configuration for marathon hatchery",
				ServiceType: services.TypeHatchery,
			}

			var c = sdk.AuthConsumer{
				ID:          cfg.ID,
				Name:        cfg.Name,
				Description: cfg.Description,
				Type:        sdk.ConsumerBuiltin,
				Data:        map[string]string{},
				GroupIDs:    []int64{},
				Scopes:      []sdk.AuthConsumerScope{sdk.AuthConsumerScopeHatchery, sdk.AuthConsumerScopeRunExecution, sdk.AuthConsumerScopeService},
			}

			conf.Hatchery.Marathon.API.Token, err = builtin.NewSigninConsumerToken(&c)
			if err != nil {
				return "", err
			}

			startupCfg.Consumers = append(startupCfg.Consumers, cfg)
			privateKey, _ := jws.NewRandomRSAKey()
			privateKeyPEM, _ := jws.ExportPrivateKey(privateKey)
			h.Marathon.RSAPrivateKey = string(privateKeyPEM)
		}
		if h.Kubernetes != nil {
			var cfg = api.StartupConfigService{
				ID:          sdk.UUID(),
				Name:        "hatchery:kubernetes",
				Description: "Autogenerated configuration for kubernetes hatchery",
				ServiceType: services.TypeHatchery,
			}

			var c = sdk.AuthConsumer{
				ID:          cfg.ID,
				Name:        cfg.Name,
				Description: cfg.Description,
				Type:        sdk.ConsumerBuiltin,
				Data:        map[string]string{},
				GroupIDs:    []int64{},
				Scopes:      []sdk.AuthConsumerScope{sdk.AuthConsumerScopeHatchery, sdk.AuthConsumerScopeRunExecution, sdk.AuthConsumerScopeService},
			}

			conf.Hatchery.Kubernetes.API.Token, err = builtin.NewSigninConsumerToken(&c)
			if err != nil {
				return "", err
			}

			startupCfg.Consumers = append(startupCfg.Consumers, cfg)
			privateKey, _ := jws.NewRandomRSAKey()
			privateKeyPEM, _ := jws.ExportPrivateKey(privateKey)
			h.Kubernetes.RSAPrivateKey = string(privateKeyPEM)
		}
	}

	if conf.Hooks != nil {
		var cfg = api.StartupConfigService{
			ID:          sdk.UUID(),
			Name:        "hooks",
			Description: "Autogenerated configuration for hooks service",
			ServiceType: services.TypeHooks,
		}

		var c = sdk.AuthConsumer{
			ID:          cfg.ID,
			Name:        cfg.Name,
			Description: cfg.Description,
			Type:        sdk.ConsumerBuiltin,
			Data:        map[string]string{},
			GroupIDs:    []int64{},
			Scopes: []sdk.AuthConsumerScope{
				sdk.AuthConsumerScopeService,
				sdk.AuthConsumerScopeHooks,
				sdk.AuthConsumerScopeProject,
				sdk.AuthConsumerScopeRun,
			},
		}

		conf.Hooks.API.Token, err = builtin.NewSigninConsumerToken(&c)
		if err != nil {
			return "", err
		}

		startupCfg.Consumers = append(startupCfg.Consumers, cfg)
	}

	if conf.Repositories != nil {
		var cfg = api.StartupConfigService{
			ID:          sdk.UUID(),
			Name:        "repositories",
			Description: "Autogenerated configuration for repositories service",
			ServiceType: services.TypeHooks,
		}

		var c = sdk.AuthConsumer{
			ID:          cfg.ID,
			Name:        cfg.Name,
			Description: cfg.Description,
			Type:        sdk.ConsumerBuiltin,
			Data:        map[string]string{},
			GroupIDs:    []int64{},
			Scopes:      []sdk.AuthConsumerScope{sdk.AuthConsumerScopeService},
		}

		conf.Repositories.API.Token, err = builtin.NewSigninConsumerToken(&c)
		if err != nil {
			return "", err
		}

		startupCfg.Consumers = append(startupCfg.Consumers, cfg)
	}

	if conf.DatabaseMigrate != nil {
		var cfg = api.StartupConfigService{
			ID:          sdk.UUID(),
			Name:        "migrate",
			Description: "Autogenerated configuration for migrate service",
			ServiceType: services.TypeDBMigrate,
		}

		var c = sdk.AuthConsumer{
			ID:          cfg.ID,
			Name:        cfg.Name,
			Description: cfg.Description,
			Type:        sdk.ConsumerBuiltin,
			Data:        map[string]string{},
			GroupIDs:    []int64{},
			Scopes:      []sdk.AuthConsumerScope{sdk.AuthConsumerScopeService},
		}

		conf.DatabaseMigrate.API.Token, err = builtin.NewSigninConsumerToken(&c)
		if err != nil {
			return "", err
		}

		startupCfg.Consumers = append(startupCfg.Consumers, cfg)
	}

	if conf.VCS != nil {
		var cfg = api.StartupConfigService{
			ID:          sdk.UUID(),
			Name:        "vcs",
			Description: "Autogenerated configuration for vcs service",
			ServiceType: services.TypeVCS,
		}

		var c = sdk.AuthConsumer{
			ID:          cfg.ID,
			Name:        cfg.Name,
			Description: cfg.Description,
			Type:        sdk.ConsumerBuiltin,
			Data:        map[string]string{},
			GroupIDs:    []int64{},
			Scopes:      []sdk.AuthConsumerScope{sdk.AuthConsumerScopeService},
		}

		conf.VCS.API.Token, err = builtin.NewSigninConsumerToken(&c)
		if err != nil {
			return "", err
		}

		startupCfg.Consumers = append(startupCfg.Consumers, cfg)
	}

	if conf.ElasticSearch != nil {
		var cfg = api.StartupConfigService{
			ID:          sdk.UUID(),
			Name:        "elasticSearch",
			Description: "Autogenerated configuration for elasticSearch service",
			ServiceType: services.TypeElasticsearch,
		}

		var c = sdk.AuthConsumer{
			ID:          cfg.ID,
			Name:        cfg.Name,
			Description: cfg.Description,
			Type:        sdk.ConsumerBuiltin,
			Data:        map[string]string{},
			GroupIDs:    []int64{},
			Scopes:      []sdk.AuthConsumerScope{sdk.AuthConsumerScopeService},
		}

		conf.ElasticSearch.API.Token, err = builtin.NewSigninConsumerToken(&c)
		if err != nil {
			return "", err
		}

		startupCfg.Consumers = append(startupCfg.Consumers, cfg)
	}

	return authentication.SignJWS(startupCfg, time.Hour)
}
