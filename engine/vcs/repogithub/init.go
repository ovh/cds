package github

var (
	apiURL string
	uiURL  string
)

// Init initializes github package
func Init(apiurl, uiurl string) {
	apiURL = apiurl
	uiURL = uiurl
}
