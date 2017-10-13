package bitbucket

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/facebookgo/httpcontrol"
)

var (
	httpClient = &http.Client{
		Transport: &httpcontrol.Transport{
			RequestTimeout: time.Second * 30,
			MaxTries:       5,
		},
	}
)

func requestString(method string, uri string, params map[string]string) string {

	// loop through params, add keys to map
	var keys []string
	for key, _ := range params {
		keys = append(keys, key)
	}

	// sort the array of header keys
	sort.StringSlice(keys).Sort()

	// create the signed string
	result := method + "&" + escape(uri)

	// loop through sorted params and append to the string
	for pos, key := range keys {
		if pos == 0 {
			result += "&"
		} else {
			result += escape("&")
		}

		result += escape(fmt.Sprintf("%s=%s", key, escape(params[key])))
	}

	return result
}
