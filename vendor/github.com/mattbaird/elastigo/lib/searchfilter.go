// Copyright 2013 Matthew Baird
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package elastigo

import (
	"encoding/json"
	"fmt"
	"github.com/araddon/gou"
)

var (
	_ = gou.DEBUG
)

// BoolClause represents aa bool (and/or) clause for use with FilterWrap
// Legacy, use new FilterOp functions instead
type BoolClause string

// TermExecutionMode refers to how a terms (not term) filter should behave
// The acceptable options are all prefixed with TEM
// See https://www.elastic.co/guide/en/elasticsearch/reference/1.5/query-dsl-terms-filter.html
type TermExecutionMode string

const (
	// TEMDefault default ES term filter behavior (plain)
	TEMDefault TermExecutionMode = ""
	// TEMPlain default ES term filter behavior
	TEMPlain TermExecutionMode = "plain"
	// TEMField field_data execution mode
	TEMField TermExecutionMode = "field_data"
	// TEMBool bool execution mode
	TEMBool TermExecutionMode = "bool"
	// TEMAnd and execution mode
	TEMAnd TermExecutionMode = "and"
	// TEMOr or execution mode
	TEMOr TermExecutionMode = "or"
)

// FilterClause is either a boolClause or FilterOp for use with FilterWrap
type FilterClause interface {
	String() string
}

// FilterWrap is the legacy struct for chaining multiple filters with a bool
// Legacy, use new FilterOp functions instead
type FilterWrap struct {
	boolClause string
	filters    []interface{}
}

// NewFilterWrap creates a new FilterWrap struct
func NewFilterWrap() *FilterWrap {
	return &FilterWrap{filters: make([]interface{}, 0), boolClause: "and"}
}

func (f *FilterWrap) String() string {
	return fmt.Sprintf(`fopv: %d:%v`, len(f.filters), f.filters)
}

// Bool sets the type of boolean filter to use.
// Accepted values are "and" and "or".
// Legacy, use new FilterOp functions instead
func (f *FilterWrap) Bool(s string) {
	f.boolClause = s
}

// Custom marshalling to support the query dsl
func (f *FilterWrap) addFilters(fl []interface{}) {
	if len(fl) > 1 {
		fc := fl[0]
		switch fc.(type) {
		case BoolClause, string:
			f.boolClause = fc.(string)
			fl = fl[1:]
		}
	}
	f.filters = append(f.filters, fl...)
}

// MarshalJSON override for FilterWrap to match the expected ES syntax with the bool at the root
func (f *FilterWrap) MarshalJSON() ([]byte, error) {
	var root interface{}
	if len(f.filters) > 1 {
		root = map[string]interface{}{f.boolClause: f.filters}
	} else if len(f.filters) == 1 {
		root = f.filters[0]
	}
	return json.Marshal(root)
}

/*
	"filter": {
		"range": {
		  "@timestamp": {
		    "from": "2012-12-29T16:52:48+00:00",
		    "to": "2012-12-29T17:52:48+00:00"
		  }
		}
	}
	"filter": {
	    "missing": {
	        "field": "repository.name"
	    }
	}

	"filter" : {
	    "terms" : {
	        "user" : ["kimchy", "elasticsearch"],
	        "execution" : "bool",
	        "_cache": true
	    }
	}

	"filter" : {
	    "term" : { "user" : "kimchy"}
	}

	"filter" : {
	    "and" : [
	        {
	            "range" : {
	                "postDate" : {
	                    "from" : "2010-03-01",
	                    "to" : "2010-04-01"
	                }
	            }
	        },
	        {
	            "prefix" : { "name.second" : "ba" }
	        }
	    ]
	}

*/

// Filter creates a blank FilterOp that can be customized with further function calls
// This is the starting point for constructing any filter query
// Examples:
//
//   Filter().Term("user","kimchy")
//
//   // we use variadics to allow n arguments, first is the "field" rest are values
//   Filter().Terms("user", "kimchy", "elasticsearch")
//
//   Filter().Exists("repository.name")
func Filter() *FilterOp {
	return &FilterOp{}
}

// CompoundFilter creates a complete FilterWrap given multiple filters
// Legacy, use new FilterOp functions instead
func CompoundFilter(fl ...interface{}) *FilterWrap {
	FilterVal := NewFilterWrap()
	FilterVal.addFilters(fl)
	return FilterVal
}

