package keys

import (
	"bytes"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/ovh/cds/sdk"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

// GetOpenPGPEntity returns a single entity from an armored entity list
func GetOpenPGPEntity(r io.Reader) (*openpgp.Entity, error) {
	entityList, err := openpgp.ReadArmoredKeyRing(r)
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to read armored key ring")
	}

	if len(entityList) != 1 {
		return nil, errors.New("GetOpenPGPEntity> Invalid PGP entity list")
	}

	keys := entityList.DecryptionKeys()
	if len(keys) != 1 {
		return nil, errors.New("GetOpenPGPEntity> Invalid PGP decryption keys")
	}

	return entityList[0], nil
}

// NewOpenPGPEntity create an openpgp Entity
func NewOpenPGPEntity(keyname, comment, email string) (*openpgp.Entity, error) {
	key, errE := openpgp.NewEntity(keyname, comment, email, nil)
	if errE != nil {
		return nil, sdk.WrapError(errE, "NewOpenPGPEntity> Cannot create new entity")
	}

	if len(key.Subkeys) != 1 {
		return nil, fmt.Errorf("Wrong key generation")
	}

	OrderRSAPrimes(key.PrivateKey)
	for _, sk := range key.Subkeys {
		OrderRSAPrimes(sk.PrivateKey)
	}

	// Self sign Identity
	for _, id := range key.Identities {
		id.SelfSignature.PreferredSymmetric = []uint8{
			uint8(packet.CipherAES256),
			uint8(packet.Cipher3DES),
		}
		id.SelfSignature.PreferredHash = []uint8{
			sha512,
		}
		if err := id.SelfSignature.SignUserId(id.UserId.Id, key.PrimaryKey, key.PrivateKey, nil); err != nil {
			return nil, sdk.WrapError(err, "Cannot sign identity")
		}
	}

	return key, nil
}

// OrderRSAPrimes makes the serialized key satisfy p < q (RFC 4880 5.5.3), which libgcrypt relies on:
// with p > q, gpg sometimes produces invalid signatures. x/crypto/openpgp writes Primes[1] as p.
func OrderRSAPrimes(k *packet.PrivateKey) {
	priv, ok := k.PrivateKey.(*rsa.PrivateKey)
	if !ok || len(priv.Primes) != 2 || priv.Primes[1].Cmp(priv.Primes[0]) < 0 {
		return
	}
	priv.Primes[0], priv.Primes[1] = priv.Primes[1], priv.Primes[0]
	priv.Precomputed = rsa.PrecomputedValues{} // drop the stale CRT values, Precompute reuses them otherwise
	priv.Precompute()
}

// generatePGPPrivateKey generates a private key
func generatePGPPrivateKey(key *openpgp.Entity) (io.Reader, error) {
	bufPrivate := new(bytes.Buffer)
	w, errPrivEncode := armor.Encode(bufPrivate, openpgp.PrivateKeyType, nil)
	if errPrivEncode != nil {
		return nil, sdk.WrapError(errPrivEncode, "generatePGPPrivateKey> Cannot encode private key")
	}
	defer w.Close()
	if err := key.SerializePrivate(w, &packet.Config{}); err != nil {
		return nil, sdk.WrapError(err, "Cannot serialize private key")
	}
	return bufPrivate, nil
}

// generatePGPPublicKey generates a public key
func generatePGPPublicKey(key *openpgp.Entity) (io.Reader, error) {
	bufPublic := new(bytes.Buffer)
	w, errEncode := armor.Encode(bufPublic, openpgp.PublicKeyType, nil)
	if errEncode != nil {
		return nil, sdk.WrapError(errEncode, "generatePGPPublicKey> Cannot encode public key")
	}
	defer w.Close()
	if err := key.Serialize(w); err != nil {
		return nil, sdk.WrapError(err, "Cannot serialize public key")
	}
	return bufPublic, nil
}

// GeneratePGPKeyPair generates a private / public PGP key
func GeneratePGPKeyPair(name, comment, email string) (sdk.Key, error) {
	k := sdk.Key{
		Name: name,
		Type: sdk.KeyTypePGP,
	}
	key, err := NewOpenPGPEntity(name, comment, email)
	if err != nil {
		return k, err
	}
	k.KeyID = key.PrimaryKey.KeyIdShortString()
	k.LongKeyID = key.PrimaryKey.KeyIdString()

	bufPrivate, err := generatePGPPrivateKey(key)
	if err != nil {
		return k, err
	}

	bufPublic, err := generatePGPPublicKey(key)
	if err != nil {
		return k, err
	}

	pub, errPub := io.ReadAll(bufPublic)
	if errPub != nil {
		return k, sdk.WrapError(errPub, "GeneratePGPKeyPair> Unable to read public key")
	}

	priv, errPriv := io.ReadAll(bufPrivate)
	if errPriv != nil {
		return k, sdk.WrapError(errPriv, "GeneratePGPKeyPair>  Unable to read private key")
	}
	k.Private = string(priv)
	k.Public = string(pub)
	return k, err
}

