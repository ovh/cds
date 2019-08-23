package cdsclient

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (c *client) AuthConsumerSignin(consumerType sdk.AuthConsumerType, request sdk.AuthConsumerSigninRequest) (*sdk.AuthConsumerSigninResponse, error) {
	var res sdk.AuthConsumerSigninResponse
	_, _, _, err := c.RequestJSON(context.Background(), "POST", "/auth/consumer/"+string(consumerType)+"/signin", request, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *client) AuthConsumerListByUser(username string) ([]sdk.AuthConsumer, error) {
	var consumers []sdk.AuthConsumer
	if _, err := c.GetJSON(context.Background(), "/user/"+username+"/auth/consumer", &consumers); err != nil {
		return nil, err
	}
	return consumers, nil
}

func (c *client) AuthConsumerDelete(username, id string) error {
	_, err := c.DeleteJSON(context.Background(), "/user/"+username+"/auth/consumer/"+id, nil)
	return err
}

func (c *client) AuthConsumerCreateForUser(username string, request sdk.AuthConsumer) (sdk.AuthConsumerCreateResponse, error) {
	var consumer sdk.AuthConsumerCreateResponse
	_, _, _, err := c.RequestJSON(context.Background(), "POST", "/user/"+username+"/auth/consumer", request, &consumer)
	if err != nil {
		return consumer, err
	}
	return consumer, nil
}
