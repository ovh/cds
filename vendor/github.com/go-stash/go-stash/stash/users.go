package stash

import (
	"fmt"
	"net/url"
)

type UsersResponse struct {
	Values        []User `json:"values"`
	Size          int    `json:"size"`
	NextPageStart int    `json:"nextPageStart"`
	IsLastPage    bool   `json:"isLastPage"`
}

type User struct {
	Username     string `json:"name"`
	EmailAddress string `json:"emailAddress"`
	DisplayName  string `json:"displayName"`
	Slug         string `json:"slug"`
}

type UserResource struct {
	client *Client
}

// Get current user
func (u *UserResource) Current() (User, error) {
	var user = User{}
	var path = "username"

	if err := u.client.do("GET", "core", path, nil, nil, &user); err != nil {
		return user, err
	}

	return user, nil
}

//FindByEmail returns a user witch matching email address
func (u *UserResource) FindByEmail(email string) (*User, error) {
	var users = UsersResponse{}
	var path = "/admin/users"
	if err := u.client.do("GET", "core", path, url.Values{"filter": []string{email}}, nil, &users); err != nil {
		return nil, err
	}
	if len(users.Values) >= 1 {
		return &users.Values[0], nil
	}
	return nil, fmt.Errorf("User not found")
}
