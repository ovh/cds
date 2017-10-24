package github

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (g *githubClient) CreateHook(repo string, hook sdk.VCSHook) error {
	return fmt.Errorf("Not yet implemented")
}
func (g *githubClient) GetHook(repo, id string) (sdk.VCSHook, error) {
	return sdk.VCSHook{}, fmt.Errorf("Not yet implemented")
}
func (g *githubClient) UpdateHook(repo, id string, hook sdk.VCSHook) error {
	return fmt.Errorf("Not yet implemented")
}
func (g *githubClient) DeleteHook(repo string, hook sdk.VCSHook) error {
	return fmt.Errorf("Not yet implemented")
}
