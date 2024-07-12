package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func arrayContains(array interface{}, s interface{}) bool {
	b := sdk.InterfaceSlice(array)
	for _, i := range b {
		if reflect.DeepEqual(i, s) {
			return true
		}
	}
	return false
}

func findAncestors(allCommits []Commit, since string) []string {
	ancestors := []string{}
	var i int
	var limit = len(allCommits) * len(allCommits)

ancestorLoop:
	if i > limit {
		return ancestors
	}

	for _, c := range allCommits {
		i++
		if c.Sha == since {
			for _, p := range c.Parents {
				if !arrayContains(ancestors, p.Sha) {
					ancestors = append(ancestors, p.Sha)
					goto ancestorLoop
				}
			}
		} else if arrayContains(ancestors, c.Sha) {
			for _, p := range c.Parents {
				if !arrayContains(ancestors, p.Sha) {
					ancestors = append(ancestors, p.Sha)
					goto ancestorLoop
				}
			}
		}

	}
	return ancestors
}

func filterCommits(allCommits []Commit, since, until string) []Commit {
	commits := []Commit{}

	sinceAncestors := findAncestors(allCommits, since)
	untilAncestors := findAncestors(allCommits, until)

	//We have to delete all common ancestors between sinceAncestors and untilAncestors
	toDelete := []string{}
	for _, c := range untilAncestors {
		if c == since {
			toDelete = append(toDelete, c)
		}
		if arrayContains(sinceAncestors, c) {
			toDelete = append(toDelete, c)
		}
	}

	for _, d := range toDelete {
		for i, x := range untilAncestors {
			if x == d {
				untilAncestors = append(untilAncestors[:i], untilAncestors[i+1:]...)
			}
		}
	}

	untilAncestors = append(untilAncestors, until)
	for _, c := range allCommits {
		if arrayContains(untilAncestors, c.Sha) {
			commits = append(commits, c)
		}
	}

	return commits
}

// Commits returns the commits list on a branch between a commit SHA (since) until another commit SHA (until). The branch is given by the branch of the first commit SHA (since)
func (g *githubClient) Commits(ctx context.Context, repo, theBranch, since, until string) ([]sdk.VCSCommit, error) {
	var commitsResult []sdk.VCSCommit

	log.Debug(ctx, "Looking for commits on repo %s since = %s until = %s", repo, since, until)
	k := cache.Key("vcs", "github", "commits", repo, "since="+since, "until="+until)
	find, err := g.Cache.Get(ctx, k, &commitsResult)
	if err != nil {
		log.Error(ctx, "cannot get from cache %s: %v", k, err)
	}
	if find {
		return commitsResult, nil
	}
	var sinceDate time.Time
	// Calculate since commit
	if since == "" {
		// If no since commit, take from the beginning of the branch
		b, errB := g.Branch(ctx, repo, sdk.VCSBranchFilters{BranchName: theBranch})
		if errB != nil {
			return nil, errB
		}
		if b == nil {
			return nil, fmt.Errorf("Commits>Cannot find branch %s", theBranch)
		}
		for _, c := range b.Parents {
			cp, errCP := g.Commit(ctx, repo, c)
			if errCP != nil {
				return nil, errCP
			}
			d := time.Unix(cp.Timestamp/1000, 0)
			if d.After(sinceDate) {
				// To not get the parent commit
				sinceDate = d.Add(1 * time.Second)
			}
		}
	} else {
		sinceCommit, errC := g.Commit(ctx, repo, since)
		if errC != nil {
			return nil, errC
		}
		sinceDate = time.Unix(sinceCommit.Timestamp/1000, 0)
	}

	var untilDate time.Time
	if until == "" {
		// If no until commit take until the end of the branch
		untilDate = time.Now()
	} else {
		untilCommit, errC := g.Commit(ctx, repo, until)
		if errC != nil {
			return nil, errC
		}
		untilDate = time.Unix(untilCommit.Timestamp/1000, 0)
	}

	//Get Commit List
	theCommits, err := g.allCommitBetween(ctx, repo, untilDate, sinceDate, theBranch)
	if err != nil {
		return nil, err
	}
	if since != "" {
		log.Debug(ctx, "filter commit (%d) between %s and %s", len(theCommits), since, until)
		theCommits = filterCommits(theCommits, since, until)
	}

	//Convert to sdk.VCSCommit
	for _, c := range theCommits {
		commit := sdk.VCSCommit{
			Timestamp: c.Commit.Author.Date.Unix() * 1000,
			Message:   c.Commit.Message,
			Hash:      c.Sha,
			URL:       c.HTMLURL,
			Author: sdk.VCSAuthor{
				DisplayName: c.Commit.Author.Name,
				Email:       c.Commit.Author.Email,
				Name:        c.Commit.Author.Name,
				Avatar:      c.Author.AvatarURL,
				Slug:        c.Commit.Author.Name,
				ID:          strconv.Itoa(c.Author.ID),
			},
			Committer: sdk.VCSAuthor{
				DisplayName: c.Commit.Committer.Name,
				Email:       c.Commit.Committer.Email,
				Name:        c.Commit.Committer.Name,
				Avatar:      c.Committer.AvatarURL,
				Slug:        c.Committer.Login,
				ID:          strconv.Itoa(c.Committer.ID),
			},
		}

		commitsResult = append(commitsResult, commit)
	}

	key := cache.Key("vcs", "github", "commits", repo, "since="+since, "until="+until)
	if err := g.Cache.SetWithTTL(ctx, key, commitsResult, 3*60*60); err != nil {
		log.Error(ctx, "cannot SetWithTTL: %s: %v", key, err)
	}

	return commitsResult, nil
}

