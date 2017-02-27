package repogithub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// GithubClient is a github.com wrapper for CDS RepositoriesManagerClient interface
type GithubClient struct {
	ClientID         string
	OAuthToken       string
	DisableSetStatus bool
	DisableStatusURL bool
}

// Repos list repositories that are accessible to the authenticated user
// https://developer.github.com/v3/repos/#list-your-repositories
func (g *GithubClient) Repos() ([]sdk.VCSRepo, error) {
	var repos = []Repository{}
	var nextPage = "/user/repos"

	for {
		if nextPage != "" {
			status, body, headers, err := g.get(nextPage)
			if err != nil {
				log.Warning("GithubClient.Repos> Error %s", err)
				return nil, err
			}
			if status >= 400 {
				return nil, sdk.NewError(sdk.ErrUnknownError, ErrorAPI(body))
			}
			nextRepos := []Repository{}

			//Github may return 304 status because we are using conditionnal request with ETag based headers
			if status == http.StatusNotModified {
				//If repos aren't updated, lets get them from cache
				cache.Get(cache.Key("reposmanager", "github", "repos", g.OAuthToken, "/user/repos"), &repos)
				break
			} else {
				if err := json.Unmarshal(body, &nextRepos); err != nil {
					log.Warning("GithubClient.Repos> Unable to parse github repositories: %s", err)
					return nil, err
				}
			}

			repos = append(repos, nextRepos...)
			nextPage = getNextPage(headers)
		} else {
			break
		}
	}

	//Put the body on cache for one hour and one minute
	cache.SetWithTTL(cache.Key("reposmanager", "github", "repos", g.OAuthToken, "/user/repos"), repos, 61*60)

	responseRepos := []sdk.VCSRepo{}
	for _, repo := range repos {
		r := sdk.VCSRepo{
			ID:           strconv.Itoa(*repo.ID),
			Name:         *repo.Name,
			Slug:         strings.Split(*repo.FullName, "/")[0],
			Fullname:     *repo.FullName,
			URL:          *repo.HTMLURL,
			HTTPCloneURL: *repo.CloneURL,
			SSHCloneURL:  *repo.SSHURL,
		}
		responseRepos = append(responseRepos, r)
	}

	return responseRepos, nil
}

// RepoByFullname Get only one repo
// https://developer.github.com/v3/repos/#list-your-repositories
func (g *GithubClient) RepoByFullname(fullname string) (sdk.VCSRepo, error) {
	repo, err := g.repoByFullname(fullname)
	if err != nil {
		return sdk.VCSRepo{}, err
	}

	if repo.ID == nil {
		return sdk.VCSRepo{}, err
	}

	r := sdk.VCSRepo{
		ID:           strconv.Itoa(*repo.ID),
		Name:         *repo.Name,
		Slug:         strings.Split(*repo.FullName, "/")[0],
		Fullname:     *repo.FullName,
		URL:          *repo.HTMLURL,
		HTTPCloneURL: *repo.CloneURL,
		SSHCloneURL:  *repo.SSHURL,
	}
	return r, nil
}

func (g *GithubClient) repoByFullname(fullname string) (Repository, error) {
	url := "/repos/" + fullname
	status, body, _, err := g.get(url)
	if err != nil {
		log.Warning("GithubClient.Repos> Error %s", err)
		return Repository{}, err
	}
	if status >= 400 {
		return Repository{}, sdk.NewError(sdk.ErrRepoNotFound, ErrorAPI(body))
	}
	repo := Repository{}

	//Github may return 304 status because we are using conditionnal request with ETag based headers
	if status == http.StatusNotModified {
		//If repo isn't updated, lets get them from cache
		cache.Get(cache.Key("reposmanager", "github", "repo", g.OAuthToken, url), &repo)
	} else {
		if err := json.Unmarshal(body, &repo); err != nil {
			log.Warning("GithubClient.Repos> Unable to parse github repository: %s", err)
			return Repository{}, err
		}
		//Put the body on cache for one hour and one minute
		cache.SetWithTTL(cache.Key("reposmanager", "github", "repo", g.OAuthToken, url), repo, 61*60)
	}

	return repo, nil
}

