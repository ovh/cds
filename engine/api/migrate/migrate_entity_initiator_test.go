package migrate

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/gorpmapper"
	enginetest "github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

// legacyEntityRow writes entity rows the way versions without the initiator column did.
type legacyEntityRow struct {
	sdk.Entity
	gorpmapper.SignedEntity
}

func (e legacyEntityRow) Canonical() gorpmapper.CanonicalForms {
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.Name}}{{.ProjectKey}}{{.ProjectRepositoryID}}{{.Type}}{{.Ref}}{{.Commit}}{{md5sum .Data}}",
	}
}

func init() {
	gorpmapping.Register(gorpmapping.New(legacyEntityRow{}, "entity", false, "id"))
}

func newEntityMigrationFixture(t *testing.T) (*enginetest.FakeTransaction, *sdk.Project, *sdk.ProjectRepository, *sdk.AuthentifiedUser) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	vcsProject := assets.InsertTestVCSProject(t, db, proj.ID, "vcs-server", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsProject.ID, "my-repo")
	u, _ := assets.InsertLambdaUser(t, db)
	return db, proj, repo, u
}

func newEntityRow(proj *sdk.Project, repo *sdk.ProjectRepository, name, commit string, head bool) sdk.Entity {
	return sdk.Entity{
		ID:                  sdk.UUID(),
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkflow,
		FilePath:            ".cds/workflows/" + name + ".yml",
		Name:                name,
		Commit:              commit,
		Ref:                 "refs/heads/master",
		Data:                "name: " + name,
		Head:                head,
		LastUpdate:          time.Now(),
	}
}

func insertLegacyEntityRow(t *testing.T, db *enginetest.FakeTransaction, e sdk.Entity) sdk.Entity {
	require.NoError(t, gorpmapping.InsertAndSign(context.TODO(), db, &legacyEntityRow{Entity: e}))
	return e
}

func TestMigrateEntityInitiators_RebuildsOwnersAndSkipsTheRest(t *testing.T) {
	db, proj, repo, u := newEntityMigrationFixture(t)
	ctx := context.TODO()
	attacker, _ := assets.InsertLambdaUser(t, db)

	userID := u.ID
	owned := newEntityRow(proj, repo, "owned", "abcdef", true)
	owned.DeprecatedUserID = &userID
	insertLegacyEntityRow(t, db, owned)
	oldVersion := newEntityRow(proj, repo, "owned", "123456", false)
	oldVersion.DeprecatedUserID = &userID
	oldVersion.LastUpdate = time.Now().Add(-72 * time.Hour)
	insertLegacyEntityRow(t, db, oldVersion)
	unowned := insertLegacyEntityRow(t, db, newEntityRow(proj, repo, "unowned", "abcdef", true))
	deletedUserID := sdk.UUID()
	orphan := newEntityRow(proj, repo, "orphan", "abcdef", true)
	orphan.DeprecatedUserID = &deletedUserID
	insertLegacyEntityRow(t, db, orphan)
	migrated := newEntityRow(proj, repo, "migrated", "abcdef", true)
	migrated.Initiator = &sdk.V2Initiator{VCS: "vcs-server", VCSUsername: "octocat"}
	require.NoError(t, entity.Insert(ctx, db, &migrated))

	require.NoError(t, migrateEntityInitiators(ctx, db.DbMap, []string{owned.ID, oldVersion.ID, unowned.ID, orphan.ID, migrated.ID, sdk.UUID()}))

	// The owner carries the same user snapshot as a run initiator
	uWithContacts, err := user.LoadByID(ctx, db, u.ID, user.LoadOptions.WithContacts)
	require.NoError(t, err)
	e, err := entity.LoadByID(ctx, db, owned.ID)
	require.NoError(t, err)
	require.Equal(t, u.ID, e.Initiator.UserID)
	require.Equal(t, u.Username, e.Initiator.User.Username)
	require.Equal(t, uWithContacts.Initiator().Email, e.Initiator.User.Email)
	require.Equal(t, u.ID, *e.DeprecatedUserID)
	require.True(t, e.Head)

	// A user deleted since then keeps its id, with an empty snapshot
	e, err = entity.LoadByID(ctx, db, orphan.ID)
	require.NoError(t, err)
	require.Equal(t, deletedUserID, e.Initiator.UserID)
	require.NotNil(t, e.Initiator.User)
	require.Empty(t, e.Initiator.Username())

	// A historical row keeps its last update date, so the as-code retention still applies to it
	e, err = entity.LoadByID(ctx, db, oldVersion.ID)
	require.NoError(t, err)
	require.Equal(t, u.ID, e.Initiator.UserID)
	require.False(t, e.Head)
	require.WithinDuration(t, oldVersion.LastUpdate, e.LastUpdate, time.Second)

	e, err = entity.LoadByID(ctx, db, unowned.ID)
	require.NoError(t, err)
	require.NotNil(t, e.Initiator)
	require.True(t, e.Initiator.IsUnknown())
	require.Nil(t, e.DeprecatedUserID)

	e, err = entity.LoadByID(ctx, db, migrated.ID)
	require.NoError(t, err)
	require.Equal(t, "octocat", e.Initiator.VCSUsername)

	ids, err := entity.LoadIDsWithoutInitiator(ctx, db)
	require.NoError(t, err)
	for _, id := range []string{owned.ID, oldVersion.ID, unowned.ID, orphan.ID, migrated.ID} {
		require.NotContains(t, ids, id)
	}

	// The rebuilt owner is covered by the new signature
	_, err = db.Exec(`UPDATE entity SET initiator = jsonb_set(initiator, '{user_id}', to_jsonb($1::text)) WHERE id = $2`, attacker.ID, owned.ID)
	require.NoError(t, err)
	_, err = entity.LoadByID(ctx, db, owned.ID)
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound), "got %v", err)
}

