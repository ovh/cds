package test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ovh/cds/sdk"
)

// AuthentifyRequestFromWorker have to be used only for tests
func AuthentifyRequestFromWorker(t *testing.T, req *http.Request, w *sdk.Worker) {
	req.Header.Add(sdk.AuthHeader, base64.StdEncoding.EncodeToString([]byte(w.ID)))
}

// NewAuthentifiedRequestFromWorker prepare a request
func NewAuthentifiedRequestFromWorker(t *testing.T, w *sdk.Worker, method, uri string, i interface{}) *http.Request {
	var btes []byte
	var err error
	if i != nil {
		btes, err = json.Marshal(i)
		if err != nil {
			t.FailNow()
		}
	}

	req, err := http.NewRequest(method, uri, bytes.NewBuffer(btes))
	if err != nil {
		t.FailNow()
	}

	AuthentifyRequestFromWorker(t, req, w)

	return req
}

func AuthHeaders(t *testing.T, u *sdk.User, pass string) http.Header {
	h := http.Header{}
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(u.Username+":"+pass))
	h.Add("Authorization", auth)
	return h
}

// AuthentifyRequest  have to be used only for tests
func AuthentifyRequest(t *testing.T, req *http.Request, u *sdk.User, pass string) {
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(u.Username+":"+pass))
	req.Header.Add("Authorization", auth)
}

//NewAuthentifiedRequest prepare a request
func NewAuthentifiedRequest(t *testing.T, u *sdk.User, pass, method, uri string, i interface{}) *http.Request {
	var btes []byte
	var err error
	if i != nil {
		btes, err = json.Marshal(i)
		if err != nil {
			t.FailNow()
		}
	}

	req, err := http.NewRequest(method, uri, bytes.NewBuffer(btes))
	if err != nil {
		t.FailNow()
	}
	AuthentifyRequest(t, req, u, pass)

	return req
}
