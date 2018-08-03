package repo

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	zglob "github.com/mattn/go-zglob"
)

// Clone a git repository from the specified url to the destination path. Use Options to force the use of SSH Key and or PGP Key on this repo
func Clone(path, url string, opts ...Option) (Repo, error) {
	r := Repo{path: path, url: url}
	for _, f := range opts {
		if err := f(&r); err != nil {
			return r, err
		}
	}
	if r.verbose {
		r.log("Cloning %s\n", r.url)
	}
	_, err := r.runCmd("git", "clone", r.url, ".")
	if err != nil {
		return r, err
	}
	return r, nil
}

// New instanciance a repo instance from the path assuming the repo has already been cloned in.
func New(path string, opts ...Option) (Repo, error) {
	r := Repo{path: path}
	for _, f := range opts {
		if err := f(&r); err != nil {
			return r, err
		}
	}
	dotGit := filepath.Join(path, ".git")
	if _, err := os.Stat(dotGit); err != nil || os.IsNotExist(err) {
		return r, err
	}
	return r, nil
}

// FetchURL returns the git URL the the remote origin
func (r Repo) FetchURL() (string, error) {
	stdOut, err := r.runCmd("git", "remote", "show", "origin", "-n")
	if err != nil {
		return "", err
	}

	reader := bufio.NewReader(strings.NewReader(stdOut))
	var fetchURL string
	for {
		b, _, err := reader.ReadLine()
		if err == io.EOF || b == nil {
			break
		}
		if err != nil {
			return "", err
		}
		s := string(b)
		if strings.Contains(s, "Fetch URL:") {
			fetchURL = strings.Replace(s, "Fetch URL:", "", 1)
			fetchURL = strings.TrimSpace(fetchURL)
		}
	}

	return fetchURL, nil
}

// Name returns the name of the repo, deduced from the remote origin URL
func (r Repo) Name() (string, error) {
	fetchURL, err := r.FetchURL()
	if err != nil {
		return "", err
	}

	return trimURL(fetchURL)
}

// LocalConfigGet returns data from the local git config
func (r Repo) LocalConfigGet(section, key string) (string, error) {
	s, err := r.runCmd("git", "config", "--local", "--get", fmt.Sprintf("%s.%s", section, key))
	if err != nil {
		return "", err
	}
	return s[:len(s)-1], nil
}

// LocalConfigSet set data in the local git config
func (r Repo) LocalConfigSet(section, key, value string) error {
	conf, _ := r.LocalConfigGet(section, key)
	s := fmt.Sprintf("%s.%s", section, key)
	if conf != "" {
		if _, err := r.runCmd("git", "config", "--local", "--unset-all", s); err != nil {
			return err
		}
	}

	if _, err := r.runCmd("git", "config", "--local", "--add", s, value); err != nil {
		return err
	}

	return nil
}

func trimURL(fetchURL string) (string, error) {
	repoName := fetchURL

	if strings.HasSuffix(repoName, ".git") {
		repoName = repoName[:len(repoName)-4]
	}

	if strings.HasPrefix(repoName, "https://") {
		repoName = repoName[8:]
		for strings.Count(repoName, "/") > 1 {
			firstSlash := strings.Index(repoName, "/")
			if firstSlash == -1 {
				return "", fmt.Errorf("invalid url")
			}
			repoName = repoName[firstSlash+1:]
		}
		return repoName, nil
	}

	if strings.HasPrefix(repoName, "ssh://") {
		// ssh://[user@]server/project.git
		repoName = repoName[6:]
		firstSlash := strings.Index(repoName, "/")
		if firstSlash == -1 {
			return "", fmt.Errorf("invalid url")
		}
		repoName = repoName[firstSlash+1:]
	} else {
		// [user@]server:project.git
		firstSemicolon := strings.Index(repoName, ":")
		if firstSemicolon == -1 {
			return "", fmt.Errorf("invalid url")
		}
		repoName = repoName[firstSemicolon+1:]
	}

	return repoName, nil
}

// LatestCommit returns the latest commit of the current branch
func (r Repo) LatestCommit() (Commit, error) {
	c := Commit{}
	hash, err := r.runCmd("git", "rev-parse", "HEAD")
	if err != nil {
		return c, err
	}

	details, err := r.runCmd("git", "show", hash[:7], "--quiet", "--pretty=%at||%an||%s||%b")
	if err != nil {
		return c, err
	}

	c.LongHash = hash[:len(hash)-1]
	c.Hash = hash[:7]

	splittedDetails := strings.SplitN(details, "||", 4)

	ts, err := strconv.ParseInt(splittedDetails[0], 10, 64)
	if err != nil {
		return c, err
	}
	c.Date = time.Unix(ts, 0)
	c.Author = splittedDetails[1]
	c.Subject = splittedDetails[2]
	c.Body = splittedDetails[3]

	return c, nil
}

