package smtpmock

type Message struct {
	FromAgent      string `json:"from-agent"`
	RemoteAddress  string `json:"remote-address"`
	User           string `json:"user"`
	From           string `json:"from"`
	To             string `json:"to"`
	Content        string `json:"content"`
	ContentDecoded string `json:"content-decoded"`
}
