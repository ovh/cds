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
	//"log"
	"strings"
)

// Query creates a new Query Dsl
func Query() *QueryDsl {
	return &QueryDsl{}
}

/*

some ways to serialize
"query": {
	"filtered": {
	  "query": {
	    "query_string": {
	      "default_operator": "OR",
	      "default_field": "_all",
	      "query": " actor:\"bob\"  AND type:\"EventType\""
	    }
	  },
	  "filter": {
	    "range": {
	      "@timestamp": {
	        "from": "2012-12-29T16:52:48+00:00",
	        "to": "2012-12-29T17:52:48+00:00"
	      }
	    }
	  }
	}
},

"query" : {
    "term" : { "user" : "kimchy" }
}

"query" : {
    "match_all" : {}
},
*/
type QueryDsl struct {
	QueryEmbed
	FilterVal *FilterOp `json:"filter,omitempty"`
}

// The core Query Syntax can be embedded as a child of a variety of different parents
type QueryEmbed struct {
	MatchAll      *MatchAll              `json:"match_all,omitempty"`
	Terms         map[string]string      `json:"term,omitempty"`
	Qs            *QueryString           `json:"query_string,omitempty"`
	MultiMatch    *MultiMatch            `json:"multi_match,omitempty"`
	FunctionScore map[string]interface{} `json:"function_score,omitempty"`
	//Exist    string            `json:"_exists_,omitempty"`
}

// MarshalJSON provides custom marshalling to support the query dsl which is a conditional
// json format, not always the same parent/children
func (qd *QueryDsl) MarshalJSON() ([]byte, error) {
	q := qd.QueryEmbed
	hasQuery := false
	if q.Qs != nil || len(q.Terms) > 0 || q.MatchAll != nil || q.MultiMatch != nil {
		hasQuery = true
	}
	// If a query has a
	if qd.FilterVal != nil && hasQuery {
		queryB, err := json.Marshal(q)
		if err != nil {
			return queryB, err
		}
		filterB, err := json.Marshal(qd.FilterVal)
		if err != nil {
			return filterB, err
		}
		return []byte(fmt.Sprintf(`{"filtered":{"query":%s,"filter":%s}}`, queryB, filterB)), nil
	}
	return json.Marshal(q)
}

// get all
func (q *QueryDsl) All() *QueryDsl {
	q.MatchAll = &MatchAll{""}
	return q
}

// Range adds a RANGE FilterOp to the search query
// Legacy. Use the Filter() function instead
func (q *QueryDsl) Range(fop *FilterOp) *QueryDsl {
	if q.FilterVal == nil {
		q.FilterVal = fop
		return q
	}

	return q
}

// Add a term search for a specific field
//    Term("user","kimchy")
func (q *QueryDsl) Term(name, value string) *QueryDsl {
	if len(q.Terms) == 0 {
		q.Terms = make(map[string]string)
	}
	q.Terms[name] = value
	return q
}

// FunctionScore sets functions to use to score the documents.
// http://www.elastic.co/guide/en/elasticsearch/reference/1.x/query-dsl-function-score-query.html
func (q *QueryDsl) FunctionScore(mode string, functions ...map[string]interface{}) *QueryDsl {
	q.QueryEmbed.FunctionScore = map[string]interface{}{
		"functions":  functions,
		"score_mode": mode,
	}
	return q
}

// The raw search strings (lucene valid)
func (q *QueryDsl) Search(searchFor string) *QueryDsl {
	//I don't think this is right, it is not a filter.query, it should be q query?
	qs := NewQueryString("", "")
	q.QueryEmbed.Qs = &qs
	q.QueryEmbed.Qs.Query = searchFor
	return q
}

// Querystring operations
func (q *QueryDsl) Qs(qs *QueryString) *QueryDsl {
	q.QueryEmbed.Qs = qs
	return q
}

// SetLenient sets whether the query should ignore format based failures,
// such as passing in text to a number field.
func (q *QueryDsl) SetLenient(lenient bool) *QueryDsl {
	q.QueryEmbed.Qs.Lenient = lenient
	return q
}

// Fields in query_string search
//     Fields("fieldname","search_for","","")
//
//     Fields("fieldname,field2,field3","search_for","","")
//
//     Fields("fieldname,field2,field3","search_for","field_exists","")
func (q *QueryDsl) Fields(fields, search, exists, missing string) *QueryDsl {
	fieldList := strings.Split(fields, ",")
	qs := NewQueryString("", "")
	q.QueryEmbed.Qs = &qs
	q.QueryEmbed.Qs.Query = search
	if len(fieldList) == 1 {
		q.QueryEmbed.Qs.DefaultField = fields
	} else {
		q.QueryEmbed.Qs.Fields = fieldList
	}
	q.QueryEmbed.Qs.Exists = exists
	q.QueryEmbed.Qs.Missing = missing
	return q
}

// Filter this query
func (q *QueryDsl) Filter(f *FilterOp) *QueryDsl {
	q.FilterVal = f
	return q
}

// MultiMatch allows searching against multiple fields.
func (q *QueryDsl) MultiMatch(s string, fields []string) *QueryDsl {
	q.QueryEmbed.MultiMatch = &MultiMatch{Query: s, Fields: fields}
	return q
}

type MultiMatch struct {
	Query  string   `json:"query"`
	Fields []string `json:"fields"`
}

type MatchAll struct {
	All string `json:"-"`
}

// should we reuse QueryDsl here?
type QueryWrap struct {
	Qs QueryString `json:"query_string,omitempty"`
}

// QueryString based search
func NewQueryString(field, query string) QueryString {
	return QueryString{"", field, query, "", "", nil, false}
}

type QueryString struct {
	DefaultOperator string   `json:"default_operator,omitempty"`
	DefaultField    string   `json:"default_field,omitempty"`
	Query           string   `json:"query,omitempty"`
	Exists          string   `json:"_exists_,omitempty"`
	Missing         string   `json:"_missing_,omitempty"`
	Fields          []string `json:"fields,omitempty"`
	Lenient         bool     `json:"lenient,omitempty"`
	//_exists_:field1,
	//_missing_:field1,
}

//I don't know how any of the Term stuff below is supposed to work. -mikeyoon

// Generic Term based (used in query, facet, filter)
type Term struct {
	Terms     Terms       `json:"terms,omitempty"`
	FilterVal *FilterWrap `json:"facet_filter,omitempty"`
}

type Terms struct {
	Fields []string `json:"field,omitempty"`
	Size   string   `json:"size,omitempty"`
	Regex  string   `json:"regex,omitempty"`
}

func NewTerm(fields ...string) *Term {
	m := &Term{Terms{Fields: fields}, nil}
	return m
}

func (s *Term) Filter(fl ...interface{}) *Term {
	if s.FilterVal == nil {
		s.FilterVal = NewFilterWrap()
	}

	s.FilterVal.addFilters(fl)
	return s
}

// Custom marshalling
func (t *Terms) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	// TODO:  this isn't getting called!?
	if len(t.Fields) == 1 {
		m["field"] = t.Fields[0]
	} else if len(t.Fields) > 1 {
		m["fields"] = t.Fields
	}
	if len(t.Regex) > 0 {
		m["regex"] = t.Regex
	}
	if len(t.Size) > 0 {
		m["size"] = t.Size
	}
	return json.Marshal(m)
}
