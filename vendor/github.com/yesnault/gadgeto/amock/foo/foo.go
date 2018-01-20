package foo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

var Client = &http.Client{}

type Foo struct {
	Identifier string `json:"identifier"`
	BarCount   int    `json:"bar_count"`
}

func GetFoo(ident string) (*Foo, error) {
	resp, err := Client.Get("http://www.foo.com/foo/" + ident)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("got http error %d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ret := &Foo{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
