package cdsclient

import (
	"context"
	"strconv"

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
	u, err := c.UserGet(username)
	if err != nil {
		return nil, err
	}

	var consumers []sdk.AuthConsumer
	if _, err := c.GetJSON(context.Background(), "/auth/consumer/"+strconv.FormatInt(u.ID, 10), &consumers); err != nil {
		return nil, err
	}

	return consumers, nil
}

func (c *client) AuthConsumerDelete(id string) error {
	_, err := c.DeleteJSON(context.Background(), "/auth/consumer/"+id, nil)
	return err
}

func (c *client) AuthConsumerCreate(request sdk.AuthConsumerRequest) (sdk.AuthConsumer, string, error) {
	var t sdk.AuthConsumer
	_, headers, _, err := c.RequestJSON(context.Background(), "POST", "/auth/consumer", request, &t)
	if err != nil {
		return t, "", err
	}
	jwt := headers.Get("X-CDS-JWT")
	return t, jwt, nil
}
