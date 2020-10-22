package cdsclient

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (c *client) AuthDriverList() (sdk.AuthDriverResponse, error) {
	var response sdk.AuthDriverResponse
	if _, err := c.GetJSON(context.Background(), "/auth/driver", &response); err != nil {
		return response, err
	}
	return response, nil
}

func (c *client) AuthConsumerSignin(consumerType sdk.AuthConsumerType, request sdk.AuthConsumerSigninRequest) (sdk.AuthConsumerSigninResponse, error) {
	var res sdk.AuthConsumerSigninResponse
	_, _, _, err := c.RequestJSON(context.Background(), "POST", "/auth/consumer/"+string(consumerType)+"/signin", request, &res)
	return res, err
}

func (c *client) AuthConsumerSignout() error {
	_, _, _, err := c.RequestJSON(context.Background(), "POST", "/auth/consumer/signout", nil, nil)
	return err
}

func (c *client) AuthConsumerLocalSignup(request sdk.AuthConsumerSigninRequest) error {
	_, _, _, err := c.RequestJSON(context.Background(), "POST", "/auth/consumer/local/signup", request, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) AuthConsumerLocalSignupVerify(token, initToken string) (sdk.AuthConsumerSigninResponse, error) {
	var res sdk.AuthConsumerSigninResponse

	req := sdk.AuthConsumerSigninRequest{
		"token": token,
	}
	if initToken != "" {
		req["init_token"] = initToken
	}

	_, err := c.PostJSON(context.Background(), "/auth/consumer/local/verify", req, &res)
	if err != nil {
		return res, err
	}
	return res, nil
}

func (c *client) AuthConsumerListByUser(username string) (sdk.AuthConsumers, error) {
	var consumers sdk.AuthConsumers
	if _, err := c.GetJSON(context.Background(), "/user/"+username+"/auth/consumer", &consumers); err != nil {
		return nil, err
	}
	return consumers, nil
}

func (c *client) AuthConsumerDelete(username, id string) error {
	_, err := c.DeleteJSON(context.Background(), "/user/"+username+"/auth/consumer/"+id, nil)
	return err
}

func (c *client) AuthConsumerRegen(username, id string) (sdk.AuthConsumerCreateResponse, error) {
	var consumer sdk.AuthConsumerCreateResponse
	request := sdk.AuthConsumerRegenRequest{RevokeSessions: true}
	_, _, _, err := c.RequestJSON(context.Background(), "POST", "/user/"+username+"/auth/consumer/"+id+"/regen", request, &consumer)
	return consumer, err
}

func (c *client) AuthConsumerCreateForUser(username string, request sdk.AuthConsumer) (sdk.AuthConsumerCreateResponse, error) {
	var consumer sdk.AuthConsumerCreateResponse
	_, _, _, err := c.RequestJSON(context.Background(), "POST", "/user/"+username+"/auth/consumer", request, &consumer)
	return consumer, err
}

func (c *client) AuthSessionListByUser(username string) (sdk.AuthSessions, error) {
	var sessions sdk.AuthSessions
	if _, err := c.GetJSON(context.Background(), "/user/"+username+"/auth/session", &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

func (c *client) AuthSessionDelete(username, id string) error {
	_, err := c.DeleteJSON(context.Background(), "/user/"+username+"/auth/session/"+id, nil)
	return err
}

func (c *client) AuthMe() (sdk.AuthCurrentConsumerResponse, error) {
	var r sdk.AuthCurrentConsumerResponse
	_, err := c.GetJSON(context.Background(), "/auth/me", &r)
	return r, err
}

func (c *client) AuthSessionGet(id string) (sdk.AuthCurrentConsumerResponse, error) {
	var r sdk.AuthCurrentConsumerResponse
	_, err := c.GetJSON(context.Background(), "/auth/session/"+id, &r)
	return r, err
}

func (c *client) AuthConsumerLocalAskResetPassword(r sdk.AuthConsumerSigninRequest) error {
	_, err := c.PostJSON(context.Background(), "/auth/consumer/local/askReset", r, nil)
	return err
}

func (c *client) AuthConsumerLocalResetPassword(token, newPassword string) (sdk.AuthConsumerSigninResponse, error) {
	var res sdk.AuthConsumerSigninResponse
	_, _, _, err := c.RequestJSON(context.Background(), "POST", "/auth/consumer/local/reset",
		sdk.AuthConsumerSigninRequest{
			"token":    token,
			"password": newPassword,
		}, &res)
	if err != nil {
		return res, err
	}
	return res, nil
}
