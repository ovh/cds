package gorpmapping

import (
	"fmt"
	"strings"
)

// IDsToQueryString returns a comma separated list of given ids.
func IDsToQueryString(ids []int64) string {
	res := make([]string, len(ids))
	for i := range ids {
		res[i] = fmt.Sprintf("%d", ids[i])
	}
	return strings.Join(res, ",")
}

// IDStringsToQueryString returns a comma separated list of given string ids.
func IDStringsToQueryString(ids []string) string {
	return strings.Join(ids, ",")
}
