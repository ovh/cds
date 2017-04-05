package repostash

var (
	apiURL string
	uiURL  string
)

// Init initializes repostash package
func Init(apiurl, uiurl string) {
	apiURL = apiurl
	uiURL = uiurl
}