// Branches returns list of branches for a repo
// https://developer.github.com/v3/repos/branches/#list-branches
func (g *GithubClient) Branches(fullname string) ([]sdk.VCSBranch, error) {
	var branches = []Branch{}
	var nextPage = "/repos/" + fullname + "/branches"

	repo, err := g.repoByFullname(fullname)
	if err != nil {
		return nil, err
	}

	for {
		if nextPage != "" {
			status, body, headers, err := g.get(nextPage)
			if err != nil {
				log.Warning("GithubClient.Branches> Error %s", err)
				return nil, err
			}
			if status >= 400 {
				return nil, sdk.NewError(sdk.ErrUnknownError, ErrorAPI(body))
			}
			nextBranches := []Branch{}

			//Github may return 304 status because we are using conditionnal request with ETag based headers
			if status == http.StatusNotModified {
				//If repos aren't updated, lets get them from cache
				cache.Get(cache.Key("reposmanager", "github", "branches", g.OAuthToken, "/repos/"+fullname+"/branches"), &branches)
				break
			} else {
				if err := json.Unmarshal(body, &nextBranches); err != nil {
					log.Warning("GithubClient.Branches> Unable to parse github branches: %s", err)
					return nil, err
				}
			}

			branches = append(branches, nextBranches...)

			nextPage = getNextPage(headers)
		} else {
			break
		}
	}

	//Put the body on cache for one hour and one minute
	cache.SetWithTTL(cache.Key("reposmanager", "github", "branches", g.OAuthToken, "/repos/"+fullname+"/branches"), branches, 61*60)

	branchesResult := []sdk.VCSBranch{}
	for _, b := range branches {
		branch := sdk.VCSBranch{
			DisplayID:    *b.Name,
			ID:           *b.Name,
			LatestCommit: b.Commit.Sha,
			Default:      *b.Name == *repo.DefaultBranch,
		}
		branchesResult = append(branchesResult, branch)
	}

	return branchesResult, nil
}

// Branch returns only detail of a branch
func (g *GithubClient) Branch(fullname, branch string) (sdk.VCSBranch, error) {
	//https://developer.github.com/v3/repos/branches/#get-branch
	//Get branch is still in developper preview, so were are using list branch
	branches, err := g.Branches(fullname)
	if err != nil {
		return sdk.VCSBranch{}, err
	}

	for _, b := range branches {
		if b.DisplayID == branch {
			return b, nil
		}
	}
	return sdk.VCSBranch{}, sdk.ErrNoBranch
}

// Commits returns the commits list on a branch between a commit SHA (since) until anotger commit SHA (until). The branch is given by the branch of the first commit SHA (since)
func (g *GithubClient) Commits(repo, theBranch, since, until string) ([]sdk.VCSCommit, error) {
	var theCommits []Commit
	var commitsResult []sdk.VCSCommit

	log.Debug("Looking for commits on repo %s since = %s until = %s", repo, since, until)
	if cache.Get(cache.Key("reposmanager", "github", "commits", repo, "since="+since, "until="+until), &commitsResult) {
		return commitsResult, nil
	}

	theCommits, err := g.allCommitsForBranch(repo, theBranch)
	if err != nil {
		return nil, err
	}

	log.Debug("Found %d commits for branch %s", len(theCommits), theBranch)

	//4. find the commits in the branch between SHA=since and SHA=until
	if since != "" {
		log.Debug("filter commit between %s and %s", since, until)
		theCommits = filterCommits(theCommits, since, until)
	}

	//5. convert to sdk.VCSCommit
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
			},
		}

		commitsResult = append(commitsResult, commit)
	}

	cache.SetWithTTL(cache.Key("reposmanager", "github", "commits", repo, "since="+since, "until="+until), commitsResult, 3*60*60)

	return commitsResult, nil
}

