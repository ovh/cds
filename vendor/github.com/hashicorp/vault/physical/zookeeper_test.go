package physical

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/vault/helper/logformat"
	log "github.com/mgutz/logxi/v1"

	"github.com/samuel/go-zookeeper/zk"
)

func TestZookeeperBackend(t *testing.T) {
	addr := os.Getenv("ZOOKEEPER_ADDR")
	if addr == "" {
		t.SkipNow()
	}

	client, _, err := zk.Connect([]string{addr}, time.Second)

	if err != nil {
		t.Fatalf("err: %v", err)
	}

	randPath := fmt.Sprintf("/vault-%d", time.Now().Unix())
	acl := zk.WorldACL(zk.PermAll)
	_, err = client.Create(randPath, []byte("hi"), int32(0), acl)

	if err != nil {
		t.Fatalf("err: %v", err)
	}

	defer func() {
		client.Delete(randPath+"/foo/nested1/nested2/nested3", -1)
		client.Delete(randPath+"/foo/nested1/nested2", -1)
		client.Delete(randPath+"/foo/nested1", -1)
		client.Delete(randPath+"/foo/bar/baz", -1)
		client.Delete(randPath+"/foo/bar", -1)
		client.Delete(randPath+"/foo", -1)
		client.Delete(randPath, -1)
		client.Close()
	}()

	logger := logformat.NewVaultLogger(log.LevelTrace)

	b, err := NewBackend("zookeeper", logger, map[string]string{
		"address": addr + "," + addr,
		"path":    randPath,
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	testBackend(t, b)
	testBackend_ListPrefix(t, b)
}

func TestZookeeperHABackend(t *testing.T) {
	addr := os.Getenv("ZOOKEEPER_ADDR")
	if addr == "" {
		t.SkipNow()
	}

	client, _, err := zk.Connect([]string{addr}, time.Second)

	if err != nil {
		t.Fatalf("err: %v", err)
	}

	randPath := fmt.Sprintf("/vault-ha-%d", time.Now().Unix())
	acl := zk.WorldACL(zk.PermAll)
	_, err = client.Create(randPath, []byte("hi"), int32(0), acl)

	if err != nil {
		t.Fatalf("err: %v", err)
	}

	defer func() {
		client.Delete(randPath+"/foo", -1)
		client.Delete(randPath, -1)
		client.Close()
	}()

	logger := logformat.NewVaultLogger(log.LevelTrace)

	b, err := NewBackend("zookeeper", logger, map[string]string{
		"address": addr + "," + addr,
		"path":    randPath,
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	ha, ok := b.(HABackend)
	if !ok {
		t.Fatalf("zookeeper does not implement HABackend")
	}
	testHABackend(t, ha, ha)
}
