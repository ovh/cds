package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/engine/api/authentication/local"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/database"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"

	// Import packages that register their gorp mappings in init()
	_ "github.com/ovh/cds/engine/api/authentication"
	_ "github.com/ovh/cds/engine/api/authentication/local"
	_ "github.com/ovh/cds/engine/api/user"
)

var (
	flagCreateAdminConfigFile string
	flagCreateAdminUsername    string
	flagCreateAdminPassword   string
	flagCreateAdminEmail      string
)

var databaseCreateAdminCmd = &cobra.Command{
	Use:   "create-admin",
	Short: "Create an admin user in the database (idempotent)",
	Long: `Create an admin user directly in the CDS database.
If the user already exists, this command does nothing.
Database connection and signing keys are read from the config file.
This is designed for local development with make watch.`,
	Example: "engine database create-admin --config conf.toml --username admin --password admin",
	Run:     databaseCreateAdminCmdFunc,
}

func databaseCreateAdminCmdFunc(cmd *cobra.Command, args []string) {
	if flagCreateAdminUsername == "" || flagCreateAdminPassword == "" {
		sdk.Exit("Error: --username and --password are required\n")
	}
	if flagCreateAdminEmail == "" {
		flagCreateAdminEmail = flagCreateAdminUsername + "@localhost.local"
	}

	// Read configuration from file (same pattern as config init-token)
	conf := configImport(nil, flagCreateAdminConfigFile, "", "", "", "", true)
	if conf.API == nil {
		sdk.Exit("Error: API configuration not found in config file\n")
	}

	ctx := context.Background()

	// Initialize database connection from config
	var err error
	connFactory, err = database.Init(ctx, conf.API.Database)
	if err != nil {
		sdk.Exit("Error connecting to database: %v\n", err)
	}

	// Configure gorpmapper signing/encryption keys from config
	signatureKeyConfig := conf.API.Database.SignatureKey.GetKeys(gorpmapper.KeySignIdentifier)
	encryptionKeyConfig := conf.API.Database.EncryptionKey.GetKeys(gorpmapper.KeyEncryptionIdentifier)
	if err := gorpmapping.ConfigureKeys(signatureKeyConfig, encryptionKeyConfig); err != nil {
		sdk.Exit("Error configuring database keys: %v\n", err)
	}

	db := connFactory.DB()
	if db == nil {
		sdk.Exit("Error: cannot get database connection\n")
	}

	dbMap := database.DBMap(gorpmapping.Mapper, db)

	// Check if user already exists
	existingUser, err := user.LoadByUsername(ctx, dbMap, flagCreateAdminUsername)
	if err == nil && existingUser != nil {
		fmt.Printf("Admin user '%s' already exists, skipping creation\n", flagCreateAdminUsername)
		return
	}

	tx, err := dbMap.Begin()
	if err != nil {
		sdk.Exit("Error starting transaction: %v\n", err)
	}
	defer tx.Rollback() // nolint

	// Create the admin user
	adminUser := &sdk.AuthentifiedUser{
		Username: flagCreateAdminUsername,
		Fullname: flagCreateAdminUsername,
		Ring:     sdk.UserRingAdmin,
	}
	if err := user.Insert(ctx, tx, adminUser); err != nil {
		sdk.Exit("Error creating user: %v\n", err)
	}

	// Create email contact
	contact := &sdk.UserContact{
		UserID:  adminUser.ID,
		Type:    sdk.UserContactTypeEmail,
		Value:   flagCreateAdminEmail,
		Primary: true,
	}
	if err := user.InsertContact(ctx, tx, contact); err != nil {
		sdk.Exit("Error creating contact: %v\n", err)
	}

	// Hash password and create local auth consumer
	passwordHash, err := local.HashPassword(flagCreateAdminPassword)
	if err != nil {
		sdk.Exit("Error hashing password: %v\n", err)
	}
	if _, err := local.NewConsumerWithHash(ctx, tx, adminUser.ID, string(passwordHash)); err != nil {
		sdk.Exit("Error creating auth consumer: %v\n", err)
	}

	if err := tx.Commit(); err != nil {
		sdk.Exit("Error committing transaction: %v\n", err)
	}

	fmt.Printf("Admin user '%s' created successfully (email: %s)\n", flagCreateAdminUsername, flagCreateAdminEmail)
}
