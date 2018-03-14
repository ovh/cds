package main

// Configuration is the configuraton structure for CDS API
type Configuration struct {
	Kafka struct {
		Brokers  string `toml:"brokers"`
		Topic    string `toml:"topic"`
		User     string `toml:"user"`
		Password string `toml:"password"`
		Group    string `toml:"group"`
	}
	ElasticSearch struct {
		Protocol string `toml:"protocol"`
		Domain   string `toml:"domain"`
		Port     string `toml:"port"`
		Username string `toml:"username"`
		Password string `toml:"password"`
		Index    string `toml:"index"`
	}
	Debug struct {
		LogLevel string `toml:"log_level"`
	}
}
