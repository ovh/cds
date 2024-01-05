package repositories

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

var vcsPublicKeys map[string][]sdk.Key

func (s *Service) processCheckout(ctx context.Context, op *sdk.Operation) error {
	gitRepo, _, currentBranch, err := s.processGitClone(ctx, op)
	if err != nil {
		return sdk.WrapError(err, "unable to process gitclone")
	}
	log.Debug(ctx, "processCheckout> repo cloned with current branch: %s", currentBranch)

	// Clean no commited changes if exists
	if err := gitRepo.ResetHard(ctx, "HEAD"); err != nil {
		return sdk.WithStack(err)
	}
	log.Debug(ctx, "processCheckout> repo reset to HEAD")

	if op.Setup.Checkout.Tag != "" {
		log.Debug(ctx, "processCheckout> fetching tag %s from %s", op.Setup.Checkout.Tag, op.URL)
		if err := gitRepo.FetchRemoteTag(ctx, "origin", op.Setup.Checkout.Tag); err != nil {
			return sdk.WithStack(err)
		}
		log.Info(ctx, "processCheckout> repository %s ready on tag '%s'", op.URL, op.Setup.Checkout.Tag)
		return nil
	}

	if op.Setup.Checkout.Branch == "" {
		op.Setup.Checkout.Branch = op.RepositoryInfo.DefaultBranch
	}
	log.Debug(ctx, "processCheckout> fetching branch %s from %s", op.Setup.Checkout.Branch, op.URL)
	if err := gitRepo.FetchRemoteBranch(ctx, "origin", op.Setup.Checkout.Branch); err != nil {
		return sdk.WithStack(err)
	}

	// Check commit
	if op.Setup.Checkout.Commit == "" {
		// Reset HARD to the latest commit of the remote branch (don't use pull because there can be conflicts if the remote was forced)
		log.Debug(ctx, "processCheckout> resetting the branch %s from remote", op.Setup.Checkout.Branch)
		if err := gitRepo.ResetHard(ctx, "origin/"+op.Setup.Checkout.Branch); err != nil {
			return sdk.WithStack(err)
		}
	} else {
		currentCommit, err := gitRepo.LatestCommit(ctx)
		if err != nil {
			return sdk.WithStack(err)
		}
		if currentCommit.LongHash != op.Setup.Checkout.Commit {
			// Not the same commit, pull and reset HARD the commit
			log.Debug(ctx, "processCheckout> resetting the branch %s from remote", op.Setup.Checkout.Branch)
			if err := gitRepo.ResetHard(ctx, "origin/"+op.Setup.Checkout.Branch); err != nil {
				return sdk.WithStack(err)
			}

			log.Debug(ctx, "processCheckout> resetting commit %s", op.Setup.Checkout.Commit)
			if err := gitRepo.ResetHard(ctx, op.Setup.Checkout.Commit); err != nil {
				return sdk.WithStack(err)
			}
		}
	}

	if op.Setup.Checkout.ProcessSemver {
		describe, err := gitRepo.Describe(ctx, nil)
		if err != nil {
			log.ErrorWithStackTrace(ctx, errors.Wrap(err, "git describe failed"))
		} else {
			if describe.Semver != nil {
				op.Setup.Checkout.Result.Semver.Current = describe.SemverString
				op.Setup.Checkout.Result.Semver.Next = describe.Semver.IncMinor().String()
			}
		}
	}

	if op.Setup.Checkout.CheckSignature && op.Setup.Checkout.Commit != "" {
		log.Debug(ctx, "retrieve gpg key id")
		c, err := gitRepo.GetCommit(ctx, op.Setup.Checkout.Commit)
		if err != nil {
			return sdk.WithStack(err)
		}

		if c.GPGKeyID == "" {
			op.Setup.Checkout.Result.CommitVerified = false
			op.Setup.Checkout.Result.Msg = "commit not signed"
			return nil
		}
		ctx = context.WithValue(ctx, cdslog.GpgKey, c.GPGKeyID)
		op.Setup.Checkout.Result.SignKeyID = c.GPGKeyID

		// Search for public key on vcsserver
		var publicKey string
		vcsKeys, has := vcsPublicKeys[op.VCSServer]
		if has {
			for _, k := range vcsKeys {
				if k.KeyID == c.GPGKeyID {
					publicKey = k.Public
					break
				}
			}
		}

		// If not key found, try to get it from a user
		if publicKey == "" {
			// Retrieve gpg public key
			userKey, err := s.Client.UserGpgKeyGet(ctx, c.GPGKeyID)
			if err != nil {
				op.Setup.Checkout.Result.CommitVerified = false
				op.Setup.Checkout.Result.Msg = fmt.Sprintf("commit signed but key %s not found in CDS: %v", c.GPGKeyID, err)
				return nil
			}
			publicKey = userKey.PublicKey
		}

		// Import gpg public key
		fileName, _, err := sdk.ImportGPGKey(os.TempDir(), c.GPGKeyID, publicKey)
		if err != nil {
			return err
		}
		log.Debug(ctx, "key: %s, fileName: %s imported", c.GPGKeyID, fileName)

		// Check commit signature
		if err := gitRepo.VerifyCommit(ctx, op.Setup.Checkout.Commit); err != nil {
			op.Setup.Checkout.Result.CommitVerified = false
			op.Setup.Checkout.Result.Msg = fmt.Sprintf("%v", err)
			return nil
		}
		op.Setup.Checkout.Result.CommitVerified = true
	}

	log.Info(ctx, "processCheckout> repository %s ready", op.URL)
	return nil
}
