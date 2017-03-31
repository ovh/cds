package web

// Action represents what can be done with web executor
type Action struct {
	Click    string `yaml:"search,omitempty"`
	Fill     []Fill `yaml:"fill,omitempty"`
	Find     string `yaml:"find,omitempty"`
	Navigate string `yaml:"find,omitempty"`
}

// Fill represents informations needed to fill input/textarea
type Fill struct {
	Find string `yaml:"search,omitempty"`
	Text string `yaml:"text,omitempty"`
}