// FilterOp holds all the information for a filter query
// Properties should not be set directly, but instead via the fluent-style API.
type FilterOp struct {
	TermsMap        map[string]interface{} `json:"terms,omitempty"`
	TermMap         map[string]interface{} `json:"term,omitempty"`
	RangeMap        map[string]RangeFilter `json:"range,omitempty"`
	ExistsProp      *propertyPathMarker    `json:"exists,omitempty"`
	MissingProp     *propertyPathMarker    `json:"missing,omitempty"`
	AndFilters      []*FilterOp            `json:"and,omitempty"`
	OrFilters       []*FilterOp            `json:"or,omitempty"`
	NotFilters      []*FilterOp            `json:"not,omitempty"`
	LimitProp       *LimitFilter           `json:"limit,omitempty"`
	TypeProp        *TypeFilter            `json:"type,omitempty"`
	IdsProp         *IdsFilter             `json:"ids,omitempty"`
	ScriptProp      *ScriptFilter          `json:"script,omitempty"`
	GeoDistMap      map[string]interface{} `json:"geo_distance,omitempty"`
	GeoDistRangeMap map[string]interface{} `json:"geo_distance_range,omitempty"`
}

type propertyPathMarker struct {
	Field string `json:"field"`
}

// LimitFilter holds the Limit filter information
// Value: number of documents to limit
type LimitFilter struct {
	Value int `json:"value"`
}

// TypeFilter filters on the document type
// Value: the document type to filter
type TypeFilter struct {
	Value string `json:"value"`
}

// IdsFilter holds the type and ids (on the _id field) to filter
// Type: a string or an array of string types. Optional.
// Values: Array of ids to match
type IdsFilter struct {
	Type   []string      `json:"type,omitempty"`
	Values []interface{} `json:"values,omitempty"`
}

// ScriptFilter will filter using a custom javascript function
// Script: the javascript to run
// Params: map of custom parameters to pass into the function (JSON), if any
// IsCached: whether to cache the results of the filter
type ScriptFilter struct {
	Script   string                 `json:"script"`
	Params   map[string]interface{} `json:"params,omitempty"`
	IsCached bool                   `json:"_cache,omitempty"`
}

// RangeFilter filters given a range. Parameters need to be comparable for ES to accept.
// Only a minimum of one comparison parameter is required. You probably shouldn't mix GT and GTE parameters.
// Gte: the greater-than-or-equal to value. Should be a number or date.
// Lte: the less-than-or-equal to value. Should be a number or date.
// Gt: the greater-than value. Should be a number or date.
// Lt: the less-than value. Should be a number or date.
// TimeZone: the timezone to use (+|-h:mm format), if the other parameters are dates
type RangeFilter struct {
	Gte      interface{} `json:"gte,omitempty"`
	Lte      interface{} `json:"lte,omitempty"`
	Gt       interface{} `json:"gt,omitempty"`
	Lt       interface{} `json:"lt,omitempty"`
	TimeZone string      `json:"time_zone,omitempty"` //Ideally this would be an int
}

// GeoLocation holds the coordinates for a geo query. Currently hashes are not supported.
type GeoLocation struct {
	Latitude  float32 `json:"lat"`
	Longitude float32 `json:"lon"`
}

// GeoField holds a GeoLocation and a field to match to.
// This exists so the struct will match the ES schema.
type GeoField struct {
	GeoLocation
	Field string
}

// Term will add a term to the filter.
// Multiple Term filters can be added, and ES will OR them.
// If the term already exists in the FilterOp, the value will be overridden.
func (f *FilterOp) Term(field string, value interface{}) *FilterOp {
	if len(f.TermMap) == 0 {
		f.TermMap = make(map[string]interface{})
	}

	f.TermMap[field] = value
	return f
}

// And will add an AND op to the filter. One or more FilterOps can be passed in.
func (f *FilterOp) And(filters ...*FilterOp) *FilterOp {
	if len(f.AndFilters) == 0 {
		f.AndFilters = filters[:]
	} else {
		f.AndFilters = append(f.AndFilters, filters...)
	}

	return f
}

// Or will add an OR op to the filter. One or more FilterOps can be passed in.
func (f *FilterOp) Or(filters ...*FilterOp) *FilterOp {
	if len(f.OrFilters) == 0 {
		f.OrFilters = filters[:]
	} else {
		f.OrFilters = append(f.OrFilters, filters...)
	}

	return f
}

