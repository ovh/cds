package github

import (
	"context"
	"net/http"
	"strings"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func (g *githubClient) Tag(ctx context.Context, fullname string, tagName string) (sdk.VCSTag, error) {
	tags, err := g.Tags(ctx, fullname)
	if err != nil {
		return sdk.VCSTag{}, err
	}
	for _, t := range tags {
		if t.Tag == tagName {
			return t, nil
		}
	}
	return sdk.VCSTag{}, sdk.WrapError(sdk.ErrNotFound, "tag not found")
}

// Tags returns list of tags for a repo
func (g *githubClient) Tags(ctx context.Context, fullname string) ([]sdk.VCSTag, error) {
	var tags []Ref
	var noEtag bool
	var attempt int

	nextPage := "/repos/" + fullname + "/git/refs/tags"
	for nextPage != "" {
		if ctx.Err() != nil {
			break
		}

		var opt getArgFunc
		if noEtag {
			opt = withoutETag
		} else {
			opt = withETag
		}

		attempt++
		status, body, headers, err := g.get(ctx, nextPage, opt)
		if err != nil {
			log.Warn(ctx, "githubClient.Tags> Error %s", err)
			return nil, err
		}
		if status >= 400 {
			if status == http.StatusNotFound {
				log.Debug(ctx, "githubClient.Tags> status 404 return nil because no tags found")
				return nil, nil
			}
			return nil, sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
		}
		nextTags := []Ref{}

		//Github may return 304 status because we are using conditional request with ETag based headers
		if status == http.StatusNotModified {
			//If repos aren't updated, lets get them from cache
			k := cache.Key("vcs", "github", "tags", sdk.Hash512(g.OAuthToken+g.username), "/repos/"+fullname+"/tags")
			if _, err := g.Cache.Get(k, &tags); err != nil {
				log.Error(ctx, "cannot get from cache %s:%v", k, err)
			}
			if len(tags) != 0 || attempt > 5 {
				//We found tags, let's exit the loop
				break
			}
			//If we did not found any branch in cache, let's retry (same nextPage) without etag
			noEtag = true
			continue
		} else {
			if err := sdk.JSONUnmarshal(body, &nextTags); err != nil {
				log.Warn(ctx, "githubClient.Tags> Unable to parse github tags: %s", err)
				return nil, err
			}
		}

		tags = append(tags, nextTags...)
		nextPage = getNextPage(headers)
	}

	//Put the body on cache for one hour and one minute
	k := cache.Key("vcs", "github", "tags", sdk.Hash512(g.OAuthToken+g.username), "/repos/"+fullname+"/tags")
	if err := g.Cache.SetWithTTL(k, tags, 61*60); err != nil {
		log.Error(ctx, "cannot SetWithTTL: %s: %v", k, err)
	}

	tagsResult := make([]sdk.VCSTag, len(tags))
	j := 0
	for i := len(tags) - 1; i >= 0; i-- {
		tagsResult[j] = sdk.VCSTag{
			Tag: strings.Replace(tags[i].Ref, sdk.GitRefTagPrefix, "", 1),
			Sha: tags[i].Object.Sha,
		}
		j++
	}

	return tagsResult, nil
}
