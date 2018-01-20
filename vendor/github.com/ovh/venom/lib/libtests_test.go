package venom_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/venom/executors/http"
	"github.com/ovh/venom/lib"
)

func TestVenomTestCase(t *testing.T) {
	v := venom.TestCase(t, "TestVenomTestCase", venom.V{
		"foo": "bar",
	})
	r := v.Do(venom.P{
		"type":   "exec",
		"script": "echo {{.foo}}",
	})
	assert.Equal(t, "bar", r["result.systemout"])

	r = v.Do(
		venom.HTTP.Get("https://www.google.com", "").
			WithHeaders(http.Headers{"test": "test"}))

	assert.Equal(t, 200, r["result.statuscode"])

}
