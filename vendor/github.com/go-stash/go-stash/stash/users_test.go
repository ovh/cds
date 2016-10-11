package stash

import (
	"testing"
)

func TestUserCurrent(t *testing.T) {
	user, err := client.Users.Current()
	if err != nil {
		t.Errorf("Unexpected error on `client.User.Current()`, got %v", err)
	}
	if user.Username == "" {
		t.Error("Expected user, got nothing")
	}
}
