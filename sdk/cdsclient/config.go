package cdsclient

//Config is the configuration data used by the cdsclient interface implementation
type Config struct {
	Host      string
	User      string
	Token     string
	Hash      string
	userAgent string
	Verbose   bool
	Retry     int
}

//ProviderConfig is the configuration data used by the cdsclient ProviderClient interface implementation
type ProviderConfig struct {
	Host                  string
	Name                  string
	Token                 string
	RequestSecondsTimeout int
	InsecureSkipVerifyTLS bool
}