// Not will add a NOT op to the filter. One or more FilterOps can be passed in.
func (f *FilterOp) Not(filters ...*FilterOp) *FilterOp {
	if len(f.NotFilters) == 0 {
		f.NotFilters = filters[:]

	} else {
		f.NotFilters = append(f.NotFilters, filters...)
	}

	return f
}

// GeoDistance will add a GEO DISTANCE op to the filter.
// distance: distance in ES distance format, i.e. "100km" or "100mi".
// fields: an array of GeoField origin coordinates. Only one coordinate needs to match.
func (f *FilterOp) GeoDistance(distance string, fields ...GeoField) *FilterOp {
	f.GeoDistMap = make(map[string]interface{})
	f.GeoDistMap["distance"] = distance
	for _, val := range fields {
		f.GeoDistMap[val.Field] = val.GeoLocation
	}

	return f
}

// GeoDistanceRange will add a GEO DISTANCE RANGE op to the filter.
// from: minimum distance in ES distance format, i.e. "100km" or "100mi".
// to: maximum distance in ES distance format, i.e. "100km" or "100mi".
// fields: an array of GeoField origin coordinates. Only one coor
func (f *FilterOp) GeoDistanceRange(from string, to string, fields ...GeoField) *FilterOp {
	f.GeoDistRangeMap = make(map[string]interface{})
	f.GeoDistRangeMap["from"] = from
	f.GeoDistRangeMap["to"] = to

	for _, val := range fields {
		f.GeoDistRangeMap[val.Field] = val.GeoLocation
	}

	return f
}

// NewGeoField is a helper function to create values for the GeoDistance filters
func NewGeoField(field string, latitude float32, longitude float32) GeoField {
	return GeoField{
		GeoLocation: GeoLocation{Latitude: latitude, Longitude: longitude},
		Field:       field}
}

// Terms adds a TERMS op to the filter.
// field: the document field
// executionMode Term execution mode, starts with TEM
// values: array of values to match
// Note: you can only have one terms clause in a filter. Use a bool filter to combine multiple.
func (f *FilterOp) Terms(field string, executionMode TermExecutionMode, values ...interface{}) *FilterOp {
	//You can only have one terms in a filter
	f.TermsMap = make(map[string]interface{})

	if executionMode != "" {
		f.TermsMap["execution"] = executionMode
	}

	f.TermsMap[field] = values

	return f
}

// Range adds a range filter for the given field.
// See the RangeFilter struct documentation for information about the parameters.
func (f *FilterOp) Range(field string, gte interface{},
	gt interface{}, lte interface{}, lt interface{}, timeZone string) *FilterOp {

	if f.RangeMap == nil {
		f.RangeMap = make(map[string]RangeFilter)
	}

	f.RangeMap[field] = RangeFilter{
		Gte:      gte,
		Gt:       gt,
		Lte:      lte,
		Lt:       lt,
		TimeZone: timeZone}

	return f
}

// Type adds a TYPE op to the filter.
func (f *FilterOp) Type(fieldType string) *FilterOp {
	f.TypeProp = &TypeFilter{Value: fieldType}
	return f
}

// Ids adds a IDS op to the filter.
func (f *FilterOp) Ids(ids ...interface{}) *FilterOp {
	f.IdsProp = &IdsFilter{Values: ids}
	return f
}

// IdsByTypes adds a IDS op to the filter, but also allows passing in an array of types for the query.
func (f *FilterOp) IdsByTypes(types []string, ids ...interface{}) *FilterOp {
	f.IdsProp = &IdsFilter{Type: types, Values: ids}
	return f
}

// Exists adds an EXISTS op to the filter.
func (f *FilterOp) Exists(field string) *FilterOp {
	f.ExistsProp = &propertyPathMarker{Field: field}
	return f
}

// Missing adds an MISSING op to the filter.
func (f *FilterOp) Missing(field string) *FilterOp {
	f.MissingProp = &propertyPathMarker{Field: field}
	return f
}

// Limit adds an LIMIT op to the filter.
func (f *FilterOp) Limit(maxResults int) *FilterOp {
	f.LimitProp = &LimitFilter{Value: maxResults}
	return f
}
