package migrate

import (
	"bytes"
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"

	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

const MigratePGPKeyPrimesName = "MigratePGPKeyPrimes"

type pgpKeyCounts struct {
	fixed, skipped, failed int
}

// MigratePGPKeyPrimes rewrites the project PGP private keys stored with an RSA p > q, which gpg sometimes
// signs wrongly with. Public keys and fingerprints do not change.
func MigratePGPKeyPrimes(ctx context.Context, db *gorp.DbMap) error {
	refs, err := project.LoadPGPKeyRefs(ctx, db)
	if err != nil {
		return err
	}
	log.Info(ctx, "%s: %d project keys to check", MigratePGPKeyPrimesName, len(refs))
	counts := migratePGPKeys(ctx, db, refs)
	log.Info(ctx, "%s: %d fixed, %d skipped, %d failed", MigratePGPKeyPrimesName, counts.fixed, counts.skipped, counts.failed)
	if counts.failed > 0 {
		return fmt.Errorf("%d PGP keys could not be checked or fixed", counts.failed)
	}
	return nil
}

func migratePGPKeys(ctx context.Context, db *gorp.DbMap, refs []project.PGPKeyRef) pgpKeyCounts {
	var counts pgpKeyCounts
	for _, ref := range refs {
		ctx := context.WithValue(ctx, cdslog.Project, ref.ProjectKey)
		label, fixed, err := migratePGPKey(ctx, db, ref.ID)
		key := fmt.Sprintf("project %s key %d", ref.ProjectKey, ref.ID)
		if label != "" {
			key += " " + label
		}
		switch {
		case sdk.ErrorIs(err, sdk.ErrNotFound):
			counts.skipped++
			log.Info(ctx, "%s: %s deleted meanwhile, skipped", MigratePGPKeyPrimesName, key)
		case err != nil:
			counts.failed++
			log.Error(ctx, "%s: %s: %v", MigratePGPKeyPrimesName, key, err)
		case fixed:
			counts.fixed++
			log.Info(ctx, "%s: %s: p > q, fixed", MigratePGPKeyPrimesName, key)
		default:
			counts.skipped++
			log.Info(ctx, "%s: %s: already valid, skipped", MigratePGPKeyPrimesName, key)
		}
	}
	return counts
}

// migratePGPKey loads and locks one key, rewrites it when its RSA primes have p > q and returns a label
// identifying it in logs; the transaction is committed only when the key was rewritten.
func migratePGPKey(ctx context.Context, db *gorp.DbMap, id int64) (string, bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return "", false, sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	k, err := project.LoadAndLockKeyByID(ctx, tx, id)
	if err != nil {
		return "", false, err
	}
	label := fmt.Sprintf("%s [%s]", k.Name, k.LongKeyID)
	private, fixed, err := fixPGPPrivateKeyPrimes(k.Private)
	if err != nil || !fixed {
		return label, false, err
	}
	k.Private = private
	if err := project.UpdateKey(tx, k); err != nil {
		return label, false, err
	}
	return label, true, sdk.WithStack(tx.Commit())
}

// fixPGPPrivateKeyPrimes rewrites an armored PGP private key holding an RSA key with p > q; it returns
// false when the key needs no change. Only the secret part changes: the public key is checked to stay identical.
func fixPGPPrivateKeyPrimes(private string) (string, bool, error) {
	entity, err := keys.GetOpenPGPEntity(strings.NewReader(private))
	if err != nil {
		return "", false, err
	}
	privateKeys, err := pgpPrivateKeys(entity)
	if err != nil {
		return "", false, err
	}
	if !hasMisorderedRSAPrimes(privateKeys) {
		return private, false, nil
	}
	publicBefore, err := armoredPGPPublicKey(entity)
	if err != nil {
		return "", false, err
	}

	// The serialization swaps the primes read by the parser, so every key is ordered, valid ones included.
	for _, pk := range privateKeys {
		keys.OrderRSAPrimes(pk)
	}
	fixed, err := armoredPGPPrivateKey(entity)
	if err != nil {
		return "", false, err
	}

	fixedEntity, err := keys.GetOpenPGPEntity(strings.NewReader(fixed))
	if err != nil {
		return "", false, err
	}
	fixedPrivateKeys, err := pgpPrivateKeys(fixedEntity)
	if err != nil {
		return "", false, err
	}
	if hasMisorderedRSAPrimes(fixedPrivateKeys) {
		return "", false, sdk.WithStack(errors.New("RSA primes still have p > q after the fix"))
	}
	publicAfter, err := armoredPGPPublicKey(fixedEntity)
	if err != nil {
		return "", false, err
	}
	if publicAfter != publicBefore {
		return "", false, sdk.WithStack(errors.New("the fix would change the public key"))
	}
	return fixed, true, nil
}

// pgpPrivateKeys returns the private parts of the primary key and subkeys, all needed to rewrite the key.
func pgpPrivateKeys(entity *openpgp.Entity) ([]*packet.PrivateKey, error) {
	privateKeys := []*packet.PrivateKey{entity.PrivateKey}
	for _, sk := range entity.Subkeys {
		privateKeys = append(privateKeys, sk.PrivateKey)
	}
	for _, pk := range privateKeys {
		if pk == nil || pk.Encrypted {
			return nil, sdk.WithStack(errors.New("private key missing or encrypted"))
		}
	}
	return privateKeys, nil
}

// hasMisorderedRSAPrimes reports whether an RSA key has p > q; the parser loads p into Primes[0].
func hasMisorderedRSAPrimes(privateKeys []*packet.PrivateKey) bool {
	for _, pk := range privateKeys {
		priv, ok := pk.PrivateKey.(*rsa.PrivateKey)
		if ok && len(priv.Primes) == 2 && priv.Primes[0].Cmp(priv.Primes[1]) > 0 {
			return true
		}
	}
	return false
}

// armoredPGPPrivateKey serializes like the keys package; SerializePrivate re-signs identities and subkeys.
func armoredPGPPrivateKey(entity *openpgp.Entity) (string, error) {
	return armoredPGPKey(openpgp.PrivateKeyType, func(w io.Writer) error { return entity.SerializePrivate(w, &packet.Config{}) })
}

func armoredPGPPublicKey(entity *openpgp.Entity) (string, error) {
	return armoredPGPKey(openpgp.PublicKeyType, entity.Serialize)
}

func armoredPGPKey(blockType string, serialize func(io.Writer) error) (string, error) {
	buf := new(bytes.Buffer)
	w, err := armor.Encode(buf, blockType, nil)
	if err != nil {
		return "", sdk.WithStack(err)
	}
	if err := serialize(w); err != nil {
		return "", sdk.WithStack(err)
	}
	if err := w.Close(); err != nil {
		return "", sdk.WithStack(err)
	}
	return buf.String(), nil
}
