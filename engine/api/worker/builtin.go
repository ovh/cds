package worker

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/sdk"
)

// CreateBuiltinActions add builtin actions in database if needed
func CreateBuiltinActions(db *gorp.DbMap) error {
	// ----------------------------------- Script ---------------------------
	script := sdk.NewAction(sdk.ScriptAction)
	script.Type = sdk.BuiltinAction
	script.Description = `CDS Builtin Action.
Execute a script, written in script attribute.`
	script.Parameter(sdk.Parameter{
		Name: "script",
		Description: `Content of your script.
You can put #!/bin/bash, or #!/bin/perl at first line.
Make sure that the binary used is in
the pre-requisites of action`,
		Type: sdk.TextParameter})
	if err := checkBuiltinAction(db, script); err != nil {
		return err
	}

	// ----------------------------------- Notif  ---------------------------
	notif := sdk.NewAction(sdk.NotifAction)
	notif.Description = `CDS Builtin Action. This action can be used to send
information to notification systems.
Each notification system can interpret this
notification as desired.

You can write content in a file, using messagefile
attribute.

Consult documentation for more information`
	notif.Type = sdk.BuiltinAction
	notif.Parameter(sdk.Parameter{
		Name:  "destination",
		Value: "all",
		Description: `Destination of notification: email, tat, slack, jabber...
Check CDS Documentation for available type. Default to all`,
		Type: sdk.StringParameter})
	notif.Parameter(sdk.Parameter{
		Name:        "title",
		Description: "Title of notification",
		Type:        sdk.StringParameter})
	notif.Parameter(sdk.Parameter{
		Name:        "message",
		Description: "Message of notification (optional)",
		Type:        sdk.TextParameter})
	notif.Parameter(sdk.Parameter{
		Name:        "messagefile",
		Value:       "",
		Description: "Message could be in this file (optional)",
		Type:        sdk.StringParameter})

	if err := checkBuiltinAction(db, notif); err != nil {
		return err
	}

	// ----------------------------------- JUnit    ---------------------------
	junit := sdk.NewAction(sdk.JUnitAction)
	junit.Type = sdk.BuiltinAction
	junit.Description = `CDS Builtin Action.
Parse given file to extract Unit Test results.`
	junit.Parameter(sdk.Parameter{
		Name:        "path",
		Description: `Path to junit xml file.`,
		Type:        sdk.TextParameter})
	if err := checkBuiltinAction(db, junit); err != nil {
		return err
	}

	return nil
}

// checkBuiltinAction add builtin actions in database if needed
func checkBuiltinAction(db *gorp.DbMap, a *sdk.Action) error {
	var name string
	query := `SELECT action.name FROM action WHERE action.name = $1`

	// Check Script action
	err := db.QueryRow(query, a.Name).Scan(&name)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if err != nil && err == sql.ErrNoRows {
		err = createBuiltinAction(db, a)
		if err != nil {
			return err
		}
	}

	return nil
}

func createBuiltinAction(db *gorp.DbMap, a *sdk.Action) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = action.InsertAction(tx, a, true)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// CreateBuiltinEnvironments creates default environment if needed
func CreateBuiltinEnvironments(db gorp.SqlExecutor) error {
	return environment.CheckDefaultEnv(db)
}
