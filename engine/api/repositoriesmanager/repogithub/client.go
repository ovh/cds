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
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
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

			//Github may return 304 status because we are using conditional request with ETag based headers
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

	//Github may return 304 status because we are using conditional request with ETag based headers
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

			//Github may return 304 status because we are using conditional request with ETag based headers
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
		for _, p := range b.Commit.Parents {
			branch.Parents = append(branch.Parents, p.Sha)
		}
		branchesResult = append(branchesResult, branch)
	}

	return branchesResult, nil
}

// Branch returns only detail of a branch
func (g *GithubClient) Branch(fullname, theBranch string) (*sdk.VCSBranch, error) {
	cacheBranchKey := cache.Key("reposmanager", "github", "branch", g.OAuthToken, "/repos/"+fullname+"/branch"+theBranch)
	repo, err := g.repoByFullname(fullname)
	if err != nil {
		return nil, err
	}

	url := "/repos/" + fullname + "/branches/" + theBranch
	status, body, _, err := g.get(url)
	if err != nil {
		cache.Delete(cacheBranchKey)
		return nil, err
	}
	if status >= 400 {
		cache.Delete(cacheBranchKey)
		return nil, sdk.NewError(sdk.ErrUnknownError, ErrorAPI(body))
	}

	//Github may return 304 status because we are using conditional request with ETag based headers
	var branch Branch
	if status == http.StatusNotModified {
		//If repos aren't updated, lets get them from cache
		cache.Get(cacheBranchKey, &branch)
	} else {
		if err := json.Unmarshal(body, &branch); err != nil {
			log.Warning("GithubClient.Branch> Unable to parse github branch: %s", err)
			return nil, err
		}
	}

	if branch.Name == nil {
		log.Warning("GithubClient.Branch> Cannot find branch %s: %v", branch, theBranch)
		cache.Delete(cacheBranchKey)
		return nil, fmt.Errorf("GithubClient.Branch > Cannot find branch %s", theBranch)
	}

	//Put the body on cache for one hour and one minute
	cache.SetWithTTL(cache.Key("reposmanager", "github", "branches", g.OAuthToken, "/repos/"+fullname+"/branch"+theBranch), branch, 61*60)

	branchResult := &sdk.VCSBranch{
		DisplayID:    *branch.Name,
		ID:           *branch.Name,
		LatestCommit: branch.Commit.Sha,
		Default:      *branch.Name == *repo.DefaultBranch,
	}

	if branch.Commit != nil {
		for _, p := range branch.Commit.Parents {
			branchResult.Parents = append(branchResult.Parents, p.Sha)
		}
	}

	return branchResult, nil
}