// User Get a single user
// https://developer.github.com/v3/users/#get-a-single-user
func (g *GithubClient) User(username string) (User, error) {
	url := "/users/" + username
	status, body, _, err := g.get(url)
	if err != nil {
		log.Warning("GithubClient.User> Error %s", err)
		return User{}, err
	}
	if status >= 400 {
		return User{}, sdk.NewError(sdk.ErrRepoNotFound, ErrorAPI(body))
	}
	user := User{}

	//Github may return 304 status because we are using conditionnal request with ETag based headers
	if status == http.StatusNotModified {
		//If repo isn't updated, lets get them from cache
		cache.Get(cache.Key("reposmanager", "github", "users", g.OAuthToken, url), &user)
	} else {
		if err := json.Unmarshal(body, &user); err != nil {
			log.Warning("GithubClient.User> Unable to parse github user: %s", err)
			return User{}, err
		}
		//Put the body on cache for one hour and one minute
		cache.SetWithTTL(cache.Key("reposmanager", "github", "users", g.OAuthToken, url), user, 61*60)
	}

	return user, nil
}

func (g *GithubClient) allCommitsForBranch(repo, branch string) ([]Commit, error) {
	var commits = []Commit{}
	urlValues := url.Values{}
	urlValues.Add("sha", branch)
	var nextPage = "/repos/" + repo + "/commits"

	for {
		if nextPage != "" {
			if strings.Contains(nextPage, "?") {
				nextPage += "&"
			} else {
				nextPage += "?"
			}
			status, body, headers, err := g.get(nextPage+urlValues.Encode(), withoutETag)
			if err != nil {
				log.Warning("GithubClient.Commits> Error %s", err)
				return nil, err
			}
			if status >= 400 {
				log.Warning("GithubClient.Commits> Error %s", ErrorAPI(body))
				return nil, sdk.NewError(sdk.ErrUnknownError, ErrorAPI(body))
			}
			nextCommits := []Commit{}

			if err := json.Unmarshal(body, &nextCommits); err != nil {
				log.Warning("GithubClient.Commits> Unable to parse github commits: %s", err)
				return nil, err
			}

			commits = append(commits, nextCommits...)
			nextPage = getNextPage(headers)
		} else {
			break
		}
	}
	return commits, nil
}

// Commit Get a single commit
// https://developer.github.com/v3/repos/commits/#get-a-single-commit
func (g *GithubClient) Commit(repo, hash string) (sdk.VCSCommit, error) {
	url := "/repos/" + repo + "/commits/" + hash
	status, body, _, err := g.get(url)
	if err != nil {
		log.Warning("GithubClient.Commit> Error %s", err)
		return sdk.VCSCommit{}, err
	}
	if status >= 400 {
		return sdk.VCSCommit{}, sdk.NewError(sdk.ErrRepoNotFound, ErrorAPI(body))
	}
	c := Commit{}

	//Github may return 304 status because we are using conditionnal request with ETag based headers
	if status == http.StatusNotModified {
		//If repo isn't updated, lets get them from cache
		cache.Get(cache.Key("reposmanager", "github", "commit", g.OAuthToken, url), &c)
	} else {
		if err := json.Unmarshal(body, &c); err != nil {
			log.Warning("GithubClient.Commit> Unable to parse github commit: %s", err)
			return sdk.VCSCommit{}, err
		}
		//Put the body on cache for one hour and one minute
		cache.SetWithTTL(cache.Key("reposmanager", "github", "commit", g.OAuthToken, url), c, 61*60)
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
		},
		URL: c.HTMLURL,
	}

	return commit, nil
}

//CreateHook is not implemented
func (g *GithubClient) CreateHook(repo, url string) error {
	return fmt.Errorf("Not yet implemented on github")
}

//DeleteHook is not implemented
func (g *GithubClient) DeleteHook(repo, url string) error {
	return fmt.Errorf("Not yet implemented on github")
}

