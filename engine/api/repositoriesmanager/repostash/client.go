package repostash

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/facebookgo/httpcontrol"
	"github.com/go-stash/go-stash/stash"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

const stashHookKey string = "de.aeffle.stash.plugin.stash-http-get-post-receive-hook%3Ahttp-get-post-receive-hook"

func init() {
	stash.DefaultClient = &http.Client{
		Transport: &httpcontrol.Transport{
			RequestTimeout: 10 * time.Second,
			MaxTries:       3,
		},
	}
}

//StashClient is a github.com/reinbach/go-stash wrapper for CDS RepositoriesManagerClient interface
type StashClient struct {
	url              string
	client           *stash.Client
	disableSetStatus bool
}

//Repos returns the list of accessible repositories
func (s *StashClient) Repos() ([]sdk.VCSRepo, error) {
	repos := []sdk.VCSRepo{}
	stashRepos, err := s.client.Repos.List()
	if err != nil {
		return repos, err
	}
	for _, r := range stashRepos {
		var repoURL string
		if r.Link != nil {
			repoURL = r.Link.URL
		}

		var sshURL, httpURL string
		if r.Links != nil && r.Links.Clone != nil {
			for _, c := range r.Links.Clone {
				if c.Name == "http" {
					httpURL = c.URL
				}
				if c.Name == "ssh" {
					sshURL = c.URL
				}
			}
		}

		repo := sdk.VCSRepo{
			Name:         r.Name,
			Slug:         r.Slug,
			Fullname:     fmt.Sprintf("%s/%s", r.Project.Key, r.Slug),
			URL:          fmt.Sprintf("%s%s", s.url, repoURL),
			HTTPCloneURL: httpURL,
			SSHCloneURL:  sshURL,
		}
		repos = append(repos, repo)
	}
	return repos, nil
}

//RepoByFullname returns the repo from its fullname
func (s *StashClient) RepoByFullname(fullname string) (sdk.VCSRepo, error) {
	repo := sdk.VCSRepo{}
	t := strings.Split(fullname, "/")
	if len(t) != 2 {
		return repo, sdk.ErrRepoNotFound
	}
	r, err := s.client.Repos.Find(t[0], t[1])
	if err != nil {
		return repo, err
	}

	var sshURL, httpURL string
	if r.Links != nil && r.Links.Clone != nil {
		for _, c := range r.Links.Clone {
			if c.Name == "http" {
				httpURL = c.URL
			}
			if c.Name == "ssh" {
				sshURL = c.URL
			}
		}
	}

	var repoURL string
	if r.Link != nil {
		repoURL = r.Link.URL
	}

	repo = sdk.VCSRepo{
		Name:         r.Name,
		Slug:         r.Slug,
		Fullname:     fmt.Sprintf("%s/%s", r.Project.Key, r.Slug),
		URL:          fmt.Sprintf("%s%s", s.url, repoURL),
		HTTPCloneURL: httpURL,
		SSHCloneURL:  sshURL,
	}
	return repo, err
}

//Branches retrieves the branches from Stash
func (s *StashClient) Branches(fullname string) ([]sdk.VCSBranch, error) {
	var stashURL, _ = url.Parse(s.url)
	var stashBranchesKey = cache.Key("reposmanager", "stash", stashURL.Host, fullname, "branches")

	branches := []sdk.VCSBranch{}

	cache.Get(stashBranchesKey, &branches)
	if branches == nil || len(branches) == 0 {
		t := strings.Split(fullname, "/")
		if len(t) != 2 {
			return branches, sdk.ErrRepoNotFound
		}
		stashBranches, err := s.client.Branches.List(t[0], t[1])
		if err != nil {
			return branches, err
		}
		for _, sb := range stashBranches {
			b := sdk.VCSBranch{
				ID:           sb.ID,
				DisplayID:    sb.DisplayID,
				LatestCommit: sb.LatestHash,
				Default:      sb.IsDefault,
			}
			branches = append(branches, b)
		}
		cache.SetWithTTL(stashBranchesKey, branches, 60)
	}
	return branches, nil
}

//Branch retrieves the branch from Stash
func (s *StashClient) Branch(fullname, branchName string) (sdk.VCSBranch, error) {
	branch := sdk.VCSBranch{}
	t := strings.Split(fullname, "/")
	if len(t) != 2 {
		return branch, sdk.ErrRepoNotFound
	}
	stashBranch, err := s.client.Branches.Find(t[0], t[1], branchName)
	if err != nil {
		return branch, err
	}
	branch = sdk.VCSBranch{
		ID:           stashBranch.ID,
		DisplayID:    stashBranch.DisplayID,
		LatestCommit: stashBranch.LatestHash,
		Default:      stashBranch.IsDefault,
	}
	return branch, nil
}

