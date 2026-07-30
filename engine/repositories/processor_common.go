package repositories

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fsamin/go-repo"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

func (s *Service) processGitClone(ctx context.Context, op *sdk.Operation) (gitRepo repo.Repo, basedir string, currentBranch string, err error) {
	r := s.Repo(*op)
	if err := s.checkOrCreateFS(r); err != nil {
		return gitRepo, "", "", err
	}

	// Get the git repository
	opts := []repo.Option{repo.WithVerbose(func(format string, args ...interface{}) { log.Info(ctx, format, args...) })}

	if op.RepositoryStrategy.ConnectionType == "ssh" {
		log.Debug(ctx, "processGitClone> %s > using ssh key %s", op.UUID, op.RepositoryStrategy.SSHKey)
		opts = append(opts, repo.WithSSHAuth([]byte(op.RepositoryStrategy.SSHKeyContent)))
	} else if op.RepositoryStrategy.User != "" && op.RepositoryStrategy.Password != "" {
		log.Debug(ctx, "processGitClone> %s > using user %s", op.UUID, op.RepositoryStrategy.User)
		opts = append(opts, repo.WithHTTPAuth(op.RepositoryStrategy.User, op.RepositoryStrategy.Password))
	}

	gitRepo, err = repo.New(ctx, r.Basedir, opts...)
	if err != nil {
		log.Info(ctx, "processGitClone> %s > cloning %s into %s", op.UUID, r.URL, r.Basedir)
		gitRepo, err = repo.Clone(ctx, r.Basedir, r.URL, opts...)
		if err != nil {
			if strings.Contains(err.Error(), "Invalid username or password") ||
				strings.Contains(err.Error(), "Permission denied (publickey)") ||
				strings.Contains(err.Error(), "could not read Username for") ||
				strings.Contains(err.Error(), "you do not have permission to access it") {
				return gitRepo, "", "", sdk.NewError(sdk.ErrForbidden, err)
			}
			return gitRepo, "", "", sdk.NewErrorFrom(err, "cannot clone repository at given url: %s", r.URL)
		}
	}

	f, err := gitRepo.FetchURL(ctx)
	if err != nil {
		return gitRepo, "", "", sdk.WithStack(err)
	}

	d, err := gitRepo.DefaultBranch(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "you do not have permission to access it") {
			return gitRepo, "", "", sdk.NewError(sdk.ErrForbidden, err)
		}
		return gitRepo, "", "", sdk.WithStack(err)
	}

	op.RepositoryInfo = &sdk.OperationRepositoryInfo{
		Name:          op.RepoFullName,
		FetchURL:      f,
		DefaultBranch: d,
	}

	//Check branch
	currentBranch, err = gitRepo.CurrentBranch(ctx)
	if err != nil {
		return gitRepo, "", "", sdk.WithStack(err)
	}
	return gitRepo, r.Basedir, currentBranch, nil
}

// checkCommitSignature verifies the signature of the operation commit or tag:
// it resolves the signing key ID, looks up the public key (VCS keys loaded at
// startup, then user keys, then VCS users through the API), imports it and
// runs the git verification. Results are reported in op.Setup.Checkout.Result.
func (s *Service) checkCommitSignature(ctx context.Context, gitRepo repo.Repo, op *sdk.Operation) error {
	if !op.Setup.Checkout.CheckSignature || (op.Setup.Checkout.Commit == "" && op.Setup.Checkout.Tag == "") {
		return nil
	}

	var gpgKeyID string
	if op.Setup.Checkout.Tag != "" {
		log.Debug(ctx, "retrieve gpg key id from tag %s", op.Setup.Checkout.Tag)
		// Check tag signature
		t, err := gitRepo.GetTag(ctx, op.Setup.Checkout.Tag)
		if err != nil {
			return sdk.WithStack(err)
		}
		gpgKeyID = t.GPGKeyID
	} else {
		log.Debug(ctx, "retrieve gpg key id from commit %s", op.Setup.Checkout.Commit)
		c, err := gitRepo.GetCommit(ctx, op.Setup.Checkout.Commit, repo.CommitOption{DisableDiffDetail: true})
		if err != nil {
			return sdk.WithStack(err)
		}
		gpgKeyID = c.GPGKeyID
	}

	if gpgKeyID == "" {
		op.Setup.Checkout.Result.CommitVerified = false
		op.Setup.Checkout.Result.Msg = "commit not signed"
		return nil
	}

	ctx = context.WithValue(ctx, cdslog.GpgKey, gpgKeyID)
	op.Setup.Checkout.Result.SignKeyID = gpgKeyID

	// Search for public key on vcsserver
	var publicKey string
	vcsKeys, has := vcsPublicKeys[op.VCSServer]
	if has {
		for _, k := range vcsKeys {
			if k.KeyID == gpgKeyID {
				publicKey = k.Public
				break
			}
		}
	}

	// If not key found, try to get it from a user
	if publicKey == "" {
		// Retrieve gpg public key on users
		userKey, _ := s.Client.UserGpgKeyGet(ctx, gpgKeyID)
		if userKey.PublicKey != "" {
			publicKey = userKey.PublicKey
		} else {
			// Retrieve gpg public key on vcs
			vcsUsers, _ := s.Client.VCSGPGKey(ctx, gpgKeyID)
			for _, vcsUser := range vcsUsers {
				if vcsUser.VCSProjectName == op.VCSServer && vcsUser.KeyID == gpgKeyID {
					publicKey = vcsUser.PublicKey
					break
				}
			}
			if publicKey == "" {
				op.Setup.Checkout.Result.CommitVerified = false
				op.Setup.Checkout.Result.Msg = fmt.Sprintf("commit signed but key %s not found in CDS", gpgKeyID)
				return nil
			}
		}
	}

	// Import gpg public key
	fileName, _, err := sdk.ImportGPGKey(os.TempDir(), gpgKeyID, []byte(publicKey))
	if err != nil {
		return err
	}
	log.Debug(ctx, "key: %s, fileName: %s imported", gpgKeyID, fileName)

	// Check commit signature
	if op.Setup.Checkout.Tag != "" {
		if _, err := gitRepo.VerifyTag(ctx, op.Setup.Checkout.Tag); err != nil {
			op.Setup.Checkout.Result.CommitVerified = false
			op.Setup.Checkout.Result.Msg = fmt.Sprintf("%v", err)
			return nil
		}
	} else {
		if err := gitRepo.VerifyCommit(ctx, op.Setup.Checkout.Commit); err != nil {
			op.Setup.Checkout.Result.CommitVerified = false
			op.Setup.Checkout.Result.Msg = fmt.Sprintf("%v", err)
			return nil
		}
	}
	op.Setup.Checkout.Result.CommitVerified = true
	return nil
}
