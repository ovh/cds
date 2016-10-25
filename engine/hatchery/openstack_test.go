package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/facebookgo/httpcontrol"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestCreateServer(t *testing.T) {
	client = &http.Client{
		Transport: &httpcontrol.Transport{
			RequestTimeout: 10 * time.Second,
			MaxTries:       5,
		},
	}

	router := mux.NewRouter()
	router.HandleFunc("/TestCreateServer/servers", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NotNil(t, body)
		t.Log(string(body))

		assert.Equal(t, `{"server":{"id":"","name":"test","imageRef":"image","flavorRef":"flavor","user_data":"blabla\u0008labla","metadata":{"worker":"test"},"networks":[{"uuid":"network","fixed_ip":"192.168.0.1"}],"links":null,"status":"","key_name":"","addresses":null,"updated":"","personality":[{"path":"/worker","contents":"dGhpcyBpcyB0aGUgY29udGVudA=="}]}}`, string(body))

	})
	http.Handle("/TestCreateServer/", router)

	s := httptest.NewServer(router)
	defer s.Close()
	w := httptest.NewRecorder()

	p := []*File{
		&File{
			Path:     "/worker",
			Contents: []byte("this is the content"),
		},
	}

	err := createServer(s.URL+"/TestCreateServer", "", "test", "image", "flavor", "network", "192.168.0.1", "blabla\blabla", p)
	assert.NoError(t, err)

	assert.NotZero(t, w.Code)

}
