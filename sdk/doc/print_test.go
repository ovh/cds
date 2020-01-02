package doc

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const expected = `+++
title = "my section"
+++

##  ` + "`/test/third`" + `

URL         | **` + "`/test/third`" + `**
----------- |----------
Method      | 
Permissions | Auth: true
Code        | [](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+%22)


## My first

URL         | **` + "`/test/one`" + `**
----------- |----------
Method      | 
Query Parameter | one=1
Query Parameter | two=2
Permissions | Midd1: Value1,Value2 - Midd2: Value1,Value2 - Auth: true
Code        | [](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+%22)

### Description
my first desc
### Request Body
` + "```" + `
{"mykey": "myval"}
` + "```" + `
### Response Body
` + "```" + `
{"mykey": "myval"}
` + "```" + `

## My second

URL         | **` + "`/test/two`" + `**
----------- |----------
Method      | 
Query Parameter | one=1
Query Parameter | two=2
Permissions | Auth: true
Scopes | one, two
Code        | [](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+%22)

### Description
my second desc
### Request Body
` + "```" + `
{"mykey": "myval"}
` + "```" + `
### Response Body
` + "```" + `
{"mykey": "myval"}
` + "```" + `
`

func TestPrintSection(t *testing.T) {
	buf := new(bytes.Buffer)

	require.NoError(t, printSection("my section", []Doc{
		Doc{
			URL:          "/test/one",
			Title:        "My first",
			Description:  "my first desc",
			QueryParams:  []string{"one=1", "two=2"},
			RequestBody:  "{\"mykey\": \"myval\"}",
			ResponseBody: "{\"mykey\": \"myval\"}",
			Middlewares: []Middleware{
				Middleware{
					Name:  "Midd1",
					Value: []string{"Value1", "Value2"},
				},
				Middleware{
					Name:  "Midd2",
					Value: []string{"Value1", "Value2"},
				},
			},
		},
		{
			URL:          "/test/two",
			Title:        "My second",
			Description:  "my second desc",
			QueryParams:  []string{"one=1", "two=2"},
			RequestBody:  "{\"mykey\": \"myval\"}",
			ResponseBody: "{\"mykey\": \"myval\"}",
			Scopes:       []string{"one", "two"},
		},
		{
			URL: "/test/third",
		},
	}, buf))

	assert.Equal(t, expected, buf.String())
}
