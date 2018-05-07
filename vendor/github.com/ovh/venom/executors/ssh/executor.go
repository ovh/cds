package ssh

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"golang.org/x/crypto/ssh"

	"github.com/ovh/venom"
	"github.com/ovh/venom/executors"
)

// Name for test ssh
const Name = "ssh"

// New returns a new Test Exec
func New() venom.Executor {
	return &Executor{}
}

// Executor represents a Test Exec
type Executor struct {
	Host       string `json:"host,omitempty" yaml:"host,omitempty"`
	Command    string `json:"command,omitempty" yaml:"command,omitempty"`
	User       string `json:"user,omitempty" yaml:"user,omitempty"`
	Password   string `json:"password,omitempty" yaml:"password,omitempty"`
	PrivateKey string `json:"privatekey,omitempty" yaml:"privatekey,omitempty"`
}

// Result represents a step result
type Result struct {
	Executor    Executor `json:"executor,omitempty" yaml:"executor,omitempty"`
	Systemout   string   `json:"systemout,omitempty" yaml:"systemout,omitempty"`
	Systemerr   string   `json:"systemerr,omitempty" yaml:"systemerr,omitempty"`
	Err         string   `json:"err,omitempty" yaml:"err,omitempty"`
	Code        string   `json:"code,omitempty" yaml:"code,omitempty"`
	TimeSeconds float64  `json:"timeseconds,omitempty" yaml:"timeseconds,omitempty"`
	TimeHuman   string   `json:"timehuman,omitempty" yaml:"timehuman,omitempty"`
}

// ZeroValueResult return an empty implemtation of this executor result
func (Executor) ZeroValueResult() venom.ExecutorResult {
	r, _ := executors.Dump(Result{})
	return r
}

// GetDefaultAssertions return default assertions for type exec
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []string{"result.code ShouldEqual 0"}}
}

// Run execute TestStep of type exec
func (Executor) Run(testCaseContext venom.TestCaseContext, l venom.Logger, step venom.TestStep, workdir string) (venom.ExecutorResult, error) {
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	if e.Command == "" {
		return nil, fmt.Errorf("Invalid command")
	}

	start := time.Now()
	result := Result{Executor: e}

	client, session, err := connectToHost(e.User, e.Password, e.PrivateKey, e.Host)
	if err != nil {
		result.Err = err.Error()
	} else {
		defer client.Close()
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		session.Stderr = stderr
		session.Stdout = stdout
		if err := session.Run(e.Command); err != nil {
			if exiterr, ok := err.(*ssh.ExitError); ok {
				status := exiterr.ExitStatus()
				result.Code = strconv.Itoa(status)
			} else if _, ok := err.(*ssh.ExitMissingError); ok {
				result.Code = strconv.Itoa(127)
				result.Err = err.Error()
			} else {
				result.Code = strconv.Itoa(137)
				result.Err = err.Error()
			}
		} else {
			result.Code = "0"
		}

		result.Systemerr = stderr.String()
		result.Systemout = stdout.String()
	}

	elapsed := time.Since(start)
	result.TimeSeconds = elapsed.Seconds()
	result.TimeHuman = fmt.Sprintf("%s", elapsed)

	return executors.Dump(result)
}

func connectToHost(u, pass, key, host string) (*ssh.Client, *ssh.Session, error) {
	//Default user is current username
	if u == "" {
		osUser, err := user.Current()
		if err != nil {
			return nil, nil, err
		}
		u = osUser.Username
	}

	//If password is set, use it
	var auth []ssh.AuthMethod
	if pass != "" {
		auth = []ssh.AuthMethod{ssh.Password(pass)}
	} else {
		//Load the the private key
		key, err := privateKey(key)
		if err != nil {
			return nil, nil, err
		}
		auth = []ssh.AuthMethod{ssh.PublicKeys(key)}
	}

	sshConfig := &ssh.ClientConfig{
		User: u,
		Auth: auth,
	}

	//If host doen't contain port, set the default port
	if !strings.Contains(host, ":") {
		host += ":22"
	}

	//Open the tcp connection
	client, err := ssh.Dial("tcp", host, sshConfig)
	if err != nil {
		return nil, nil, err
	}

	//New ssh session
	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, nil, err
	}

	return client, session, nil
}

func privateKey(file string) (key ssh.Signer, err error) {
	//Default private key is $HOME/.ssh/id_rsa
	if file == "" {
		usr, err := user.Current()
		if err != nil {
			return nil, err
		}
		file = usr.HomeDir + "/.ssh/id_rsa"
	}
	//Read the file
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	//Parse it
	key, err = ssh.ParsePrivateKey(buf)
	if err != nil {
		return nil, err
	}
	return key, nil
}
