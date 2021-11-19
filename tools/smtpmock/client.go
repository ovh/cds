package smtpmock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

func NewClient(url string) Client {
	return &client{url: url}
}

type Client interface {
	Signin(token string) (SigninResponse, error)
	GetMessages() ([]Message, error)
	GetRecipientMessages(email string) ([]Message, error)
	GetRecipientLatestMessage(email string) (Message, error)
}

type client struct {
	url          string
	sessionToken string
}

func (c *client) requestJSON(method string, url string, body io.Reader, data interface{}) error {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return errors.WithStack(err)
	}

	req.Header.Add("Content-Type", "application/json")

	if c.sessionToken != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.sessionToken))
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	if res.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("error request at %s %s", method, req.URL.String()))
	}

	buf, err := io.ReadAll(res.Body)
	if err != nil {
		return errors.WithStack(err)
	}

	return errors.WithStack(json.Unmarshal(buf, data))
}

func (c *client) Signin(token string) (SigninResponse, error) {
	var res SigninResponse

	buf, err := json.Marshal(SigninRequest{
		SigninToken: token,
	})
	if err != nil {
		return res, errors.WithStack(err)
	}

	if err := c.requestJSON(http.MethodPost, c.url+"/signin", bytes.NewBuffer(buf), &res); err != nil {
		return res, err
	}
	c.sessionToken = res.SessionToken
	return res, nil
}

func (c *client) GetMessages() ([]Message, error) {
	var res []Message
	if err := c.requestJSON(http.MethodGet, c.url+"/messages", nil, &res); err != nil {
		return res, err
	}
	return res, nil
}

func (c *client) GetRecipientMessages(email string) ([]Message, error) {
	var res []Message
	if err := c.requestJSON(http.MethodGet, fmt.Sprintf("%s/messages/%s", c.url, email), nil, &res); err != nil {
		return res, err
	}
	return res, nil
}

func (c *client) GetRecipientLatestMessage(email string) (Message, error) {
	var res Message
	if err := c.requestJSON(http.MethodGet, fmt.Sprintf("%s/messages/%s/latest", c.url, email), nil, &res); err != nil {
		return res, err
	}
	return res, nil
}