func TestMigrateEntityInitiator_WholeTable(t *testing.T) {
	db, proj, repo, u := newEntityMigrationFixture(t)
	ctx := context.TODO()

	userID := u.ID
	head := newEntityRow(proj, repo, "wf", "abcdef", true)
	head.DeprecatedUserID = &userID
	insertLegacyEntityRow(t, db, head)
	old := insertLegacyEntityRow(t, db, newEntityRow(proj, repo, "wf", "123456", false))

	require.NoError(t, MigrateEntityInitiator(ctx, db.DbMap))

	e, err := entity.LoadByID(ctx, db, head.ID)
	require.NoError(t, err)
	require.Equal(t, u.ID, e.Initiator.UserID)

	e, err = entity.LoadByID(ctx, db, old.ID)
	require.NoError(t, err)
	require.True(t, e.Initiator.IsUnknown())

	ids, err := entity.LoadIDsWithoutInitiator(ctx, db)
	require.NoError(t, err)
	require.NotContains(t, ids, head.ID)
	require.NotContains(t, ids, old.ID)
}

func TestLegacyOwners_LoadsEachUserOnce(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)
	ctx := context.TODO()
	u, _ := assets.InsertLambdaUser(t, db)
	owners := newLegacyOwners()

	nobody, err := owners.owner(ctx, db, nil)
	require.NoError(t, err)
	require.True(t, nobody.IsUnknown())
	require.Empty(t, owners.users)

	first, err := owners.owner(ctx, db, &u.ID)
	require.NoError(t, err)
	second, err := owners.owner(ctx, db, &u.ID)
	require.NoError(t, err)
	require.Len(t, owners.users, 1)
	require.Equal(t, u.Username, first.User.Username)
	require.Equal(t, first.User, second.User)
	require.NotSame(t, first.User, second.User)

	deletedUserID := sdk.UUID()
	orphan, err := owners.owner(ctx, db, &deletedUserID)
	require.NoError(t, err)
	require.Equal(t, deletedUserID, orphan.UserID)
	require.Equal(t, &sdk.V2InitiatorUser{}, orphan.User)
	require.Len(t, owners.users, 2)
}
