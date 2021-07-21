package bitbucketserver

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// -----------------------------------------------------------------------------
// Private Helper Functions

var nonceCounter uint64
var hostname string

func init() {
	hostname, _ = os.Hostname()

	if err := binary.Read(rand.Reader, binary.BigEndian, &nonceCounter); err != nil {
		nonceCounter = uint64(time.Now().UnixNano()) // fallback
	}
}

// nonce returns a unique string.
func nonce() string {
	return hostname + strconv.FormatUint(atomic.AddUint64(&nonceCounter, 1), 16)
}

// Timestamp generates a timestamp, expressed in the number of seconds
// since January 1, 1970 00:00:00 GMT.
func timestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

func escape(s string) string {
	t := make([]byte, 0, 3*len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isEscapable(c) {
			t = append(t, '%')
			t = append(t, "0123456789ABCDEF"[c>>4])
			t = append(t, "0123456789ABCDEF"[c&15])
		} else {
			t = append(t, s[i])
		}
	}
	return string(t)
}

func isEscapable(b byte) bool {
	return !('A' <= b && b <= 'Z' || 'a' <= b && b <= 'z' || '0' <= b && b <= '9' || b == '-' || b == '.' || b == '_' || b == '~')

}

func authorizationString(params map[string]string) string {

	// loop through params, add keys to map
	var keys []string
	for key := range params {
		keys = append(keys, key)
	}

	// sort the array of header keys
	sort.StringSlice(keys).Sort()

	// create the signed string
	var str string
	var cnt = 0

	// loop through sorted params and append to the string
	for _, key := range keys {

		// we previously encoded all params (url params, form data & oauth params)
		// but for the authorization string we should only encode the oauth params
		if !strings.HasPrefix(key, "oauth_") {
			continue
		}

		if cnt > 0 {
			str += ","
		}

		str += fmt.Sprintf("%s=%q", key, escape(params[key]))
		cnt++
	}

	return fmt.Sprintf("OAuth %s", str)
}
