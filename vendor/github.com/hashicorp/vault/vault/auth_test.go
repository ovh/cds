package vault

import (
	"reflect"
	"testing"

	"github.com/hashicorp/vault/logical"
)

func TestCore_DefaultAuthTable(t *testing.T) {
	c, key, _ := TestCoreUnsealed(t)
	verifyDefaultAuthTable(t, c.auth)

	// Start a second core with same physical
	conf := &CoreConfig{
		Physical:     c.physical,
		DisableMlock: true,
	}
	c2, err := NewCore(conf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	unseal, err := TestCoreUnseal(c2, key)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !unseal {
		t.Fatalf("should be unsealed")
	}

	// Verify matching mount tables
	if !reflect.DeepEqual(c.auth, c2.auth) {
		t.Fatalf("mismatch: %v %v", c.auth, c2.auth)
	}
}

func TestCore_EnableCredential(t *testing.T) {
	c, key, _ := TestCoreUnsealed(t)
	c.credentialBackends["noop"] = func(*logical.BackendConfig) (logical.Backend, error) {
		return &NoopBackend{}, nil
	}

	me := &MountEntry{
		Table: credentialTableType,
		Path:  "foo",
		Type:  "noop",
	}
	err := c.enableCredential(me)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	match := c.router.MatchingMount("auth/foo/bar")
	if match != "auth/foo/" {
		t.Fatalf("missing mount")
	}

	conf := &CoreConfig{
		Physical:     c.physical,
		DisableMlock: true,
	}
	c2, err := NewCore(conf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	c2.credentialBackends["noop"] = func(*logical.BackendConfig) (logical.Backend, error) {
		return &NoopBackend{}, nil
	}
	unseal, err := TestCoreUnseal(c2, key)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !unseal {
		t.Fatalf("should be unsealed")
	}

	// Verify matching auth tables
	if !reflect.DeepEqual(c.auth, c2.auth) {
		t.Fatalf("mismatch: %v %v", c.auth, c2.auth)
	}
}

func TestCore_EnableCredential_twice_409(t *testing.T) {
	c, _, _ := TestCoreUnsealed(t)
	c.credentialBackends["noop"] = func(*logical.BackendConfig) (logical.Backend, error) {
		return &NoopBackend{}, nil
	}

	me := &MountEntry{
		Table: credentialTableType,
		Path:  "foo",
		Type:  "noop",
	}
	err := c.enableCredential(me)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// 2nd should be a 409 error
	err2 := c.enableCredential(me)
	switch err2.(type) {
	case logical.HTTPCodedError:
		if err2.(logical.HTTPCodedError).Code() != 409 {
			t.Fatalf("invalid code given")
		}
	default:
		t.Fatalf("expected a different error type")
	}
}

func TestCore_EnableCredential_Token(t *testing.T) {
	c, _, _ := TestCoreUnsealed(t)
	me := &MountEntry{
		Table: credentialTableType,
		Path:  "foo",
		Type:  "token",
	}
	err := c.enableCredential(me)
	if err.Error() != "token credential backend cannot be instantiated" {
		t.Fatalf("err: %v", err)
	}
}

func TestCore_DisableCredential(t *testing.T) {
	c, key, _ := TestCoreUnsealed(t)
	c.credentialBackends["noop"] = func(*logical.BackendConfig) (logical.Backend, error) {
		return &NoopBackend{}, nil
	}

	existed, err := c.disableCredential("foo")
	if existed || err.Error() != "no matching backend" {
		t.Fatalf("existed: %v; err: %v", existed, err)
	}

	me := &MountEntry{
		Table: credentialTableType,
		Path:  "foo",
		Type:  "noop",
	}
	err = c.enableCredential(me)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	existed, err = c.disableCredential("foo")
	if !existed || err != nil {
		t.Fatalf("existed: %v; err: %v", existed, err)
	}

	match := c.router.MatchingMount("auth/foo/bar")
	if match != "" {
		t.Fatalf("backend present")
	}

	conf := &CoreConfig{
		Physical:     c.physical,
		DisableMlock: true,
	}
	c2, err := NewCore(conf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	unseal, err := TestCoreUnseal(c2, key)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !unseal {
		t.Fatalf("should be unsealed")
	}

	// Verify matching mount tables
	if !reflect.DeepEqual(c.auth, c2.auth) {
		t.Fatalf("mismatch: %v %v", c.auth, c2.auth)
	}
}

func TestCore_DisableCredential_Protected(t *testing.T) {
	c, _, _ := TestCoreUnsealed(t)
	existed, err := c.disableCredential("token")
	if !existed || err.Error() != "token credential backend cannot be disabled" {
		t.Fatalf("existed: %v; err: %v", existed, err)
	}
}

func TestCore_DisableCredential_Cleanup(t *testing.T) {
	noop := &NoopBackend{
		Login: []string{"login"},
	}
	c, _, _ := TestCoreUnsealed(t)
	c.credentialBackends["noop"] = func(*logical.BackendConfig) (logical.Backend, error) {
		return noop, nil
	}

	me := &MountEntry{
		Table: credentialTableType,
		Path:  "foo",
		Type:  "noop",
	}
	err := c.enableCredential(me)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Store the view
	view := c.router.MatchingStorageView("auth/foo/")

	// Inject data
	se := &logical.StorageEntry{
		Key:   "plstodelete",
		Value: []byte("test"),
	}
	if err := view.Put(se); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Generate a new token auth
	noop.Response = &logical.Response{
		Auth: &logical.Auth{
			Policies: []string{"foo"},
		},
	}
	r := &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "auth/foo/login",
	}
	resp, err := c.HandleRequest(r)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp.Auth.ClientToken == "" {
		t.Fatalf("bad: %#v", resp)
	}

	// Disable should cleanup
	existed, err := c.disableCredential("foo")
	if !existed || err != nil {
		t.Fatalf("existed: %v; err: %v", existed, err)
	}

	// Token should be revoked
	te, err := c.tokenStore.Lookup(resp.Auth.ClientToken)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if te != nil {
		t.Fatalf("bad: %#v", te)
	}

	// View should be empty
	out, err := logical.CollectKeys(view)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("bad: %#v", out)
	}
}

func TestDefaultAuthTable(t *testing.T) {
	table := defaultAuthTable()
	verifyDefaultAuthTable(t, table)
}

func verifyDefaultAuthTable(t *testing.T, table *MountTable) {
	if len(table.Entries) != 1 {
		t.Fatalf("bad: %v", table.Entries)
	}
	if table.Type != credentialTableType {
		t.Fatalf("bad: %v", *table)
	}
	for idx, entry := range table.Entries {
		switch idx {
		case 0:
			if entry.Path != "token/" {
				t.Fatalf("bad: %v", entry)
			}
			if entry.Type != "token" {
				t.Fatalf("bad: %v", entry)
			}
		}
		if entry.Description == "" {
			t.Fatalf("bad: %v", entry)
		}
		if entry.UUID == "" {
			t.Fatalf("bad: %v", entry)
		}
	}
}