// ReplacePGPKeyEmail rewrites an armored PGP private key so that its single user ID carries email, keeping its
// name and comment; key material and IDs are unchanged. It returns false when the key already has only that email.
func ReplacePGPKeyEmail(private, email string) (sdk.Key, bool, error) {
	entity, err := GetOpenPGPEntity(strings.NewReader(private))
	if err != nil {
		return sdk.Key{}, false, err
	}
	privateKeys := []*packet.PrivateKey{entity.PrivateKey}
	for _, sk := range entity.Subkeys {
		privateKeys = append(privateKeys, sk.PrivateKey)
	}
	for _, pk := range privateKeys {
		if pk == nil || pk.Encrypted {
			return sdk.Key{}, false, sdk.WithStack(errors.New("private key missing or encrypted"))
		}
	}

	current := primaryIdentity(entity)
	if current == nil || current.SelfSignature == nil {
		return sdk.Key{}, false, sdk.WithStack(errors.New("PGP key has no self-signed user ID"))
	}
	if len(entity.Identities) == 1 && current.UserId.Email == email {
		return sdk.Key{}, false, nil
	}
	uid := packet.NewUserId(current.UserId.Name, current.UserId.Comment, email)
	if uid == nil {
		return sdk.Key{}, false, sdk.WithStack(fmt.Errorf("invalid PGP user ID email %q", email))
	}
	fingerprints := pgpFingerprints(entity)

	// Serialization swaps the primes the parser read: order them or p > q comes back.
	for _, pk := range privateKeys {
		OrderRSAPrimes(pk)
	}
	// The new self-signature keeps the flags and preferences of the current one; only its date changes.
	sig := *current.SelfSignature
	sig.CreationTime = time.Now()
	isPrimaryID := true
	sig.IsPrimaryId = &isPrimaryID
	if err := sig.SignUserId(uid.Id, entity.PrimaryKey, entity.PrivateKey, nil); err != nil {
		return sdk.Key{}, false, sdk.WrapError(err, "cannot sign user ID")
	}
	entity.Identities = map[string]*openpgp.Identity{uid.Id: {Name: uid.Id, UserId: uid, SelfSignature: &sig}}

	bufPrivate, err := generatePGPPrivateKey(entity)
	if err != nil {
		return sdk.Key{}, false, err
	}
	bufPublic, err := generatePGPPublicKey(entity)
	if err != nil {
		return sdk.Key{}, false, err
	}
	priv, err := io.ReadAll(bufPrivate)
	if err != nil {
		return sdk.Key{}, false, sdk.WrapError(err, "unable to read private key")
	}
	pub, err := io.ReadAll(bufPublic)
	if err != nil {
		return sdk.Key{}, false, sdk.WrapError(err, "unable to read public key")
	}

	// Re-read both parts: the self-signature must verify and no key may have moved.
	for _, armored := range [][]byte{priv, pub} {
		el, err := openpgp.ReadArmoredKeyRing(bytes.NewReader(armored))
		if err != nil {
			return sdk.Key{}, false, sdk.WrapError(err, "unable to read rewritten PGP key")
		}
		if len(el) != 1 || len(el[0].Identities) != 1 || primaryIdentity(el[0]).UserId.Email != email ||
			!slices.Equal(pgpFingerprints(el[0]), fingerprints) {
			return sdk.Key{}, false, sdk.WithStack(errors.New("rewritten PGP key does not match the original one"))
		}
	}

	return sdk.Key{
		Type:      sdk.KeyTypePGP,
		KeyID:     entity.PrimaryKey.KeyIdShortString(),
		LongKeyID: entity.PrimaryKey.KeyIdString(),
		Private:   string(priv),
		Public:    string(pub),
	}, true, nil
}

// PGPKeyEmails returns the sorted emails of the user IDs of an armored PGP key.
func PGPKeyEmails(armored string) ([]string, error) {
	el, err := openpgp.ReadArmoredKeyRing(strings.NewReader(armored))
	if err != nil {
		return nil, sdk.WrapError(err, "unable to read PGP key")
	}
	if len(el) != 1 {
		return nil, sdk.WithStack(errors.New("invalid PGP entity list"))
	}
	emails := make([]string, 0, len(el[0].Identities))
	for _, id := range el[0].Identities {
		emails = append(emails, id.UserId.Email)
	}
	slices.Sort(emails)
	return emails, nil
}

// primaryIdentity returns the identity flagged primary, else the first one by name, for a stable choice.
func primaryIdentity(e *openpgp.Entity) *openpgp.Identity {
	names := slices.Sorted(maps.Keys(e.Identities))
	for _, name := range names {
		if sig := e.Identities[name].SelfSignature; sig != nil && sig.IsPrimaryId != nil && *sig.IsPrimaryId {
			return e.Identities[name]
		}
	}
	if len(names) == 0 {
		return nil
	}
	return e.Identities[names[0]]
}

// pgpFingerprints lists the fingerprints of the primary key, then of its subkeys.
func pgpFingerprints(e *openpgp.Entity) [][20]byte {
	fps := [][20]byte{e.PrimaryKey.Fingerprint}
	for _, sk := range e.Subkeys {
		fps = append(fps, sk.PublicKey.Fingerprint)
	}
	return fps
}
