package api_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/ovh/cds/engine/api"
	"github.com/stretchr/testify/assert"
)

func TestQuerySort(t *testing.T) {
	url, _ := url.Parse("http://localhost?sort=column1,column2:asc,column3:desc")
	m, _ := api.QuerySort(&http.Request{URL: url})
	assert.Len(t, m, 3)
	assert.Equal(t, m, map[string]api.SortOrder{
		"column1": api.ASC,
		"column2": api.ASC,
		"column3": api.DESC,
	})
}
func TestQuerySortError(t *testing.T) {
	url, _ := url.Parse("http://localhost?sort=column1,,column3:desc")
	_, err := api.QuerySort(&http.Request{URL: url})
	assert.Error(t, err)

	url, _ = url.Parse("http://localhost?sort=column1,column3:unknown")
	_, err = api.QuerySort(&http.Request{URL: url})
	assert.Error(t, err)
}
