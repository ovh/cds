package appium

import "github.com/sclevine/agouti"

type Option func(*config)

func AgoutiOptions(options ...agouti.Option) Option {
	return func(c *config) {
		c.agoutiOptions = options
	}
}

func Desired(capabilities agouti.Capabilities) Option {
	return func(c *config) {
		c.agoutiOptions = append(c.agoutiOptions, agouti.Desired(capabilities))
	}
}

type config struct {
	agoutiOptions []agouti.Option
}

func (c config) merge(options []Option) *config {
	for _, option := range options {
		option(&c)
	}
	return &c
}
