package bitbucket

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHooksConfig(t *testing.T) {
	a := `{
	"branchFilter2": "",
	"httpMethod": "POST",
	"httpMethod2": "GET",
	"locationCount": "2",
	"postContentType": "application/x-www-form-urlencoded",
	"postContentType2": "application/x-www-form-urlencoded",
	"postData": "",
	"skipSsl2": true,
	"tagFilter": "",
	"tagFilter2": "",
	"url": "aa" ,
	"url2": "http://foo.local",
	"userFilter": "",
	"userFilter2": "",
	"version": "3"
	}`
	h := HooksConfig{}
	err := json.Unmarshal([]byte(a), &h)
	assert.NoError(t, err)

	b, err := json.Marshal(h)
	assert.NoError(t, err)

	h2 := HooksConfig{}
	err = json.Unmarshal(b, &h2)
	assert.NoError(t, err)
	for _, d := range h.Details {
		if d.URL != "aa" && d.URL != "http://foo.local" {
			assert.Fail(t, "Unmarshal failed. Url should be aa or http://foo.local")
		}
	}
	for _, d := range h2.Details {
		if d.PostContentType != "application/x-www-form-urlencoded" {
			assert.Fail(t, "Unmarshal failed. PostContentType should be application/x-www-form-urlencoded")
		}
	}

}
