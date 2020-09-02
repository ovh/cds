package database

import (
	"github.com/ovh/symmecrypt/keyloader"
)

// DBConfigurationWithEncryption is the exposed type for database configuration that requires encryption.
type DBConfigurationWithEncryption struct {
	User           string           `toml:"user" default:"cds" json:"user"`
	Role           string           `toml:"role" default:"" commented:"true" comment:"Set a specific role to run SET ROLE for each connection" json:"role"`
	Password       string           `toml:"password" default:"cds" json:"-"`
	Name           string           `toml:"name" default:"cds" json:"name"`
	Schema         string           `toml:"schema" json:"schema" comment:"Database schema name, default value is 'public'"`
	Host           string           `toml:"host" default:"localhost" json:"host"`
	Port           int              `toml:"port" default:"5432" json:"port"`
	SSLMode        string           `toml:"sslmode" default:"disable" comment:"DB SSL Mode: require (default), verify-full, or disable" json:"sslmode"`
	MaxConn        int              `toml:"maxconn" default:"20" comment:"DB Max connection" json:"maxconn"`
	ConnectTimeout int              `toml:"connectTimeout" default:"10" comment:"Maximum wait for connection, in seconds" json:"connectTimeout"`
	Timeout        int              `toml:"timeout" default:"3000" comment:"Statement timeout value in milliseconds" json:"timeout"`
	SignatureKey   RollingKeyConfig `json:"-" toml:"signatureRollingKeys" comment:"Signature rolling keys" mapstructure:"signatureRollingKeys"`
	EncryptionKey  RollingKeyConfig `json:"-" toml:"encryptionRollingKeys" comment:"Encryption rolling keys" mapstructure:"encryptionRollingKeys"`
}

// DBConfiguration is the exposed type for database configuration that is used by migrate service.
type DBConfiguration struct {
	User           string `toml:"user" default:"cds" json:"user"`
	Role           string `toml:"role" default:"" commented:"true" comment:"Set a specific role to run SET ROLE for each connection" json:"role"`
	Password       string `toml:"password" default:"cds" json:"-"`
	Name           string `toml:"name" default:"cds" json:"name"`
	Schema         string `toml:"schema" json:"schema"`
	Host           string `toml:"host" default:"localhost" json:"host"`
	Port           int    `toml:"port" default:"5432" json:"port"`
	SSLMode        string `toml:"sslmode" default:"disable" comment:"DB SSL Mode: require (default), verify-full, or disable" json:"sslmode"`
	MaxConn        int    `toml:"maxconn" default:"20" comment:"DB Max connection" json:"maxconn"`
	ConnectTimeout int    `toml:"connectTimeout" default:"10" comment:"Maximum wait for connection, in seconds" json:"connectTimeout"`
	Timeout        int    `toml:"timeout" default:"3000" comment:"Statement timeout value in milliseconds" json:"timeout"`
}

type RollingKeyConfig struct {
	Cipher string      `toml:"cipher" mapstructure:"cipher"`
	Keys   []KeyConfig `toml:"keys" mapstructure:"keys"`
}

type KeyConfig struct {
	Timestamp int64  `toml:"timestamp,omitempty" mapstructure:"timestamp"`
	Key       string `toml:"key" mapstructure:"key"`
}

func (k RollingKeyConfig) GetKeys(identifier string) []keyloader.KeyConfig {
	var res = []keyloader.KeyConfig{}
	for _, key := range k.Keys {
		res = append(res, keyloader.KeyConfig{
			Identifier: identifier,
			Cipher:     k.Cipher,
			Key:        key.Key,
			Timestamp:  key.Timestamp,
		})
	}
	return res
}