// RateLimit Get your current rate limit status
// https://developer.github.com/v3/rate_limit/#get-your-current-rate-limit-status
func (g *GithubClient) RateLimit() error {
	url := "/rate_limit"
	status, body, _, err := g.get(url)
	if err != nil {
		log.Warning("GithubClient.RateLimit> Error %s", err)
		return err
	}
	if status >= 400 {
		return sdk.NewError(sdk.ErrUnknownError, ErrorAPI(body))
	}
	rateLimit := &RateLimit{}
	if err := json.Unmarshal(body, rateLimit); err != nil {
		log.Warning("GithubClient.RateLimit> Error %s", err)
		return err
	}
	if rateLimit.Rate.Remaining < 100 {
		log.Critical("Github Rate Limit nearly exceeded %v", rateLimit)
		return ErrorRateLimit
	}
	return nil
}

//PushEvents returns push events as commits
func (g *GithubClient) PushEvents(fullname string, dateRef time.Time) ([]sdk.VCSPushEvent, time.Duration, error) {
	log.Debug("GithubClient.PushEvents> loading events for %s after %v", fullname, dateRef)
	var events = []Event{}

	interval := 60 * time.Second

	status, body, headers, err := g.get("/repos/" + fullname + "/events")
	if err != nil {
		log.Warning("GithubClient.PushEvents> Error %s", err)
		return nil, interval, err
	}

	if status >= http.StatusBadRequest {
		err := sdk.NewError(sdk.ErrUnknownError, ErrorAPI(body))
		log.Warning("GithubClient.PushEvents> Error http %s", err)
		return nil, interval, err
	}

	if status == http.StatusNotModified {
		return nil, interval, fmt.Errorf("No new events")
	}

	nextEvents := []Event{}
	if err := json.Unmarshal(body, &nextEvents); err != nil {
		log.Warning("GithubClient.PushEvents> Unable to parse github events: %s", err)
		return nil, interval, fmt.Errorf("Unable to parse github events %s: %s", string(body), err)
	}
	//Check here only events after the reference date and only of type PushEvent or CreateEvent
	for _, e := range nextEvents {
		if e.CreatedAt.After(dateRef) {
			if e.Type == "PushEvent" || e.Type == "CreateEvent" { //May be we should manage PullRequestEvent; payload.action = opened
				events = append(events, e)
			}
		}
	}

	//Check poll interval
	if headers.Get("X-Poll-Interval") != "" {
		f, err := strconv.ParseFloat(headers.Get("X-Poll-Interval"), 64)
		if err == nil {
			interval = time.Duration(f) * time.Second
		}
	}

	lastCommitPerBranch := map[string]sdk.VCSCommit{}
	for _, e := range events {
		branch := strings.Replace(e.Payload.Ref, "refs/heads/", "", 1)
		for _, c := range e.Payload.Commits {
			commit := sdk.VCSCommit{
				Hash:      c.Sha,
				Message:   c.Message,
				Timestamp: e.CreatedAt.Unix() * 1000,
				URL:       c.URL,
				Author: sdk.VCSAuthor{
					DisplayName: c.Author.Name,
					Email:       c.Author.Email,
					Name:        e.Actor.DisplayLogin,
					Avatar:      e.Actor.AvatarURL,
				},
			}
			l, b := lastCommitPerBranch[branch]
			if !b || l.Timestamp < commit.Timestamp {
				lastCommitPerBranch[branch] = commit
				continue
			}
		}
	}

	res := []sdk.VCSPushEvent{}
	for b, c := range lastCommitPerBranch {
		branch, err := g.Branch(fullname, b)
		if err != nil {
			log.Warning("GithubClient.PushEvents> Unable to find branch %s in %s : %s", b, fullname, err)
			continue
		}
		res = append(res, sdk.VCSPushEvent{
			Branch: branch,
			Commit: c,
		})
	}

	return res, interval, nil
}
