package mongodb

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
	"gopkg.in/mgo.v2"
)

func Factory(conf *logical.BackendConfig) (logical.Backend, error) {
	return Backend().Setup(conf)
}

func Backend() *framework.Backend {
	var b backend
	b.Backend = &framework.Backend{
		Help: strings.TrimSpace(backendHelp),

		Paths: []*framework.Path{
			pathConfigConnection(&b),
			pathConfigLease(&b),
			pathListRoles(&b),
			pathRoles(&b),
			pathCredsCreate(&b),
		},

		Secrets: []*framework.Secret{
			secretCreds(&b),
		},

		Clean: b.ResetSession,
	}

	return b.Backend
}

type backend struct {
	*framework.Backend

	session *mgo.Session
	lock    sync.Mutex
}

// Session returns the database connection.
func (b *backend) Session(s logical.Storage) (*mgo.Session, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.session != nil {
		if err := b.session.Ping(); err == nil {
			return b.session, nil
		}
		b.session.Close()
	}

	connConfigJSON, err := s.Get("config/connection")
	if err != nil {
		return nil, err
	}
	if connConfigJSON == nil {
		return nil, fmt.Errorf("configure the MongoDB connection with config/connection first")
	}

	var connConfig connectionConfig
	if err := connConfigJSON.DecodeJSON(&connConfig); err != nil {
		return nil, err
	}

	dialInfo, err := parseMongoURI(connConfig.URI)
	if err != nil {
		return nil, err
	}

	b.session, err = mgo.DialWithInfo(dialInfo)
	if err != nil {
		return nil, err
	}
	b.session.SetSyncTimeout(1 * time.Minute)
	b.session.SetSocketTimeout(1 * time.Minute)

	return b.session, nil
}

// ResetSession forces creation of a new connection next time Session() is called.
func (b *backend) ResetSession() {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.session != nil {
		b.session.Close()
	}

	b.session = nil
}

// LeaseConfig returns the lease configuration
func (b *backend) LeaseConfig(s logical.Storage) (*configLease, error) {
	entry, err := s.Get("config/lease")
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	var result configLease
	if err := entry.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

const backendHelp = `
The mongodb backend dynamically generates MongoDB credentials.

After mounting this backend, configure it using the endpoints within
the "config/" path.
`
