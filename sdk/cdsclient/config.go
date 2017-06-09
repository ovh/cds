package cdsclient

type Config struct {
	Host      string
	User      string
	Password  string
	Token     string
	Hash      string
	userAgent string
	Verbose   bool
	Retry     int
}
