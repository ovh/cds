package mssql

import (
	"database/sql"
	"fmt"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

const SecretCredsType = "creds"

func secretCreds(b *backend) *framework.Secret {
	return &framework.Secret{
		Type: SecretCredsType,
		Fields: map[string]*framework.FieldSchema{
			"username": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Username",
			},

			"password": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Password",
			},
		},

		Renew:  b.secretCredsRenew,
		Revoke: b.secretCredsRevoke,
	}
}

func (b *backend) secretCredsRenew(
	req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	// Get the lease information
	leaseConfig, err := b.LeaseConfig(req.Storage)
	if err != nil {
		return nil, err
	}
	if leaseConfig == nil {
		leaseConfig = &configLease{}
	}

	f := framework.LeaseExtend(leaseConfig.TTL, leaseConfig.TTLMax, b.System())
	return f(req, d)
}

func (b *backend) secretCredsRevoke(
	req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	// Get the username from the internal data
	usernameRaw, ok := req.Secret.InternalData["username"]
	if !ok {
		return nil, fmt.Errorf("secret is missing username internal data")
	}
	username, ok := usernameRaw.(string)

	// Get our connection
	db, err := b.DB(req.Storage)
	if err != nil {
		return nil, err
	}

	// First disable server login
	disableStmt, err := db.Prepare(fmt.Sprintf("ALTER LOGIN [%s] DISABLE;", username))
	if err != nil {
		return nil, err
	}
	defer disableStmt.Close()
	if _, err := disableStmt.Exec(); err != nil {
		return nil, err
	}

	// Query for sessions for the login so that we can kill any outstanding
	// sessions.  There cannot be any active sessions before we drop the logins
	// This isn't done in a transaction because even if we fail along the way,
	// we want to remove as much access as possible
	sessionStmt, err := db.Prepare(fmt.Sprintf(
		"SELECT session_id FROM sys.dm_exec_sessions WHERE login_name = '%s';", username))
	if err != nil {
		return nil, err
	}
	defer sessionStmt.Close()

	sessionRows, err := sessionStmt.Query()
	if err != nil {
		return nil, err
	}
	defer sessionRows.Close()

	var revokeStmts []string
	for sessionRows.Next() {
		var sessionID int
		err = sessionRows.Scan(&sessionID)
		if err != nil {
			return nil, err
		}
		revokeStmts = append(revokeStmts, fmt.Sprintf("KILL %d;", sessionID))
	}

	// Query for database users using undocumented stored procedure for now since
	// it is the easiest way to get this information;
	// we need to drop the database users before we can drop the login and the role
	// This isn't done in a transaction because even if we fail along the way,
	// we want to remove as much access as possible
	stmt, err := db.Prepare(fmt.Sprintf("EXEC sp_msloginmappings '%s';", username))
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var loginName, dbName, qUsername string
		var aliasName sql.NullString
		err = rows.Scan(&loginName, &dbName, &qUsername, &aliasName)
		if err != nil {
			return nil, err
		}
		revokeStmts = append(revokeStmts, fmt.Sprintf(dropUserSQL, dbName, username, username))
	}

	// we do not stop on error, as we want to remove as
	// many permissions as possible right now
	var lastStmtError error
	for _, query := range revokeStmts {
		stmt, err := db.Prepare(query)
		if err != nil {
			lastStmtError = err
			continue
		}
		defer stmt.Close()
		_, err = stmt.Exec()
		if err != nil {
			lastStmtError = err
		}
	}

	// can't drop if not all database users are dropped
	if rows.Err() != nil {
		return nil, fmt.Errorf("cound not generate sql statements for all rows: %s", rows.Err())
	}
	if lastStmtError != nil {
		return nil, fmt.Errorf("could not perform all sql statements: %s", lastStmtError)
	}

	// Drop this login
	stmt, err = db.Prepare(fmt.Sprintf(dropLoginSQL, username, username))
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	if _, err := stmt.Exec(); err != nil {
		return nil, err
	}

	return nil, nil
}

const dropUserSQL = `
USE [%s]
IF EXISTS
  (SELECT name
   FROM sys.database_principals
   WHERE name = N'%s')
BEGIN
  DROP USER [%s]
END
`

const dropLoginSQL = `
IF EXISTS
  (SELECT name
   FROM master.sys.server_principals
   WHERE name = N'%s')
BEGIN
  DROP LOGIN [%s]
END
`