func (g *githubClient) allCommitBetween(ctx context.Context, repo string, untilDate time.Time, sinceDate time.Time, branch string) ([]Commit, error) {
	var commits = []Commit{}
	urlValues := url.Values{}
	urlValues.Add("sha", branch)
	urlValues.Add("since", sinceDate.Format(time.RFC3339))
	urlValues.Add("until", untilDate.Format(time.RFC3339))

	var nextPage = "/repos/" + repo + "/commits"
	for nextPage != "" {
		if ctx.Err() != nil {
			break
		}

		if strings.Contains(nextPage, "?") {
			nextPage += "&"
		} else {
			nextPage += "?"
		}
		status, body, headers, err := g.get(ctx, nextPage+urlValues.Encode(), withoutETag)
		if err != nil {
			log.Warn(ctx, "githubClient.Commits> Error %s", err)
			return nil, err
		}
		if status >= 400 {
			log.Warn(ctx, "githubClient.Commits> Error %s", errorAPI(body))
			return nil, sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
		}
		nextCommits := []Commit{}

		if err := sdk.JSONUnmarshal(body, &nextCommits); err != nil {
			log.Warn(ctx, "githubClient.Commits> Unable to parse github commits: %s", err)
			return nil, err
		}

		commits = append(commits, nextCommits...)
		nextPage = getNextPage(headers)
	}

	return commits, nil
}

