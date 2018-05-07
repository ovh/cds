package redisctx

import (
	"fmt"

	"github.com/garyburd/redigo/redis"
	"github.com/ovh/venom"
)

// Name is Context Type name.
const Name = "redis"

//Client represents interface of redis client used by venom
type Client interface {
	Close() error
	Do(commandName string, args ...interface{}) (reply interface{}, err error)
}

// New returns a new TestCaseContext.
func New() venom.TestCaseContext {
	ctx := &RedisTestCaseContext{}
	ctx.Name = Name
	return ctx
}

// RedisTestCaseContext represents the context of a testcase.
type RedisTestCaseContext struct {
	venom.CommonTestCaseContext
	Client Client
}

// Init Initialize the context.
func (tcc *RedisTestCaseContext) Init() error {
	var dialURL string
	if v, found := tcc.TestCase.Context["dialURL"]; found {
		if url, ok := v.(string); ok {
			dialURL = url
		} else {
			return fmt.Errorf("DialURL property must be a string")
		}
	} else {
		return fmt.Errorf("DialURL property isn't present")
	}

	var err error
	tcc.Client, err = redis.DialURL(dialURL)
	return err
}

// Close the context.
func (tcc *RedisTestCaseContext) Close() error {
	return tcc.Client.Close()
}
