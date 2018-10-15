package database

const (
	// ViolateUniqueKeyPGCode is the pg code when duplicating unique key
	ViolateUniqueKeyPGCode = "23505"
)

// DBConfiguration is the exposed type for database API configuration
type DBConfiguration struct {
	User           string `toml:"user" default:"cds" json:"user"`
	Role           string `toml:"role" default:"" commented:"true" comment:"Set a specific role to run SET ROLE for each connection" json:"role"`
	Password       string `toml:"password" default:"cds" json:"-"`
	Name           string `toml:"name" default:"cds" json:"name"`
	Host           string `toml:"host" default:"localhost" json:"host"`
	Port           int    `toml:"port" default:"5432" json:"port"`
	SSLMode        string `toml:"sslmode" default:"disable" comment:"DB SSL Mode: require (default), verify-full, or disable" json:"sslmode"`
	MaxConn        int    `toml:"maxconn" default:"20" comment:"DB Max connection" json:"maxconn"`
	ConnectTimeout int    `toml:"connectTimeout" default:"10" comment:"Maximum wait for connection, in seconds" json:"connectTimeout"`
	Timeout        int    `toml:"timeout" default:"3000" comment:"Statement timeout value in milliseconds" json:"timeout"`
}
