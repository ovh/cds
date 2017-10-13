package github

import "fmt"

//CreateHook is not implemented
func (g *githubClient) CreateHook(repo, url string) error {
	return fmt.Errorf("Not yet implemented on github")
}

//DeleteHook is not implemented
func (g *githubClient) DeleteHook(repo, url string) error {
	return fmt.Errorf("Not yet implemented on github")
}
