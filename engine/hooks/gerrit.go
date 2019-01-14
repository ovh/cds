package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"golang.org/x/crypto/ssh"
	url2 "net/url"
	"strings"
	"time"
)

func ListenGerritStreamEvent(ctx context.Context, v sdk.VCSConfiguration) {
	signer, err := ssh.ParsePrivateKey([]byte(v.Password))
	if err != nil {
		log.Error("unable to read ssh key: %v", err)
	}

	// Create config
	config := &ssh.ClientConfig{
		User: v.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	url, _ := url2.Parse(v.URL)

	// Dial TCP
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", url.Hostname(), v.SSHPort), config)
	if err != nil {
		log.Error("unable to open ssh connection to gerrit: %v", err)
		return
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		log.Error("unable to create new session: %v", err)
		return
	}

	bufferOut := &bytes.Buffer{}
	bufferErr := &bytes.Buffer{}
	session.Stdout = bufferOut
	session.Stderr = bufferErr

	go func() {
		// Run command
		log.Debug("Listening to gerrit event stream %s", v.URL)
		if err := session.Run("gerrit stream-events"); err != nil {
			log.Error("unable to run gerrit stream-events command: %v", err)
		}
	}()

	tick := time.NewTicker(50 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			session.Close()
			conn.Close()
		case <-tick.C:
			if bufferOut.Len() != 0 {
				events := strings.Split(string(bufferOut.Bytes()), "\n")
				for _, e := range events {
					if e == "" {
						continue
					}
					var event GerritEvent
					if err := json.Unmarshal([]byte(e), &event); err != nil {
						log.Error("unable to read gerrit event %v: %s", err, e)
						continue
					}
					// Send event

				}
				bufferOut.Reset()
			}
		default:
		}
	}

}
