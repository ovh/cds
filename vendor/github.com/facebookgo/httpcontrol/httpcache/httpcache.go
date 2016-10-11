// Package httpcache provides a cache enabled http Transport.
package httpcache

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"
)

type ByteCache interface {
	Store(key string, value []byte, timeout time.Duration) error
	Get(key string) ([]byte, error)
}

type Config interface {
	// Generates the cache key for the given http.Request. An empty string will
	// disable caching.
	Key(req *http.Request) string

	// Provides the max cache age for the given request/response pair. A zero
	// value will disable caching for the pair. The request is available via
	// res.Request.
	MaxAge(res *http.Response) time.Duration
}

// Cache enabled http.Transport.
type Transport struct {
	Config    Config            // Provides cache key & timeout logic.
	ByteCache ByteCache         // Cache where serialized responses will be stored.
	Transport http.RoundTripper // The underlying http.RoundTripper for actual requests.
}

type cacheEntry struct {
	Response *http.Response
	Body     []byte
}

// A cache enabled RoundTrip.
func (t *Transport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	key := t.Config.Key(req)
	var entry cacheEntry

	// from cache
	if key != "" {
		raw, err := t.ByteCache.Get(key)
		if err != nil {
			return nil, err
		}

		if raw != nil {
			if err = json.Unmarshal(raw, &entry); err != nil {
				return nil, err
			}

			// setup fake http.Response
			res = entry.Response
			res.Body = ioutil.NopCloser(bytes.NewReader(entry.Body))
			res.Request = req
			return res, nil
		}
	}

	// real request
	res, err = t.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// no caching required
	if key == "" {
		return res, nil
	}

	// fully buffer response for caching purposes
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}

	// remove properties we want to skip in serialization
	res.Body = nil
	res.Request = nil

	// serialize the cache entry
	entry.Response = res
	entry.Body = body
	raw, err := json.Marshal(&entry)
	if err != nil {
		return nil, err
	}

	// put back non serialized properties
	res.Body = ioutil.NopCloser(bytes.NewReader(body))
	res.Request = req

	// determine timeout & put it in cache
	timeout := t.Config.MaxAge(res)
	if timeout != 0 {
		if err = t.ByteCache.Store(key, raw, timeout); err != nil {
			return nil, err
		}
	}

	// reset body in case the config.Timeout logic consumed it
	res.Body = ioutil.NopCloser(bytes.NewReader(body))
	return res, nil
}

type cacheByPath time.Duration

func (c cacheByPath) Key(req *http.Request) string {
	if req.Method != "GET" && req.Method != "HEAD" {
		return ""
	}
	return req.URL.Host + "/" + req.URL.Path
}

func (c cacheByPath) MaxAge(res *http.Response) time.Duration {
	return time.Duration(c)
}

// This caches against the host + path (ignoring scheme, auth, query etc) for
// the specified duration.
func CacheByPath(timeout time.Duration) Config {
	return cacheByPath(timeout)
}

type cacheByURL time.Duration

func (c cacheByURL) Key(req *http.Request) string {
	if req.Method != "GET" && req.Method != "HEAD" {
		return ""
	}
	return req.URL.String()
}

func (c cacheByURL) MaxAge(res *http.Response) time.Duration {
	return time.Duration(c)
}

// This caches against the entire URL for the specified duration.
func CacheByURL(timeout time.Duration) Config {
	return cacheByURL(timeout)
}
