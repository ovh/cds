package workflow

var baseUIURL, defaultOS, defaultArch string

//Initialize starts goroutines for workflows
func Initialize(uiURL, confDefaultOS, confDefaultArch string) {
	baseUIURL = uiURL
	defaultOS = confDefaultOS
	defaultArch = confDefaultArch
}
