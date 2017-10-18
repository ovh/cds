package bitbucket

import (
	"fmt"
	"net/url"
)

func (b *bitbucketClient) findByEmail(email string) (*User, error) {
	var users = UsersResponse{}
	var path = "/admin/users"
	if err := b.do("GET", "core", path, url.Values{"filter": []string{email}}, nil, &users); err != nil {
		return nil, err
	}
	if len(users.Values) >= 1 {
		return &users.Values[0], nil
	}
	return nil, fmt.Errorf("User not found")
}
