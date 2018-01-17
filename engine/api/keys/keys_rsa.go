package keys

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/ovh/cds/sdk"

	"golang.org/x/crypto/ssh"
)

// Values from https://tools.ietf.org/html/rfc4880#section-9
const (
	sha512 = 10
)

// generateSSHKeyPair generates a RSA private / public key, 4096 bits
func generateSSHKeyPair(keyname string) (pub io.Reader, priv io.Reader, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	var privb = new(bytes.Buffer)
	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	if err := pem.Encode(privb, privateKeyPEM); err != nil {
		return nil, nil, err
	}

	// generate and write public key
	pubkey, err := getSSHPublicKey(keyname, privateKey)
	if err != nil {
		return nil, nil, err
	}

	return pubkey, privb, err
}

//getSSHPrivateKey returns the RSA private key
func getSSHPrivateKey(r io.Reader) (*rsa.PrivateKey, error) {
	privBytes, errr := ioutil.ReadAll(r)
	if errr != nil {
		return nil, sdk.WrapError(errr, "getSSHPrivateKey> Unable to read private key")
	}

	privBlock, _ := pem.Decode(privBytes)
	if privBlock == nil {
		return nil, sdk.WrapError(errors.New("No Block found"), "getSSHPrivateKey> Unable to decode PEM private key")
	}
	if privBlock.Type != "RSA PRIVATE KEY" {
		return nil, sdk.WrapError(errors.New("Unsupported Key type"), "getSSHPrivateKey> Unable to decode PEM private key")
	}
	//Parse the block
	key, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
	if err != nil {
		return nil, sdk.WrapError(err, "GetSSHPrivateKey> Unable to parse PKCS1 private key")
	}

	return key, nil
}

//getSSHPublicKey returns the public key from a private key
func getSSHPublicKey(name string, privateKey *rsa.PrivateKey) (io.Reader, error) {
	// generate and write public key
	pubkey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	pub := string(ssh.MarshalAuthorizedKey(pubkey))
	// add label to public key
	pub = fmt.Sprintf("%s %s@cds", pub, name)
	return strings.NewReader(pub), nil
}

// GenerateSSHKey Generate a new ssh key
func GenerateSSHKey(name string) (sdk.Key, error) {
	k := sdk.Key{
		Name: name,
		Type: sdk.KeyTypeSSH,
	}
	pubR, privR, errGenerate := generateSSHKeyPair(name)
	if errGenerate != nil {
		return k, sdk.WrapError(errGenerate, "getSSHPublicKey> Cannot generate sshKey")
	}
	pub, errPub := ioutil.ReadAll(pubR)
	if errPub != nil {
		return k, sdk.WrapError(errPub, "getSSHPublicKey> Unable to read public key")
	}

	priv, errPriv := ioutil.ReadAll(privR)
	if errPriv != nil {
		return k, sdk.WrapError(errPriv, "getSSHPublicKey> Unable to read private key")
	}
	k.Public = string(pub)
	k.Private = string(priv)
	return k, nil
}
