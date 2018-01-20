package keysutil

import (
	"reflect"
	"testing"

	"github.com/hashicorp/vault/logical"
)

var (
	keysArchive []KeyEntry
)

func resetKeysArchive() {
	keysArchive = []KeyEntry{KeyEntry{}}
}

func Test_KeyUpgrade(t *testing.T) {
	testKeyUpgradeCommon(t, NewLockManager(false))
	testKeyUpgradeCommon(t, NewLockManager(true))
}

func testKeyUpgradeCommon(t *testing.T, lm *LockManager) {
	storage := &logical.InmemStorage{}
	p, lock, upserted, err := lm.GetPolicyUpsert(PolicyRequest{
		Storage: storage,
		KeyType: KeyType_AES256_GCM96,
		Name:    "test",
	})
	if lock != nil {
		defer lock.RUnlock()
	}
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("nil policy")
	}
	if !upserted {
		t.Fatal("expected an upsert")
	}

	testBytes := make([]byte, len(p.Keys[1].AESKey))
	copy(testBytes, p.Keys[1].AESKey)

	p.Key = p.Keys[1].AESKey
	p.Keys = nil
	p.MigrateKeyToKeysMap()
	if p.Key != nil {
		t.Fatal("policy.Key is not nil")
	}
	if len(p.Keys) != 1 {
		t.Fatal("policy.Keys is the wrong size")
	}
	if !reflect.DeepEqual(testBytes, p.Keys[1].AESKey) {
		t.Fatal("key mismatch")
	}
}

func Test_ArchivingUpgrade(t *testing.T) {
	testArchivingUpgradeCommon(t, NewLockManager(false))
	testArchivingUpgradeCommon(t, NewLockManager(true))
}

func testArchivingUpgradeCommon(t *testing.T, lm *LockManager) {
	resetKeysArchive()

	// First, we generate a policy and rotate it a number of times. Each time
	// we'll ensure that we have the expected number of keys in the archive and
	// the main keys object, which without changing the min version should be
	// zero and latest, respectively

	storage := &logical.InmemStorage{}
	p, lock, _, err := lm.GetPolicyUpsert(PolicyRequest{
		Storage: storage,
		KeyType: KeyType_AES256_GCM96,
		Name:    "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if p == nil || lock == nil {
		t.Fatal("nil policy or lock")
	}
	lock.RUnlock()

	// Store the initial key in the archive
	keysArchive = append(keysArchive, p.Keys[1])
	checkKeys(t, p, storage, "initial", 1, 1, 1)

	for i := 2; i <= 10; i++ {
		err = p.Rotate(storage)
		if err != nil {
			t.Fatal(err)
		}
		keysArchive = append(keysArchive, p.Keys[i])
		checkKeys(t, p, storage, "rotate", i, i, i)
	}

	// Now, wipe the archive and set the archive version to zero
	err = storage.Delete("archive/test")
	if err != nil {
		t.Fatal(err)
	}
	p.ArchiveVersion = 0

	// Store it, but without calling persist, so we don't trigger
	// handleArchiving()
	buf, err := p.Serialize()
	if err != nil {
		t.Fatal(err)
	}

	// Write the policy into storage
	err = storage.Put(&logical.StorageEntry{
		Key:   "policy/" + p.Name,
		Value: buf,
	})
	if err != nil {
		t.Fatal(err)
	}

	// If we're caching, expire from the cache since we modified it
	// under-the-hood
	if lm.CacheActive() {
		delete(lm.cache, "test")
	}

	// Now get the policy again; the upgrade should happen automatically
	p, lock, err = lm.GetPolicyShared(storage, "test")
	if err != nil {
		t.Fatal(err)
	}
	if p == nil || lock == nil {
		t.Fatal("nil policy or lock")
	}
	lock.RUnlock()

	checkKeys(t, p, storage, "upgrade", 10, 10, 10)

	// Let's check some deletion logic while we're at it

	// The policy should be in there
	if lm.CacheActive() && lm.cache["test"] == nil {
		t.Fatal("nil policy in cache")
	}

	// First we'll do this wrong, by not setting the deletion flag
	err = lm.DeletePolicy(storage, "test")
	if err == nil {
		t.Fatal("got nil error, but should not have been able to delete since we didn't set the deletion flag on the policy")
	}

	// The policy should still be in there
	if lm.CacheActive() && lm.cache["test"] == nil {
		t.Fatal("nil policy in cache")
	}

	p, lock, err = lm.GetPolicyShared(storage, "test")
	if err != nil {
		t.Fatal(err)
	}
	if p == nil || lock == nil {
		t.Fatal("policy or lock nil after bad delete")
	}
	lock.RUnlock()

	// Now do it properly
	p.DeletionAllowed = true
	err = p.Persist(storage)
	if err != nil {
		t.Fatal(err)
	}
	err = lm.DeletePolicy(storage, "test")
	if err != nil {
		t.Fatal(err)
	}

	// The policy should *not* be in there
	if lm.CacheActive() && lm.cache["test"] != nil {
		t.Fatal("non-nil policy in cache")
	}

	p, lock, err = lm.GetPolicyShared(storage, "test")
	if err != nil {
		t.Fatal(err)
	}
	if p != nil || lock != nil {
		t.Fatal("policy or lock not nil after delete")
	}
}

func Test_Archiving(t *testing.T) {
	testArchivingCommon(t, NewLockManager(false))
	testArchivingCommon(t, NewLockManager(true))
}

func testArchivingCommon(t *testing.T, lm *LockManager) {
	resetKeysArchive()

	// First, we generate a policy and rotate it a number of times. Each time // we'll ensure that we have the expected number of keys in the archive and
	// the main keys object, which without changing the min version should be
	// zero and latest, respectively

	storage := &logical.InmemStorage{}
	p, lock, _, err := lm.GetPolicyUpsert(PolicyRequest{
		Storage: storage,
		KeyType: KeyType_AES256_GCM96,
		Name:    "test",
	})
	if lock != nil {
		defer lock.RUnlock()
	}
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("nil policy")
	}

	// Store the initial key in the archive
	keysArchive = append(keysArchive, p.Keys[1])
	checkKeys(t, p, storage, "initial", 1, 1, 1)

	for i := 2; i <= 10; i++ {
		err = p.Rotate(storage)
		if err != nil {
			t.Fatal(err)
		}
		keysArchive = append(keysArchive, p.Keys[i])
		checkKeys(t, p, storage, "rotate", i, i, i)
	}

	// Move the min decryption version up
	for i := 1; i <= 10; i++ {
		p.MinDecryptionVersion = i

		err = p.Persist(storage)
		if err != nil {
			t.Fatal(err)
		}
		// We expect to find:
		// * The keys in archive are the same as the latest version
		// * The latest version is constant
		// * The number of keys in the policy itself is from the min
		// decryption version up to the latest version, so for e.g. 7 and
		// 10, you'd need 7, 8, 9, and 10 -- IOW, latest version - min
		// decryption version plus 1 (the min decryption version key
		// itself)
		checkKeys(t, p, storage, "minadd", 10, 10, p.LatestVersion-p.MinDecryptionVersion+1)
	}

	// Move the min decryption version down
	for i := 10; i >= 1; i-- {
		p.MinDecryptionVersion = i

		err = p.Persist(storage)
		if err != nil {
			t.Fatal(err)
		}
		// We expect to find:
		// * The keys in archive are never removed so same as the latest version
		// * The latest version is constant
		// * The number of keys in the policy itself is from the min
		// decryption version up to the latest version, so for e.g. 7 and
		// 10, you'd need 7, 8, 9, and 10 -- IOW, latest version - min
		// decryption version plus 1 (the min decryption version key
		// itself)
		checkKeys(t, p, storage, "minsub", 10, 10, p.LatestVersion-p.MinDecryptionVersion+1)
	}
}