// Commits returns the commits list on a branch between a commit SHA (since) until another commit SHA (until). The branch is given by the branch of the first commit SHA (since)
func (g *GithubClient) Commits(repo, theBranch, since, until string) ([]sdk.VCSCommit, error) {
	var commitsResult []sdk.VCSCommit

	log.Debug("Looking for commits on repo %s since = %s until = %s", repo, since, until)
	if cache.Get(cache.Key("reposmanager", "github", "commits", repo, "since="+since, "until="+until), &commitsResult) {
		return commitsResult, nil
	}

	var sinceDate time.Time
	// Calculate since commit
	if since == "" {
		// If no since commit, take from the begining of the branch
		b, errB := g.Branch(repo, theBranch)
		if errB != nil {
			return nil, errB
		}
		if b == nil {
			return nil, fmt.Errorf("Commits>Cannot find branch %s", theBranch)
		}
		for _, c := range b.Parents {
			cp, errCP := g.Commit(repo, c)
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
		sinceCommit, errC := g.Commit(repo, since)
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
		untilCommit, errC := g.Commit(repo, until)
		if errC != nil {
			return nil, errC
		}
		untilDate = time.Unix(untilCommit.Timestamp/1000, 0)
	}

	//Get Commit List
	theCommits, err := g.allCommitBetween(repo, untilDate, sinceDate, theBranch)
	if err != nil {
		return nil, err
	}
	if since != "" {
		log.Debug("filter commit (%d) between %s and %s", len(theCommits), since, until)
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

	//Github may return 304 status because we are using conditional request with ETag based headers
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

func (g *GithubClient) allCommitBetween(repo string, untilDate time.Time, sinceDate time.Time, branch string) ([]Commit, error) {
	var commits = []Commit{}
	urlValues := url.Values{}
	urlValues.Add("sha", branch)
	urlValues.Add("since", sinceDate.Format(time.RFC3339))
	urlValues.Add("until", untilDate.Format(time.RFC3339))
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

	//Github may return 304 status because we are using conditional request with ETag based headers
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
		log.Error("Github Rate Limit nearly exceeded %v", rateLimit)
		return ErrorRateLimit
	}
	return nil
}

//GetEvents calls Github et returns GithubEvents as []interface{}
func (g *GithubClient) GetEvents(fullname string, dateRef time.Time) ([]interface{}, time.Duration, error) {
	log.Debug("GithubClient.GetEvents> loading events for %s after %v", fullname, dateRef)
	var events = []interface{}{}

	interval := 60 * time.Second

	status, body, headers, err := g.get("/repos/" + fullname + "/events")
	if err != nil {
		log.Warning("GithubClient.GetEvents> Error %s", err)
		return nil, interval, err
	}

	if status >= http.StatusBadRequest {
		err := sdk.NewError(sdk.ErrUnknownError, ErrorAPI(body))
		log.Warning("GithubClient.GetEvents> Error http %s", err)
		return nil, interval, err
	}

	if status == http.StatusNotModified {
		return nil, interval, fmt.Errorf("No new events")
	}

	nextEvents := []Event{}
	if err := json.Unmarshal(body, &nextEvents); err != nil {
		log.Warning("GithubClient.GetEvents> Unable to parse github events: %s", err)
		return nil, interval, fmt.Errorf("Unable to parse github events %s: %s", string(body), err)
	}

	//Check here only events after the reference date and only of type PushEvent or CreateEvent
	for _, e := range nextEvents {
		var skipEvent bool
		if e.CreatedAt.After(dateRef) {
			for i := range events {
				e1 := events[i].(Event)
				if e.Payload.Ref == e1.Payload.Ref {
					if e.Type == "DeleteEvent" && e1.Type == "CreateEvent" {
						//Delete event after create event
						if e.CreatedAt.After(e1.CreatedAt.Time) {
							skipEvent = true
						} else {
							//Avoid delete
							events = append(events[:i], events[i+1:]...)
						}
						break
					} else if e.Type == "CreateEvent" && e1.Type == "DeleteEvent" {
						//Delete event before create event
						if e.CreatedAt.After(e1.CreatedAt.Time) {
							events = append(events[:i], events[i+1:]...)
						}
						break
					}
				}
			}

			fmt.Println(events)
			if !skipEvent {
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

	return events, interval, nil
}

//PushEvents returns push events as commits
func (g *GithubClient) PushEvents(fullname string, iEvents []interface{}) ([]sdk.VCSPushEvent, error) {
	events := Events{}
	//Cast all the events
	for _, i := range iEvents {
		e := i.(Event)
		if e.Type == "PushEvent" {
			events = append(events, e)
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
		if err != nil || branch == nil {
			log.Warning("GithubClient.PushEvents> Unable to find branch %s in %s : %s", b, fullname, err)
			continue
		}
		res = append(res, sdk.VCSPushEvent{
			Branch: *branch,
			Commit: c,
		})
	}

	return res, nil
}

//CreateEvents checks create events from a event list
func (g *GithubClient) CreateEvents(fullname string, iEvents []interface{}) ([]sdk.VCSCreateEvent, error) {
	events := Events{}
	//Cast all the events
	for _, i := range iEvents {
		e := i.(Event)
		if e.Type == "CreateEvent" {
			events = append(events, e)
		}
	}

	res := []sdk.VCSCreateEvent{}
	for _, e := range events {
		b := e.Payload.Ref
		branch, err := g.Branch(fullname, b)
		if err != nil || branch == nil {
			log.Warning("GithubClient.CreateEvents> Unable to find branch %s in %s : %s", b, fullname, err)
			continue
		}
		event := sdk.VCSCreateEvent{
			Branch: *branch,
		}

		c, err := g.Commit(fullname, branch.LatestCommit)
		if err != nil {
			log.Warning("GithubClient.CreateEvents> Unable to find commit %s in %s : %s", branch.LatestCommit, fullname, err)
			continue
		}
		event.Commit = c

		res = append(res, event)
	}

	log.Debug("GithubClient.CreateEvents> found %d create events : %#v", len(res), res)

	return res, nil
}

//DeleteEvents checks delete events from a event list
func (g *GithubClient) DeleteEvents(fullname string, iEvents []interface{}) ([]sdk.VCSDeleteEvent, error) {
	events := Events{}
	//Cast all the events
	for _, i := range iEvents {
		e := i.(Event)
		if e.Type == "DeleteEvent" {
			events = append(events, e)
		}
	}

	res := []sdk.VCSDeleteEvent{}
	for _, e := range events {
		event := sdk.VCSDeleteEvent{
			Branch: sdk.VCSBranch{
				DisplayID: e.Payload.Ref,
			},
		}
		res = append(res, event)
	}

	log.Debug("GithubClient.DeleteEvents> found %d delete events : %#v", len(res), res)
	return res, nil
}

//PullRequestEvents checks pull request events from a event list
func (g *GithubClient) PullRequestEvents(fullname string, iEvents []interface{}) ([]sdk.VCSPullRequestEvent, error) {
	fmt.Println("coucou")
	fmt.Println(iEvents)
	events := Events{}
	//Cast all the events
	for _, i := range iEvents {
		e := i.(Event)
		if e.Type == "PullRequestEvent" {
			events = append(events, e)
		}
	}

	res := []sdk.VCSPullRequestEvent{}
	for _, e := range events {
		event := sdk.VCSPullRequestEvent{
			Action: e.Payload.Action,
			Base: sdk.VCSPushEvent{
				Branch: sdk.VCSBranch{
					ID:           e.Payload.PullRequest.Base.Ref,
					DisplayID:    e.Payload.PullRequest.Base.Ref,
					LatestCommit: e.Payload.PullRequest.Base.Sha,
				},
				Commit: sdk.VCSCommit{
					// Author: sdk.VCSAuthor{
					// 	Name:        e.Payload.PullRequest.Base.User.Name,
					// 	DisplayName: e.Payload.PullRequest.Base.User.Login,
					// 	Email:       e.Payload.PullRequest.Base.User.Email,
					// },
					Hash:    e.Payload.PullRequest.Base.Sha,
					Message: e.Payload.PullRequest.Base.Label,
				},
				CloneURL: *e.Payload.PullRequest.Base.Repo.CloneURL,
			},
		}
		res = append(res, event)
	}

	log.Debug("GithubClient.PullRequestEvents> found %d pull request events : %#v", len(res), res)
	return res, nil
}
