package main

import (
	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/api/database"
)

func init() {
	pflags := mainCmd.PersistentFlags()
	pflags.String("db-user", "cds", "DB User")
	pflags.String("db-password", "", "DB Password")
	pflags.String("db-name", "cds", "DB Name")
	pflags.String("db-host", "localhost", "DB Host")
	pflags.String("db-port", "5432", "DB Port")
	pflags.String("db-sslmode", "require", "DB SSL Mode: require (default), verify-full, or disable")
	pflags.Int("db-maxconn", 20, "DB Max connection")
	pflags.Int("db-timeout", 3000, "Statement timeout value")
	viper.BindPFlag("db_user", pflags.Lookup("db-user"))
	viper.BindPFlag("db_password", pflags.Lookup("db-password"))
	viper.BindPFlag("db_name", pflags.Lookup("db-name"))
	viper.BindPFlag("db_host", pflags.Lookup("db-host"))
	viper.BindPFlag("db_port", pflags.Lookup("db-port"))
	viper.BindPFlag("db_sslmode", pflags.Lookup("db-sslmode"))
	viper.BindPFlag("db_maxconn", pflags.Lookup("db-maxconn"))
	viper.BindPFlag("db_timeout", pflags.Lookup("db-timeout"))

	pflags.String("db-secret", "cds/db", "DB Secret: used in secret backend manager")
	viper.BindPFlag("db_secret", pflags.Lookup("db-secret"))

	flags := mainCmd.Flags()

	flags.String("log-level", "notice", "Log Level : debug, info, notice, warning, critical")
	viper.BindPFlag("log_level", flags.Lookup("log-level"))

	flags.Bool("db-logging", false, "Logging in Database: true of false")
	viper.BindPFlag("db_logging", flags.Lookup("db-logging"))

	flags.String("base-url", "", "CDS UI Base URL")
	viper.BindPFlag("base_url", flags.Lookup("base-url"))

	flags.String("api-url", "", "CDS API Base URL")
	viper.BindPFlag("api_url", flags.Lookup("api-url"))

	flags.String("listen-port", "8081", "CDS Engine HTTP(S) Port")
	viper.BindPFlag("listen_port", flags.Lookup("listen-port"))

	flags.Int("grpc-port", 8082, "CDS Engine GRPC Port")
	viper.BindPFlag("grpc_port", flags.Lookup("grpc-port"))

	flags.String("artifact-mode", "filesystem", "Artifact Mode: openstack or filesystem")
	flags.String("artifact-address", "", "Artifact Adress: used with --artifact-mode=openstask")
	flags.String("artifact-user", "", "Artifact User: used with --artifact-mode=openstask")
	flags.String("artifact-password", "", "Artifact Password: used with --artifact-mode=openstask")
	flags.String("artifact-tenant", "", "Artifact Tenant: used with --artifact-mode=openstask")
	flags.String("artifact-region", "", "Artifact Region: used with --artifact-mode=openstask")
	flags.String("artifact-basedir", "/tmp", "Artifact Basedir: used with --artifact-mode=filesystem")
	viper.BindPFlag("artifact_mode", flags.Lookup("artifact-mode"))
	viper.BindPFlag("artifact_address", flags.Lookup("artifact-address"))
	viper.BindPFlag("artifact_user", flags.Lookup("artifact-user"))
	viper.BindPFlag("artifact_password", flags.Lookup("artifact-password"))
	viper.BindPFlag("artifact_tenant", flags.Lookup("artifact-tenant"))
	viper.BindPFlag("artifact_region", flags.Lookup("artifact-region"))
	viper.BindPFlag("artifact_basedir", flags.Lookup("artifact-basedir"))

	flags.Bool("no-smtp", true, "No SMTP mode: true or false")
	flags.String("smtp-host", "", "SMTP Host")
	flags.String("smtp-port", "", "SMTP Port")
	flags.Bool("smtp-tls", false, "SMTP TLS")
	flags.String("smtp-user", "", "SMTP Username")
	flags.String("smtp-password", "", "SMTP Password")
	flags.String("smtp-from", "", "SMTP From")
	viper.BindPFlag("no_smtp", flags.Lookup("no-smtp"))
	viper.BindPFlag("smtp_host", flags.Lookup("smtp-host"))
	viper.BindPFlag("smtp_port", flags.Lookup("smtp-port"))
	viper.BindPFlag("smtp_tls", flags.Lookup("smtp-tls"))
	viper.BindPFlag("smtp_user", flags.Lookup("smtp-user"))
	viper.BindPFlag("smtp_password", flags.Lookup("smtp-password"))
	viper.BindPFlag("smtp_from", flags.Lookup("smtp-from"))

	flags.String("download-directory", "/app", "Directory prefix for cds binaries")
	viper.BindPFlag("download_directory", flags.Lookup("download-directory"))

	flags.String("keys-directory", "/app/keys", "Directory keys for repositories managers")
	viper.BindPFlag("keys_directory", flags.Lookup("keys-directory"))

	flags.Bool("ldap-enable", false, "Enable LDAP Auth mode : true|false")
	viper.BindPFlag("ldap_enable", flags.Lookup("ldap-enable"))

	flags.String("ldap-host", "", "LDAP Host")
	viper.BindPFlag("ldap_host", flags.Lookup("ldap-host"))

	flags.Int("ldap-port", 636, "LDAP Post")
	viper.BindPFlag("ldap_port", flags.Lookup("ldap-port"))

	flags.Bool("ldap-ssl", true, "LDAP SSL mode")
	viper.BindPFlag("ldap_ssl", flags.Lookup("ldap-ssl"))

	flags.String("ldap-base", "", "LDAP Base")
	viper.BindPFlag("ldap_base", flags.Lookup("ldap-base"))

	flags.String("ldap-dn", "uid=%s,ou=people,{{.ldap-base}}", "LDAP Bind DN")
	viper.BindPFlag("ldap_dn", flags.Lookup("ldap-dn"))

	flags.String("ldap-user-fullname", "{{.givenName}} {{.sn}}", "LDAP User fullname")
	viper.BindPFlag("ldap_user_fullname", flags.Lookup("ldap-user-fullname"))

	flags.String("secret-backend", "", "Secret Backend plugin")
	viper.BindPFlag("secret_backend", flags.Lookup("secret-backend"))

	flags.StringSlice("secret-backend-option", []string{}, "Secret Backend plugin options")
	viper.BindPFlag("secret_backend_option", flags.Lookup("secret-backend-option"))

	flags.String("redis-host", "localhost:6379", "Redis hostname")
	viper.BindPFlag("redis_host", flags.Lookup("redis-host"))

	flags.String("redis-password", "", "Redis password")
	viper.BindPFlag("redis_password", flags.Lookup("redis-password"))

	flags.String("cache", "local", "Cache : local|redis")
	viper.BindPFlag("cache", flags.Lookup("cache"))

	flags.Int("cache-ttl", 600, "Cache Time to Live (seconds)")
	viper.BindPFlag("cache_ttl", flags.Lookup("cache-ttl"))

	flags.String("auth-local-mode", "basic", "Authentification mode  basic|session")
	viper.BindPFlag("auth_local_mode", flags.Lookup("auth-local-mode"))

	flags.Int("session-ttl", 60, "Session Time to Live (minutes)")
	viper.BindPFlag("session_ttl", flags.Lookup("session-ttl"))

	flags.Bool("event-kafka-enabled", false, "Enable Event over Kafka")
	viper.BindPFlag("event_kafka_enabled", flags.Lookup("event-kafka-enabled"))

	flags.String("event-kafka-broker-addresses", "", "Ex: --event-kafka-broker-addresses=host:port,host2:port2")
	viper.BindPFlag("event_kafka_broker_addresses", flags.Lookup("event-kafka-broker-addresses"))

	flags.String("event-kafka-topic", "", "Ex: --kafka-topic=your-kafka-topic")
	viper.BindPFlag("event_kafka_topic", flags.Lookup("event-kafka-topic"))

	flags.String("event-kafka-user", "", "Ex: --kafka-user=your-kafka-user")
	viper.BindPFlag("event_kafka_user", flags.Lookup("event-kafka-user"))

	flags.String("event-kafka-password", "", "Ex: --kafka-password=your-kafka-password")
	viper.BindPFlag("event_kafka_password", flags.Lookup("event-kafka-password"))

	flags.Bool("no-scheduler", false, "Disable CDS Scheduler (crontab)")
	viper.BindPFlag("no_scheduler", flags.Lookup("no-scheduler"))

	flags.Bool("no-repo-polling", false, "Disable repositories manager polling")
	viper.BindPFlag("no_repo_polling", flags.Lookup("no-repo-polling"))

	flags.Bool("no-repo-cache-loader", false, "Disable repositories cache loader")
	viper.BindPFlag("no_repo_cache_loader", flags.Lookup("no-repo-cache-loader"))

	flags.Bool("no-stash-status", false, "Disable Stash Statuses")
	viper.BindPFlag("no_stash_status", flags.Lookup("no-stash-status"))

	flags.Bool("no-github-status", false, "Disable Github Statuses")
	viper.BindPFlag("no_github_status", flags.Lookup("no-github-status"))

	flags.Bool("no-github-status-url", false, "Disable Target URL in Github Statuses")
	viper.BindPFlag("no_github_status_url", flags.Lookup("no-github-status-url"))

	flags.String("default-group", "", "Default group for new users")
	viper.BindPFlag("default_group", flags.Lookup("default-group"))

	mainCmd.AddCommand(database.DBCmd)
}
