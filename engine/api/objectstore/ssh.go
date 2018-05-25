package objectstore

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ovh/cds/sdk"
	"golang.org/x/crypto/ssh"
)

// SSHStore implements ObjectStore interface with ssh
type SSHStore struct {
	p      string
	h      string
	config ssh.ClientConfig
	d      string
	client *ssh.Client
}

//Status return filesystem storage status
func (s *SSHStore) Status() sdk.MonitoringStatusLine {
	if s.client == nil {
		return sdk.MonitoringStatusLine{Component: "Object-Store", Value: "SSH Storage (no client)", Status: sdk.MonitoringStatusAlert}
	}
	session, err := s.client.NewSession()
	if err != nil {
		return sdk.MonitoringStatusLine{Component: "Object-Store", Value: fmt.Sprintf("SSH Storage (%v)", err), Status: sdk.MonitoringStatusAlert}
	}
	session.Close()
	return sdk.MonitoringStatusLine{Component: "Object-Store", Value: "SSH Storage", Status: sdk.MonitoringStatusOK}
}

// Copies the encoded contents of an io.Reader to a remote location
func (s *SSHStore) copy(session *ssh.Session, r io.Reader, path string) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	cmd := bytes.NewBufferString("echo \"")
	cmd.WriteString(base64.StdEncoding.EncodeToString(b))
	cmd.WriteString("\" >")
	cmd.WriteString(path)

	return session.Run(cmd.String())
}

func (s *SSHStore) read(session *ssh.Session, path string) (io.ReadCloser, error) {
	output, err := session.CombinedOutput("cat " + path)
	if err != nil {
		return nil, err
	}

	output, err = base64.StdEncoding.DecodeString(string(output))
	if err != nil {
		return nil, err
	}

	return ioutil.NopCloser(bytes.NewReader(output)), nil
}

// Store store a object on disk
func (s *SSHStore) Store(o Object, data io.ReadCloser) (string, error) {
	//Create the directory
	session, err := s.client.NewSession()
	if err != nil {
		return "", err
	}

	cmd := fmt.Sprintf("mkdir -p %s/%s", s.d, o.GetPath())
	if err := session.Run(cmd); err != nil {
		session.Close()
		return "", err
	}

	session.Close()
	//Create the file
	session, err = s.client.NewSession()
	if err != nil {
		return "", err
	}

	path := o.GetPath() + "/" + o.GetName()
	if err := s.copy(session, data, path); err != nil {
		return "", err
	}

	defer session.Close()

	return path, nil
}

// Fetch lookup on disk for data
func (s *SSHStore) Fetch(o Object) (io.ReadCloser, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return nil, err
	}

	defer session.Close()
	return s.read(session, o.GetPath()+"+"+o.GetName())
}

// Delete data on disk
func (s *SSHStore) Delete(o Object) error {
	session, err := s.client.NewSession()
	if err != nil {
		return err
	}

	defer session.Close()

	return session.Run("rm -f " + o.GetPath() + "+" + o.GetName())
}