//Commits returns commit data from a given starting commit, between two commits
//The commits may be identified by branch or tag name or by hash.
func (s *StashClient) Commits(repo, branch, since, until string) ([]sdk.VCSCommit, error) {
	commits := []sdk.VCSCommit{}
	t := strings.Split(repo, "/")
	if len(t) != 2 {
		return commits, fmt.Errorf("fullname %s must be <project>/<slug>", repo)
	}
	var stashURL, _ = url.Parse(s.url)
	var stashCommitsKey = cache.Key("reposmanager", "stash", stashURL.Host, repo, "commits", "since@"+since, "until@"+until)

	var stashCommits = []stash.Commit{}

	cache.Get(stashCommitsKey, &stashCommits)

	if stashCommits == nil || len(stashCommits) == 0 {
		var err error
		stashCommits, err = s.client.Commits.GetBetween(t[0], t[1], since, until)
		if err != nil {
			return commits, err
		}
		cache.Set(stashCommitsKey, stashCommits)
	}

	urlCommit := s.url + "/projects/" + t[0] + "/repos/" + t[1] + "/commits/"

	for _, sc := range stashCommits {
		c := sdk.VCSCommit{
			Hash:      sc.Hash,
			Timestamp: sc.Timestamp,
			Message:   sc.Message,
			Author: sdk.VCSAuthor{
				Name:  sc.Author.Name,
				Email: sc.Author.Email,
			},
			URL: urlCommit + sc.Hash,
		}
		var stashUser = stash.User{}
		var stashUserKey = cache.Key("reposmanager", "stash", stashURL.Host, sc.Author.Email)
		if !cache.Get(stashUserKey, &stashUser) && sc.Author.Email != "" {
			newStashUser, err := s.client.Users.FindByEmail(sc.Author.Email)
			if err != nil {
				log.Warning("Unable to get stash user %s : %s", sc.Author.Email, err)
				newStashUserUnknown := newUnknownStashUser(*sc.Author)
				cache.SetWithTTL(stashUserKey, newStashUserUnknown, 86400) // 1 day
				stashUser = *newStashUserUnknown
			} else {
				cache.Set(stashUserKey, newStashUser)
				stashUser = *newStashUser
			}
		}
		c.Author.DisplayName = stashUser.DisplayName
		if stashUser.Slug != "" && stashUser.Slug != "unknownSlug" {
			c.Author.Avatar = fmt.Sprintf("%s/users/%s/avatar.png", s.url, stashUser.Slug)
		}
		commits = append(commits, c)
	}
	return commits, nil
}

func newUnknownStashUser(author stash.Author) *stash.User {
	return &stash.User{
		Username:     author.Name,
		EmailAddress: author.Email,
		DisplayName:  author.Name,
		Slug:         "unknownSlug",
	}
}

//Commit retrieves a specific according to a hash
func (s *StashClient) Commit(repo, hash string) (sdk.VCSCommit, error) {
	commit := sdk.VCSCommit{}
	t := strings.Split(repo, "/")
	if len(t) != 2 {
		return commit, fmt.Errorf("fullname %s must be <project>/<slug>", repo)
	}

	var stashURL, _ = url.Parse(s.url)
	var stashCommitKey = cache.Key("reposmanager", "stash", stashURL.Host, repo, hash)
	var stashCommit = &stash.Commit{}

	cache.Get(stashCommitKey, stashCommit)

	if stashCommit.Hash == "" {
		var err error
		stashCommit, err = s.client.Commits.Get(t[0], t[1], hash)
		if err != nil {
			return commit, err
		}
		cache.SetWithTTL(stashCommitKey, stashCommit, -1)
	}
	urlCommit := s.url + "/projects/" + t[0] + "/repos/" + t[1] + "/commits/" + stashCommit.Hash
	commit = sdk.VCSCommit{
		Hash:      stashCommit.Hash,
		Timestamp: stashCommit.Timestamp,
		Message:   stashCommit.Message,
		Author: sdk.VCSAuthor{
			Name:  stashCommit.Author.Name,
			Email: stashCommit.Author.Email,
		},
		URL: urlCommit,
	}

	var stashUser = stash.User{}
	var stashUserKey = cache.Key("reposmanager", "stash", stashURL.Host, stashCommit.Author.Email)
	cache.Get(stashUserKey, &stashUser)
	if !cache.Get(stashUserKey, &stashUser) && stashCommit.Author.Email != "" {
		newStashUser, err := s.client.Users.FindByEmail(stashCommit.Author.Email)
		if err != nil {
			log.Warning("Unable to get stash user %s : %s", stashCommit.Author.Email, err)
			newStashUserUnknown := newUnknownStashUser(*stashCommit.Author)
			cache.SetWithTTL(stashUserKey, newStashUserUnknown, 86400) // 1 day
			stashUser = *newStashUserUnknown
		} else {
			cache.SetWithTTL(stashUserKey, newStashUser, -1)
			stashUser = *newStashUser
		}
	}
	commit.Author.DisplayName = stashUser.DisplayName
	if stashUser.Slug != "" && stashUser.Slug != "unknownSlug" {
		commit.Author.Avatar = fmt.Sprintf("%s/users/%s/avatar.png", s.url, stashUser.Slug)
	}

	return commit, nil
}

