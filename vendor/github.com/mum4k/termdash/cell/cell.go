// Copyright 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package cell implements cell options and attributes.
package cell

// Option is used to provide options for cells on a 2-D terminal.
type Option interface {
	// Set sets the provided option.
	Set(*Options)
}

// Options stores the provided options.
type Options struct {
	FgColor Color
	BgColor Color
}

// Set allows existing options to be passed as an option.
func (o *Options) Set(other *Options) {
	*other = *o
}

// NewOptions returns a new Options instance after applying the provided options.
func NewOptions(opts ...Option) *Options {
	o := &Options{}
	for _, opt := range opts {
		opt.Set(o)
	}
	return o
}

// option implements Option.
type option func(*Options)

// Set implements Option.set.
func (co option) Set(opts *Options) {
	co(opts)
}

// FgColor sets the foreground color of the cell.
func FgColor(color Color) Option {
	return option(func(co *Options) {
		co.FgColor = color
	})
}

// BgColor sets the background color of the cell.
func BgColor(color Color) Option {
	return option(func(co *Options) {
		co.BgColor = color
	})
}
