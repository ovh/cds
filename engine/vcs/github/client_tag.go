package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Tags returns list of tags for a repo
func (g *githubClient) Tags(ctx context.Context, fullname string) ([]sdk.VCSTag, error) {
	var tags []Ref
	var noEtag bool
	var attempt int
	nextPage := "/repos/" + fullname + "/git/refs/tags"

	for {
		if nextPage != "" {
			var opt getArgFunc
			if noEtag {
				opt = withoutETag
			} else {
				opt = withETag
			}

			attempt++
			status, body, headers, err := g.get(nextPage, opt)
			if err != nil {
				log.Warning("githubClient.Tags> Error %s", err)
				return nil, err
			}
			if status >= 400 {
				return nil, sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
			}
			nextTags := []Ref{}

			//Github may return 304 status because we are using conditional request with ETag based headers
			if status == http.StatusNotModified {
				//If repos aren't updated, lets get them from cache
				g.Cache.Get(cache.Key("vcs", "github", "tags", g.OAuthToken, "/repos/"+fullname+"/tags"), &tags)
				if len(tags) != 0 || attempt > 5 {
					//We found tags, let's exit the loop
					break
				}
				//If we did not found any branch in cache, let's retry (same nextPage) without etag
				noEtag = true
				continue
			} else {
				if err := json.Unmarshal(body, &nextTags); err != nil {
					log.Warning("githubClient.Tags> Unable to parse github tags: %s", err)
					return nil, err
				}
			}

			tags = append(tags, nextTags...)
			nextPage = getNextPage(headers)
		} else {
			break
		}
	}

	//Put the body on cache for one hour and one minute
	g.Cache.SetWithTTL(cache.Key("vcs", "github", "tags", g.OAuthToken, "/repos/"+fullname+"/tags"), tags, 61*60)

	fmt.Printf("tags %+v\n", tags)
	tagsResult := make([]sdk.VCSTag, len(tags))
	for i, tag := range tags {
		tagsResult[i] = sdk.VCSTag{
			Tag: strings.Replace(tag.Ref, "refs/tags/", "", 1),
			Sha: tag.Object.Sha,
		}
	}

	return tagsResult, nil
}
