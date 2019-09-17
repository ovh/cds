// +build !skipkeychaintests

package internal

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStoreContextWithNoContent(t *testing.T) {
	rdata := bytes.NewReader([]byte(``))

	cdsContext := CDSContext{
		Context:               "TestStoreContext",
		Host:                  "http://localhost:8080/test",
		InsecureSkipVerifyTLS: true,
		Session:               "the-token-test",
		Token:                 "the-token",
	}

	wdata := &bytes.Buffer{}
	err := StoreContext(rdata, wdata, cdsContext)
	assert.NoError(t, err)

	t.Log(wdata.String())
	rdata2 := bytes.NewReader(wdata.Bytes())

	cdsContextRead, err := GetContext(rdata2, cdsContext.Context)
	assert.NoError(t, err)
	assert.NotNil(t, cdsContextRead)
	assert.Equal(t, cdsContext.Token, cdsContextRead.Token)
	assert.Equal(t, cdsContext.Session, cdsContextRead.Session)
	assert.Equal(t, cdsContext.InsecureSkipVerifyTLS, cdsContextRead.InsecureSkipVerifyTLS)
}

func TestStoreContextWithExistingContent(t *testing.T) {
	rdata := bytes.NewReader([]byte(`current = "TestStoreContext"

[TestStoreContext]
host = "http://localhost:8080/test"
InsecureSkipVerifyTLS = "false"
Session = "the-token-test"
Token = "the-token"`))

	cdsContext := CDSContext{
		Context:               "TestStoreContext",
		Host:                  "http://localhost:8080/test",
		InsecureSkipVerifyTLS: false,
		Session:               "the-token-test",
		Token:                 "the-token",
	}

	wdata := &bytes.Buffer{}
	err := StoreContext(rdata, wdata, cdsContext)
	assert.NoError(t, err)

	t.Log(wdata.String())
	rdata2 := bytes.NewReader(wdata.Bytes())

	cdsContextRead, err := GetContext(rdata2, cdsContext.Context)
	assert.NoError(t, err)
	assert.NotNil(t, cdsContextRead)
	assert.Equal(t, cdsContext.Token, cdsContextRead.Token)
	assert.Equal(t, cdsContext.Session, cdsContextRead.Session)
}

func TestStoreContextWithTwoContexts(t *testing.T) {
	rdata := bytes.NewReader([]byte(`current = "TestStoreContext"

[TestStoreContext]
host = "http://localhost:8080/test"
InsecureSkipVerifyTLS = false
Session = "the-token-test"
Token = "the-token"`))

	cdsContext2 := CDSContext{
		Context:               "TestStoreContext2",
		Host:                  "http://localhost:8080/test2",
		InsecureSkipVerifyTLS: false,
		Session:               "the-token-test2",
		Token:                 "the-token",
	}

	wdata := &bytes.Buffer{}
	err := StoreContext(rdata, wdata, cdsContext2)
	assert.NoError(t, err)

	content := wdata.String()
	t.Log(content)

	wdata2 := &bytes.Buffer{}
	wdata2.WriteString(content)
	wdata3 := &bytes.Buffer{}
	wdata3.WriteString(content)

	cdsContextRead2, err := GetContext(wdata3, cdsContext2.Context)
	assert.NoError(t, err)
	assert.NotNil(t, cdsContextRead2)
	assert.Equal(t, cdsContext2.Token, cdsContextRead2.Token)
	assert.Equal(t, cdsContext2.Session, cdsContextRead2.Session)
	assert.Equal(t, cdsContext2.InsecureSkipVerifyTLS, cdsContextRead2.InsecureSkipVerifyTLS)

	wdata4 := &bytes.Buffer{}
	wdata4.WriteString(content)
	cdsContextReadCurrent, err := GetCurrentContext(wdata4)
	assert.NoError(t, err)
	assert.NotNil(t, cdsContextReadCurrent)
	assert.Equal(t, cdsContextRead2, cdsContextReadCurrent)

	wdata5 := &bytes.Buffer{}
	wdata5.WriteString(content)
	cdsContextReadCurrentName, err := GetCurrentContextName(wdata5)
	assert.NoError(t, err)
	assert.Equal(t, cdsContextRead2.Context, cdsContextReadCurrentName)

	t.Log(content)
	reader := bytes.NewReader([]byte(content))

	writer := &bytes.Buffer{}
	err = SetCurrentContext(reader, writer, "TestStoreContext")
	assert.NoError(t, err)
}