// CurrentBranch returns the current branch
func (r Repo) CurrentBranch() (string, error) {
	b, err := r.runCmd("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return b[:len(b)-1], nil
}

// FetchRemoteBranch runs a git fetch then checkout the remote branch
func (r Repo) FetchRemoteBranch(remote, branch string) error {
	if _, err := r.runCmd("git", "fetch"); err != nil {
		return fmt.Errorf("unable to git fetch: %s", err)
	}

	var branchExist bool
	if _, err := r.runCmd("git", "rev-parse", "--verify", branch); err == nil {
		branchExist = true
	}

	var hasUpstream bool
	if _, err := r.runCmd("git", "rev-parse", "--abbrev-ref", branch+"@{upstream}"); err == nil {
		hasUpstream = true
	}

	if branchExist {
		if hasUpstream {
			_, err := r.runCmd("git", "checkout", branch)
			if err != nil {
				return fmt.Errorf("unable to git checkout: %s", err)
			}
			return nil
		}
		// the branch exist but has no upstream. Delete it
		if _, err := r.runCmd("git", "branch", "-d", branch); err != nil {
			return fmt.Errorf("unable to git delete: %s", err)
		}
	}

	_, err := r.runCmd("git", "checkout", "-b", branch, "--track", remote+"/"+branch)
	if err != nil {
		return fmt.Errorf("unable to git checkout new branch: %s", err)
	}
	return nil
}

// Checkout checkouts a branch on the local repository
func (r Repo) Checkout(branch string) error {
	_, err := r.runCmd("git", "checkout", branch)
	if err != nil {
		return fmt.Errorf("unable to git checkout: %s", err)
	}
	return nil
}

// CheckoutNewBranch checkouts a new branch on the local repository
func (r Repo) CheckoutNewBranch(branch string) error {
	_, err := r.runCmd("git", "checkout", "-b", branch)
	if err != nil {
		return fmt.Errorf("unable to git checkout: %s", err)
	}
	return nil
}

// DeleteBranch deletes a branch on the local repository
func (r Repo) DeleteBranch(branch string) error {
	_, err := r.runCmd("git", "branch", "-d", branch)
	if err != nil {
		return fmt.Errorf("unable to delete branch: %s", err)
	}
	return nil
}

// Pull pulls a branch from a remote
func (r Repo) Pull(remote, branch string) error {
	_, err := r.runCmd("git", "pull", remote, branch)
	return err
}

// ResetHard hard resets a ref
func (r Repo) ResetHard(hash string) error {
	_, err := r.runCmd("git", "reset", "--hard", hash)
	return err
}

// DefaultBranch returns the default branch of the remote origin
func (r Repo) DefaultBranch() (string, error) {
	s, err := r.runCmd("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	if err != nil {
		return "", err
	}
	s = strings.Replace(s, "\n", "", 1)
	s = strings.Replace(s, "refs/remotes/origin/", "", 1)
	return s, nil
}

// Glob returns the matching files in the repo
func (r Repo) Glob(s string) ([]string, error) {
	p := filepath.Join(r.path, s)
	files, err := zglob.Glob(p)
	if err != nil {
		return nil, err
	}
	for i, f := range files {
		files[i], err = filepath.Rel(r.path, f)
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

// Open opens a file from the repo
func (r Repo) Open(s string) (*os.File, error) {
	p := filepath.Join(r.path, s)
	return os.Open(p)
}

// Write writes a file in the repo
func (r Repo) Write(s string, content io.Reader) error {
	p := filepath.Join(r.path, s)
	f, err := os.OpenFile(p, os.O_WRONLY, os.FileMode(0644))
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, content); err != nil {
		return err
	}
	return nil
}

// Add file contents to the index
func (r Repo) Add(s ...string) error {
	args := append([]string{"add"}, s...)
	out, err := r.runCmd("git", args...)
	if err != nil {
		return fmt.Errorf("command 'git add' failed: %v (%s)", err, out)
	}
	return nil
}

// Commit the index
func (r Repo) Commit(m string) error {
	out, err := r.runCmd("git", "commit", "-m", strconv.Quote(m))
	if err != nil {
		return fmt.Errorf("command 'git commit' failed: %v (%s)", err, out)
	}
	return nil
}

// Push (always with force) the branch
func (r Repo) Push(remote, branch string) error {
	out, err := r.runCmd("git", "push", "-f", "-u", remote, branch)
	if err != nil {
		return fmt.Errorf("command 'git push' failed: %v (%s)", err, out)
	}
	return nil
}

// Option is a function option
type Option func(r *Repo) error

// WithUser configure the git command to use user
func WithUser(email, name string) Option {
	return func(r *Repo) error {
		out, err := r.runCmd("git", "config", "user.email", email)
		if err != nil {
			return fmt.Errorf("command 'git config user.email' failed: %v (%s)", err, out)
		}
		out, err = r.runCmd("git", "config", "user.name", name)
		if err != nil {
			return fmt.Errorf("command 'git config user.name' failed: %v (%s)", err, out)
		}
		return err
	}
}

// WithSSHAuth configure the git command to use a specific private key
func WithSSHAuth(privateKey []byte) Option {
	return func(r *Repo) error {
		r.sshKey = &sshKey{
			content: privateKey,
		}

		h := md5.New()
		if _, err := io.WriteString(h, string(privateKey)); err != nil {
			return err
		}

		u, err := user.Current()
		if err != nil {
			return err
		}

		md5sum := fmt.Sprintf("%x", h.Sum(nil))
		dir := filepath.Join(u.HomeDir, ".lib-git-repo", md5sum)
		if err := os.MkdirAll(dir, os.FileMode(0700)); err != nil {
			return err
		}
		r.sshKey.filename = filepath.Join(dir, "id_rsa")
		return ioutil.WriteFile(r.sshKey.filename, r.sshKey.content, os.FileMode(0600))
	}
}

// WithHTTPAuth override the repo configuration to use http auth
func WithHTTPAuth(username string, password string) Option {
	return func(r *Repo) error {
		u, err := url.Parse(r.url)
		if err != nil {
			return err
		}
		u.User = url.UserPassword(username, password)
		r.url = u.String()
		return nil
	}
}

// InstallPGPKey install a pgp key in the repo configuration
func InstallPGPKey(privateKey []byte) Option {
	return func(r *Repo) error {
		return nil
	}
}

// WithVerbose add some logs
func WithVerbose() Option {
	return func(r *Repo) error {
		r.verbose = true
		return nil
	}
}

func (r Repo) log(format string, i ...interface{}) {
	if r.logger != nil {
		r.logger(format, i...)
	}
}
