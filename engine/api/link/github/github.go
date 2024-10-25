package github

import (
  "context"

  "github.com/ovh/cds/engine/api/driver/github"
  "github.com/ovh/cds/engine/api/link"
  "github.com/ovh/cds/sdk"
)

type LinkGithubDriver struct {
  d sdk.Driver
}

func NewLinkGithubDriver(cdsURL string, githubUI string, githubAPI string, clientID string, clientSecret string) link.LinkDriver {
  return &LinkGithubDriver{
    d: github.NewGithubDriver(cdsURL, githubUI, githubAPI, clientID, clientSecret),
  }
}

func (l LinkGithubDriver) GetUserInfo(ctx context.Context, req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
  return l.d.GetUserInfoFromDriver(ctx, req)
}

func (l LinkGithubDriver) GetDriver() sdk.Driver {
  return l.d
}
