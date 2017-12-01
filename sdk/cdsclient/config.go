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