func checkKeys(t *testing.T,
	p *Policy,
	storage logical.Storage,
	action string,
	archiveVer, latestVer, keysSize int) {

	// Sanity check
	if len(keysArchive) != latestVer+1 {
		t.Fatalf("latest expected key version is %d, expected test keys archive size is %d, "+
			"but keys archive is of size %d", latestVer, latestVer+1, len(keysArchive))
	}

	archive, err := p.LoadArchive(storage)
	if err != nil {
		t.Fatal(err)
	}

	badArchiveVer := false
	if archiveVer == 0 {
		if len(archive.Keys) != 0 || p.ArchiveVersion != 0 {
			badArchiveVer = true
		}
	} else {
		// We need to subtract one because we have the indexes match key
		// versions, which start at 1. So for an archive version of 1, we
		// actually have two entries -- a blank 0 entry, and the key at spot 1
		if archiveVer != len(archive.Keys)-1 || archiveVer != p.ArchiveVersion {
			badArchiveVer = true
		}
	}
	if badArchiveVer {
		t.Fatalf(
			"expected archive version %d, found length of archive keys %d and policy archive version %d",
			archiveVer, len(archive.Keys), p.ArchiveVersion,
		)
	}

	if latestVer != p.LatestVersion {
		t.Fatalf(
			"expected latest version %d, found %d",
			latestVer, p.LatestVersion,
		)
	}

	if keysSize != len(p.Keys) {
		t.Fatalf(
			"expected keys size %d, found %d, action is %s, policy is \n%#v\n",
			keysSize, len(p.Keys), action, p,
		)
	}

	for i := p.MinDecryptionVersion; i <= p.LatestVersion; i++ {
		if _, ok := p.Keys[i]; !ok {
			t.Fatalf(
				"expected key %d, did not find it in policy keys", i,
			)
		}
	}

	for i := p.MinDecryptionVersion; i <= p.LatestVersion; i++ {
		if !reflect.DeepEqual(p.Keys[i], keysArchive[i]) {
			t.Fatalf("key %d not equivalent between policy keys and test keys archive", i)
		}
	}

	for i := 1; i < len(archive.Keys); i++ {
		if !reflect.DeepEqual(archive.Keys[i].AESKey, keysArchive[i].AESKey) {
			t.Fatalf("key %d not equivalent between policy archive and test keys archive", i)
		}
	}
}
