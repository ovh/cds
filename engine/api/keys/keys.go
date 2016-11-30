package keys

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"golang.org/x/crypto/ssh"
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
