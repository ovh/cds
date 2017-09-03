package keys

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
	"golang.org/x/crypto/ssh"

	"github.com/ovh/cds/sdk"
)

// Values from https://tools.ietf.org/html/rfc4880#section-9
const (
	sha512 = 10
)

// Generatekeypair generates a RSA private / public key, 4096 bits
func Generatekeypair(keyname string) (string, string, error) {
	privateKey, errGenerate := rsa.GenerateKey(rand.Reader, 4096)
	if errGenerate != nil {
		return "", "", errGenerate
	}

	var privb bytes.Buffer
	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	if err := pem.Encode(&privb, privateKeyPEM); err != nil {
		return "", "", err
	}
	// generate and write public key
	pubkey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}

	pub := string(ssh.MarshalAuthorizedKey(pubkey))
	// add label to public key
	pub = fmt.Sprintf("%s %s@cds", pub, keyname)
	priv := privb.String()

	return pub, priv, err
}

// GeneratePGPKeyPair generates a private / public PGP key
func GeneratePGPKeyPair(keyname string) (string, string, string, error) {
	key, errE := openpgp.NewEntity(keyname, keyname, "cds@locahost", nil)
	if errE != nil {
		return "", "", "", sdk.WrapError(errE, "GenerateGPGKeyPair> Cannot create new entity")
	}

	if len(key.Subkeys) != 1 {
		return "", "", "", fmt.Errorf("Wrong key generation")
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
			return "", "", "", sdk.WrapError(err, "GenerateGPGKeyPair> Cannot sign identity")
		}
	}
	// Self-sign the Subkeys
	for _, subkey := range key.Subkeys {
		if err := subkey.Sig.SignKey(subkey.PublicKey, key.PrivateKey, nil); err != nil {
			return "", "", "", sdk.WrapError(err, "GenerateGPGKeyPair> Cannot sign key")
		}
	}

	bufPrivate := new(bytes.Buffer)
	encodePrivate, errPrivEncode := armor.Encode(bufPrivate, openpgp.PrivateKeyType, nil)
	if errPrivEncode != nil {
		return "", "", "", sdk.WrapError(errPrivEncode, "GenerateGPGKeyPair> Cannot encode private key")
	}
	key.SerializePrivate(encodePrivate, &packet.Config{})
	encodePrivate.Close()

	bufPublic := new(bytes.Buffer)
	w, errEncode := armor.Encode(bufPublic, openpgp.PublicKeyType, nil)
	if errEncode != nil {
		return "", "", "", sdk.WrapError(errEncode, "GenerateGPGKeyPair> Cannot encode public key key")
	}
	key.Serialize(w)
	w.Close()

	return key.PrimaryKey.KeyIdShortString(), bufPublic.String(), bufPrivate.String(), nil
}