//CreateHook enables the defaut HTTP POST Hook in Stash
func (s *StashClient) CreateHook(repo, url string) error {
	var branchFilter, tagFilter, userFilter string

	t := strings.Split(repo, "/")
	if len(t) != 2 {
		return fmt.Errorf("fullname %s must be <project>/<slug>", repo)
	}
	log.Notice("CreateHook> Ask Stash to create Hook on %s/%s (%s): %s", t[0], t[1], stashHookKey, url)
	h, err := s.client.Hooks.CreateHook(t[0], t[1], stashHookKey, "POST", url, branchFilter, tagFilter, userFilter)
	if err != nil {
		if strings.Contains(err.Error(), "Unauthorized") {
			return sdk.ErrNoReposManagerClientAuth
		}
		return err
	}
	log.Notice("CreateHook> Hook created %s", h)
	return nil
}

//DeleteHook disables the defaut HTTP POST Hook in Stash
func (s *StashClient) DeleteHook(repo, url string) error {
	t := strings.Split(repo, "/")
	if len(t) != 2 {
		return fmt.Errorf("fullname %s must be <project>/<slug>", repo)
	}
	log.Notice("DeleteHook> Ask Stash to delete Hook on %s/%s (%s) : %s", t[0], t[1], stashHookKey, url)
	if err := s.client.Hooks.DeleteHook(t[0], t[1], stashHookKey, url); err != nil {
		if strings.Contains(err.Error(), "Unauthorized") {
			return sdk.ErrNoReposManagerClientAuth
		}
		return err
	}
	log.Notice("DeleteHook> Hook successfully deleted")
	return nil
}

//PushEvents is not implemented
func (s *StashClient) PushEvents(repo string, dateRef time.Time) ([]sdk.VCSPushEvent, time.Duration, error) {
	return nil, 0.0, fmt.Errorf("Not implemented on stash")
}

const (
	inProgress = "INPROGRESS"
	successful = "SUCCESSFUL"
	failed     = "FAILED"
)

//SetStatus set build status on stash
func (s *StashClient) SetStatus(event sdk.Event) error {
	log.Debug("process> receive: type:%s all: %+v", event.EventType, event)
	var eventpb sdk.EventPipelineBuild

	if event.EventType != fmt.Sprintf("%T", sdk.EventPipelineBuild{}) {
		return nil
	}

	if s.disableSetStatus {
		log.Warning("⚠ Stash statuses are disabled")
		return nil
	}

	if err := mapstructure.Decode(event.Payload, &eventpb); err != nil {
		log.Warning("Error during consumption: %s", err)
		return err
	}

	log.Debug("Process event:%+v", event)

	cdsProject := eventpb.ProjectKey
	cdsApplication := eventpb.ApplicationName
	cdsPipelineName := eventpb.PipelineName
	cdsBuildNumber := eventpb.BuildNumber
	cdsEnvironmentName := eventpb.EnvironmentName

	key := fmt.Sprintf("%s-%s-%s",
		cdsProject,
		cdsApplication,
		cdsPipelineName,
	)

	// project/CDS/application/cds2tat/pipeline/monPipeline/build/855?env=monEnvi
	url := fmt.Sprintf("%s/#/project/%s/application/%s/pipeline/%s/build/%d?env=%s",
		viper.GetString("base_url"),
		cdsProject,
		cdsApplication,
		cdsPipelineName,
		cdsBuildNumber,
		url.QueryEscape(cdsEnvironmentName),
	)

	status := stash.Status{
		Key:   key,
		Name:  fmt.Sprintf("%s%d", key, cdsBuildNumber),
		State: getBitbucketStateFromStatus(eventpb.Status),
		URL:   url,
	}

	log.Debug("SetStatus> hash:%s status:%+v", eventpb.Hash, status)
	if err := s.client.Commits.SetStatus(eventpb.Hash, status); err != nil {
		return fmt.Errorf("SetStatus> err on bitbucket: %ss", err)
	}

	return nil
}

func getBitbucketStateFromStatus(status sdk.Status) string {
	switch status {
	case sdk.StatusSuccess:
		return successful
	case sdk.StatusWaiting:
		return inProgress
	case sdk.StatusBuilding:
		return inProgress
	case sdk.StatusFail:
		return failed
	default:
		return failed
	}
}
