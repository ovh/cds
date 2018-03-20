package hooks

import (
	"strings"

	"github.com/ovh/cds/sdk"
)

func fillPayload(pushEvent sdk.VCSPushEvent) map[string]string {
	payload := make(map[string]string)
	payload["git.author"] = pushEvent.Commit.Author.Name
	payload["git.author.email"] = pushEvent.Commit.Author.Email
	payload["git.branch"] = strings.TrimPrefix(strings.TrimPrefix(pushEvent.Branch.DisplayID, "refs/heads/"), "refs/tags/")
	payload["git.hash"] = pushEvent.Commit.Hash
	payload["git.repository"] = pushEvent.Repo
	payload["cds.triggered_by.username"] = pushEvent.Commit.Author.DisplayName
	payload["cds.triggered_by.fullname"] = pushEvent.Commit.Author.Name
	payload["cds.triggered_by.email"] = pushEvent.Commit.Author.Email
	payload["git.message"] = pushEvent.Commit.Message

	if strings.HasPrefix(pushEvent.Branch.DisplayID, "refs/tags/") {
		payload["git.tag"] = strings.TrimPrefix(pushEvent.Branch.DisplayID, "refs/tags/")
	}

	return payload
}
