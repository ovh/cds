package entity_test

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
	"github.com/ovh/cds/engine/gorpmapper"
	enginetest "github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

// legacyEntity writes rows the way former versions did: NULL initiator column and a signature that
// does not cover the owner.
type legacyEntity struct {
	sdk.Entity
	gorpmapper.SignedEntity
}

func (e legacyEntity) Canonical() gorpmapper.CanonicalForms {
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.Name}}{{.ProjectKey}}{{.ProjectRepositoryID}}{{.Type}}{{.Ref}}{{.Commit}}{{md5sum .Data}}",
	}
}

func init() {
	gorpmapping.Register(gorpmapping.New(legacyEntity{}, "entity", false, "id"))
}

type entityFixture struct {
	db   *enginetest.FakeTransaction
	proj *sdk.Project
	repo *sdk.ProjectRepository
	user *sdk.AuthentifiedUser
}

func newEntityFixture(t *testing.T) entityFixture {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	vcsProject := assets.InsertTestVCSProject(t, db, proj.ID, "vcs-server", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsProject.ID, "my-repo")
	u, _ := assets.InsertLambdaUser(t, db)
	return entityFixture{db: db, proj: proj, repo: repo, user: u}
}

func (f entityFixture) newEntity(name string) sdk.Entity {
	return sdk.Entity{
		ProjectKey:          f.proj.Key,
		ProjectRepositoryID: f.repo.ID,
		Type:                sdk.EntityTypeWorkflow,
		FilePath:            ".cds/workflows/" + name + ".yml",
		Name:                name,
		Commit:              "abcdef",
		Ref:                 "refs/heads/master",
		Data:                "name: " + name,
		Head:                true,
	}
}

func (f entityFixture) insertLegacy(t *testing.T, name string, head bool, userID *string) sdk.Entity {
	e := f.newEntity(name)
	e.ID = sdk.UUID()
	e.Head = head
	e.DeprecatedUserID = userID
	e.LastUpdate = time.Now()
	require.NoError(t, gorpmapping.InsertAndSign(context.TODO(), f.db, &legacyEntity{Entity: e}))
	return e
}

func (f entityFixture) countStoredOwner(t *testing.T, id string) int64 {
	n, err := f.db.SelectInt("SELECT COUNT(*) FROM entity WHERE id = $1 AND initiator IS NOT NULL", id)
	require.NoError(t, err)
	return n
}

func indexOf(ids []string, id string) int {
	for i := range ids {
		if ids[i] == id {
			return i
		}
	}
	return -1
}

func TestInsert_PersistsTheOwnerWithoutPrivileges(t *testing.T) {
	f := newEntityFixture(t)
	ctx := context.TODO()

	e := f.newEntity("cds-user-owner")
	e.Initiator = &sdk.V2Initiator{UserID: f.user.ID, User: f.user.Initiator(), IsAdminWithMFA: true}
	require.NoError(t, entity.Insert(ctx, f.db, &e))

	loaded, err := entity.LoadByID(ctx, f.db, e.ID)
	require.NoError(t, err)
	require.Equal(t, f.user.ID, loaded.Initiator.UserID)
	require.Equal(t, f.user.Username, loaded.Initiator.User.Username)
	require.False(t, loaded.Initiator.IsAdminWithMFA)
	require.NotNil(t, loaded.DeprecatedUserID)
	require.Equal(t, f.user.ID, *loaded.DeprecatedUserID)
}

func TestInsert_PersistsAVCSOwner(t *testing.T) {
	f := newEntityFixture(t)
	ctx := context.TODO()

	e := f.newEntity("vcs-owner")
	e.Initiator = &sdk.V2Initiator{VCS: "vcs-server", VCSUsername: "octocat"}
	require.NoError(t, entity.Insert(ctx, f.db, &e))

	loaded, err := entity.LoadByID(ctx, f.db, e.ID)
	require.NoError(t, err)
	require.False(t, loaded.Initiator.IsUnknown())
	require.False(t, loaded.Initiator.IsUser())
	require.Equal(t, "vcs-server", loaded.Initiator.VCS)
	require.Equal(t, "octocat", loaded.Initiator.VCSUsername)
	require.Nil(t, loaded.DeprecatedUserID)
}

func TestInsert_RebuildsANilOwnerFromUserID(t *testing.T) {
	f := newEntityFixture(t)
	ctx := context.TODO()

	withUser := f.newEntity("with-user-id")
	userID := f.user.ID
	withUser.DeprecatedUserID = &userID
	require.NoError(t, entity.Insert(ctx, f.db, &withUser))

	loaded, err := entity.LoadByID(ctx, f.db, withUser.ID)
	require.NoError(t, err)
	require.Equal(t, f.user.ID, loaded.Initiator.UserID)
	require.Equal(t, f.user.ID, *loaded.DeprecatedUserID)

	nobody := f.newEntity("nobody")
	require.NoError(t, entity.Insert(ctx, f.db, &nobody))

	loaded, err = entity.LoadByID(ctx, f.db, nobody.ID)
	require.NoError(t, err)
	require.NotNil(t, loaded.Initiator)
	require.True(t, loaded.Initiator.IsUnknown())
	require.Nil(t, loaded.DeprecatedUserID)
	require.EqualValues(t, 1, f.countStoredOwner(t, nobody.ID), "nobody must be stored, NULL is reserved to rows not migrated yet")
}

