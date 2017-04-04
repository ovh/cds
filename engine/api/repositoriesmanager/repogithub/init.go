package repogithub

var (
	apiURL string
	uiURL  string
)

// Init initializes repogithub package
func Init(apiurl, uiurl string) {
	apiURL = apiurl
	uiURL = uiurl
}
