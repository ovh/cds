package keys

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"golang.org/x/crypto/ssh"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

func generatekeypair(keyname string) (pub string, priv string, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", "", err
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

	pub = string(ssh.MarshalAuthorizedKey(pubkey))
	// add label to public key
	pub = fmt.Sprintf("%s %s@cds", pub, keyname)
	priv = privb.String()

	return
}

// AddKeyPairToProject generate a ssh key pair and add them as project variables
func AddKeyPairToProject(db database.QueryExecuter, proj *sdk.Project, keyname string) error {

	pub, priv, err := generatekeypair(keyname)
	if err != nil {
		return err
	}

	v := sdk.Variable{
		Name:  keyname,
		Type:  sdk.KeyVariable,
		Value: priv,
	}
	err = project.InsertVariableInProject(db, proj, v)
	if err != nil {
		return err
	}

	p := sdk.Variable{
		Name:  keyname + ".pub",
		Type:  sdk.TextVariable,
		Value: pub,
	}
	err = project.InsertVariableInProject(db, proj, p)
	if err != nil {
		return err
	}

	return nil
}

// AddKeyPairToApplication generate a ssh key pair and add them as application variables
func AddKeyPairToApplication(db database.QueryExecuter, app *sdk.Application, keyname string) error {
	pub, priv, err := generatekeypair(keyname)
	if err != nil {
		return err
	}

	v := sdk.Variable{
		Name:  keyname,
		Type:  sdk.KeyVariable,
		Value: priv,
	}
	err = application.InsertVariable(db, app, v)
	if err != nil {
		return err
	}

	p := sdk.Variable{
		Name:  keyname + ".pub",
		Type:  sdk.TextVariable,
		Value: pub,
	}
	err = application.InsertVariable(db, app, p)
	if err != nil {
		return err
	}

	return nil
}

// AddKeyPairToEnvironment generate a ssh key pair and add them as env variables
func AddKeyPairToEnvironment(db database.QueryExecuter, envID int64, keyname string) error {
	pub, priv, err := generatekeypair(keyname)
	if err != nil {
		return err
	}

	v := &sdk.Variable{
		Name:  keyname,
		Type:  sdk.KeyVariable,
		Value: priv,
	}
	err = environment.InsertVariable(db, envID, v)
	if err != nil {
		return err
	}

	p := &sdk.Variable{
		Name:  keyname + ".pub",
		Type:  sdk.TextVariable,
		Value: pub,
	}
	err = environment.InsertVariable(db, envID, p)
	if err != nil {
		return err
	}
	return nil
}