func TestLoad_LegacyRowHasNoOwnerButKeepsItsUserID(t *testing.T) {
	f := newEntityFixture(t)
	ctx := context.TODO()

	userID := f.user.ID
	legacy := f.insertLegacy(t, "legacy", true, &userID)

	loaded, err := entity.LoadByID(ctx, f.db, legacy.ID)
	require.NoError(t, err)
	require.Nil(t, loaded.Initiator)
	require.NotNil(t, loaded.DeprecatedUserID)
	require.Equal(t, f.user.ID, *loaded.DeprecatedUserID)

	es, err := entity.LoadHeadEntitiesByRepositoryAndRef(ctx, f.db, f.repo.ID, "refs/heads/master")
	require.NoError(t, err)
	require.Len(t, es, 1)
	require.Nil(t, es[0].Initiator)
	require.Equal(t, f.user.ID, *es[0].DeprecatedUserID)
}

func TestLoadIDsWithoutInitiator_ListsLegacyRowsHeadFirst(t *testing.T) {
	f := newEntityFixture(t)
	ctx := context.TODO()

	oldVersion := f.insertLegacy(t, "legacy", false, nil)
	headVersion := f.insertLegacy(t, "legacy-head", true, nil)
	migrated := f.newEntity("migrated")
	require.NoError(t, entity.Insert(ctx, f.db, &migrated))

	ids, err := entity.LoadIDsWithoutInitiator(ctx, f.db)
	require.NoError(t, err)
	require.Contains(t, ids, oldVersion.ID)
	require.Contains(t, ids, headVersion.ID)
	require.NotContains(t, ids, migrated.ID)
	require.Less(t, indexOf(ids, headVersion.ID), indexOf(ids, oldVersion.ID))
}

func TestUpdate_MigratesALegacyRow(t *testing.T) {
	f := newEntityFixture(t)
	ctx := context.TODO()

	userID := f.user.ID
	owned := f.insertLegacy(t, "owned", true, &userID)
	unowned := f.insertLegacy(t, "unowned", true, nil)

	loaded, err := entity.LoadByID(ctx, f.db, owned.ID)
	require.NoError(t, err)
	require.Nil(t, loaded.Initiator)
	require.NoError(t, entity.Update(ctx, f.db, loaded))
	require.Equal(t, f.user.ID, loaded.Initiator.UserID)

	reloaded, err := entity.LoadByID(ctx, f.db, owned.ID)
	require.NoError(t, err)
	require.Equal(t, f.user.ID, reloaded.Initiator.UserID)
	require.Equal(t, f.user.ID, *reloaded.DeprecatedUserID)

	loaded, err = entity.LoadByID(ctx, f.db, unowned.ID)
	require.NoError(t, err)
	require.NoError(t, entity.Update(ctx, f.db, loaded))

	reloaded, err = entity.LoadByID(ctx, f.db, unowned.ID)
	require.NoError(t, err)
	require.NotNil(t, reloaded.Initiator)
	require.True(t, reloaded.Initiator.IsUnknown())
	require.Nil(t, reloaded.DeprecatedUserID)

	ids, err := entity.LoadIDsWithoutInitiator(ctx, f.db)
	require.NoError(t, err)
	require.NotContains(t, ids, owned.ID)
	require.NotContains(t, ids, unowned.ID)
}

func TestLoad_RejectsATamperedOwner(t *testing.T) {
	f := newEntityFixture(t)
	ctx := context.TODO()
	attacker, _ := assets.InsertLambdaUser(t, f.db)

	owned := f.newEntity("owned")
	owned.Initiator = &sdk.V2Initiator{UserID: f.user.ID, User: f.user.Initiator()}
	require.NoError(t, entity.Insert(ctx, f.db, &owned))

	_, err := f.db.Exec(`UPDATE entity SET initiator = jsonb_set(initiator, '{user_id}', to_jsonb($1::text)) WHERE id = $2`, attacker.ID, owned.ID)
	require.NoError(t, err)
	_, err = entity.LoadByID(ctx, f.db, owned.ID)
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound), "an owner replaced inside the signed column must be rejected, got %v", err)

	nobody := f.newEntity("nobody")
	require.NoError(t, entity.Insert(ctx, f.db, &nobody))

	_, err = f.db.Exec(`UPDATE entity SET initiator = NULL, user_id = $1 WHERE id = $2`, attacker.ID, nobody.ID)
	require.NoError(t, err)
	_, err = entity.LoadByID(ctx, f.db, nobody.ID)
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound), "a migrated row turned back into an unmigrated one must be rejected, got %v", err)
}

func TestLoad_ServesTheSignedOwnerNotTheUnsignedColumns(t *testing.T) {
	f := newEntityFixture(t)
	ctx := context.TODO()
	attacker, _ := assets.InsertLambdaUser(t, f.db)

	owned := f.newEntity("owned")
	owned.Initiator = &sdk.V2Initiator{UserID: f.user.ID, User: f.user.Initiator()}
	require.NoError(t, entity.Insert(ctx, f.db, &owned))

	_, err := f.db.Exec(`UPDATE entity SET user_id = $1, initiator = jsonb_set(initiator, '{is_admin_with_mfa}', 'true') WHERE id = $2`, attacker.ID, owned.ID)
	require.NoError(t, err)

	loaded, err := entity.LoadByID(ctx, f.db, owned.ID)
	require.NoError(t, err)
	require.Equal(t, f.user.ID, loaded.Initiator.UserID)
	require.False(t, loaded.Initiator.IsAdminWithMFA)
	require.Equal(t, f.user.ID, *loaded.DeprecatedUserID)

	es, err := entity.LoadHeadEntitiesByRepositoryAndRef(ctx, f.db, f.repo.ID, "refs/heads/master")
	require.NoError(t, err)
	require.Len(t, es, 1)
	require.False(t, es[0].Initiator.IsAdminWithMFA)
	require.Equal(t, f.user.ID, *es[0].DeprecatedUserID)
}