// Commit Get a single commit
// https://developer.github.com/v3/repos/commits/#get-a-single-commit
func (g *githubClient) Commit(ctx context.Context, repo, hash string) (sdk.VCSCommit, error) {
	url := "/repos/" + repo + "/commits/" + hash
	status, body, _, err := g.get(ctx, url)
	if err != nil {
		log.Warn(ctx, "githubClient.Commit> Error %s", err)
		return sdk.VCSCommit{}, err
	}
	if status >= 400 {
		return sdk.VCSCommit{}, sdk.NewError(sdk.ErrRepoNotFound, errorAPI(body))
	}
	c := Commit{}

	//Github may return 304 status because we are using conditional request with ETag based headers
	if status == http.StatusNotModified {
		//If repo isn't updated, lets get them from cache
		k := cache.Key("vcs", "github", "commit", sdk.Hash512(g.OAuthToken+g.username), url)
		if _, err := g.Cache.Get(ctx, k, &c); err != nil {
			log.Error(ctx, "cannot get from cache %s: %v", k, err)
		}
	} else {
		if err := sdk.JSONUnmarshal(body, &c); err != nil {
			log.Warn(ctx, "githubClient.Commit> Unable to parse github commit: %s", err)
			return sdk.VCSCommit{}, err
		}
		//Put the body on cache for one hour and one minute
		k := cache.Key("vcs", "github", "commit", sdk.Hash512(g.OAuthToken+g.username), url)
		if err := g.Cache.SetWithTTL(ctx, k, c, 61*60); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", k, err)
		}
	}

	commit := sdk.VCSCommit{
		Timestamp: c.Commit.Author.Date.Unix() * 1000,
		Message:   c.Commit.Message,
		Hash:      c.Sha,
		Author: sdk.VCSAuthor{
			DisplayName: c.Commit.Author.Name,
			Email:       c.Commit.Author.Email,
			Name:        c.Author.Login,
			Avatar:      c.Author.AvatarURL,
			ID:          strconv.Itoa(c.Author.ID),
		},
		URL:       c.HTMLURL,
		Verified:  c.Commit.Verification.Verified,
		Signature: c.Commit.Verification.Signature,
		Committer: sdk.VCSAuthor{
			DisplayName: c.Commit.Author.Name,
			Email:       c.Commit.Author.Email,
			Name:        c.Author.Login,
			Avatar:      c.Author.AvatarURL,
			ID:          strconv.Itoa(c.Author.ID),
		},
	}

	return commit, nil
}

func (g *githubClient) CommitsBetweenRefs(ctx context.Context, repo, base, head string) ([]sdk.VCSCommit, error) {
	var commits []sdk.VCSCommit
	url := fmt.Sprintf("/repos/%s/compare/%s...%s", repo, base, head)
	status, body, _, err := g.get(ctx, url)
	if err != nil {
		log.Warn(ctx, "githubClient.CommitsBetweenRefs> Error %s", err)
		return commits, err
	}
	if status >= 400 {
		return commits, sdk.NewError(sdk.ErrRepoNotFound, errorAPI(body))
	}

	var diff DiffCommits
	//Github may return 304 status because we are using conditional request with ETag based headers
	if status == http.StatusNotModified {
		//If repo isn't updated, lets get them from cache
		k := cache.Key("vcs", "github", "commitdiff", sdk.Hash512(g.OAuthToken+g.username), url)
		if _, err := g.Cache.Get(ctx, k, &commits); err != nil {
			log.Error(ctx, "cannot get from cache %s: %v", k, err)
		}
	} else {
		if err := sdk.JSONUnmarshal(body, &diff); err != nil {
			log.Warn(ctx, "githubClient.CommitsBetweenRefs> Unable to parse github commit: %s", err)
			return commits, err
		}

		commits = make([]sdk.VCSCommit, len(diff.Commits))
		for i, commit := range diff.Commits {
			commits[i] = sdk.VCSCommit{
				Timestamp: commit.Commit.Author.Date.Unix() * 1000,
				Message:   commit.Commit.Message,
				Hash:      commit.Sha,
				Author: sdk.VCSAuthor{
					DisplayName: commit.Commit.Author.Name,
					Email:       commit.Commit.Author.Email,
					Name:        commit.Author.Login,
					Avatar:      commit.Author.AvatarURL,
					ID:          strconv.Itoa(commit.Author.ID),
				},
				URL: commit.HTMLURL,
			}
		}
		//Put the body on cache for one hour and one minute
		k := cache.Key("vcs", "github", "commitdiff", sdk.Hash512(g.OAuthToken+g.username), url)
		if err := g.Cache.SetWithTTL(ctx, k, &commits, 61*60); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", k, err)
		}
	}

	return commits, nil
}
