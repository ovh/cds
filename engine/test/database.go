package test

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/database"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type Bootstrapf func(context.Context, sdk.DefaultValues, func() *gorp.DbMap) error

var mDBConnectionFactoriesMutex sync.RWMutex
var mDBConnectionFactories map[string]*database.DBConnectionFactory

func init() {
	log.Initialize(context.TODO(), &log.Conf{Level: "debug"})
	mDBConnectionFactories = map[string]*database.DBConnectionFactory{}
}

// SigningKey is a test key, do not use it in real life
var SigningKey = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEApYHspLfY2IY8ZFIQEuKWMWSgGMUApOz6R16ZEifBEh+B/E3v
Gl9Pm+G/QhwTUVM/OvLCajX2Gf0zvpDFmOfLhIFYGnI8o94E8zo/5xoz/sJxy9iN
1soS9vpC9MxQU4W4QjoX78OHKBkYQkqsQs6sb+ZRxX90Og/PQUl/F0eeldX+8INd
HzCF1Ju8yh2pDjGKTEO3seCEoHBM6OG8optE7mcjTvDs2VpSzOeaZ0uxTOV9YxOV
73aTrcm79kVw12lyXbeNsx3fKKPiELY6TWR6loi8tIBb77tzJRCdScudVhF9MV9q
mbKqHZU/XE2FlSmHxs1Ijn8z+plXG9Gg7bYFpQIDAQABAoIBABBU8MgUSDadkGoJ
2wIyD8YR+uZW0kh0BgJy6EHtYFTsfQQroJOGojFsplXctV9KCqxDdkHKz10jKi78
1DLRdLi/lrUNXsSAzRY/Qj0Izeauw1HtLZnrWNG8Qk0ruCV1xYfreZ80OSsQxt8L
xVHWWRe1r44AlLSCCN6VZRAkBhcc3SI3AuW5l+BwicMljkPeHzbq2SGAIxxHw4Hn
3+aV+U1tth2taeDFhYnUwx4uGscsMFJebohllILqyz5oXJkm4Xo5ulViPBNudYvC
aH2IdwLhyngYjEHexVzuj9SEcKXYgeNWRL9hO5by5a85q0h4NVsYHaqCttEuW7Rp
l2zs7sUCgYEA1kkQMY0QR5WTT4Ogdmw2dN2TLuzmzt8fFYmuDw/1Hp62twrTnlq9
QBFbtcD00UkrLkSnSrLsHFgtGPFlcT0y8YD2n+zlgncF+zthXTqdAYDEZc2lNtIB
t3GJnKU2q8U9CGEwxMS0izQPbriwbrhIKVJPJ/lR0fUNM6N3sXWDsRcCgYEAxboC
gVLZSC5PtmcEDLFnYvqfReVEQ9nTYdIajIAq4nIWpqOxVbEb5KQZkm7KvmU6d7Yq
EjNQHuFqg9txyrw8z+kYoNrmo6T3wwoPspmwSzsu49S5pD+cEwnyF0hgU+yzsKIZ
MGU+Skpy1BQF0Anox0QFcLf/XxizLKm+DcWIXKMCgYBuqZrQTC5NGaTS2oIixi21
Wrxo7nUf/sA5yjl2k+Idpw9rJg81Z1z22kAHdBe6gVPoeBIBFLe0x6C6kee2fElz
yQsUei3om3keToMwt1Vf8lT60iHxVrEGQH81w2iheqHTUwXxiDhI72DM6FpNQ6QY
muZAGZS0nh3sPg5ROgQBjwKBgQDBHdjeiJWRizHtvBXXc9m/cXroYHFZN8HeM8Ac
Y/3+p2F6JjzIri/JE4GqZK1+Yg5F59SVbCqfzpgi6szsLwfSJR8Z1FMZl8EpbIVC
chsej1JP0W/zfPEqIzehB96VeYVTSi8B9pBtLOOUQW4f79276bLKkdtI/S3avHrU
po51swKBgC3+fkww8C8F6YIitAruTy4AkFp6Gr736hbnCPeLWtfW2HxXN5ca86h4
RvySYHtJvP64+7ncMqNjMnX8MbJZdoW3FE4gKomVof26/oUk/zHKNn5BECPIqEQv
XeJEyyEjosSa3qWACDYorGMnzRXdeJa5H7J0W+G3x4tH2LMW8VHS
-----END RSA PRIVATE KEY-----`)

type FakeTransaction struct {
	*gorp.DbMap
}

func (f *FakeTransaction) Rollback() error { return nil }
func (f *FakeTransaction) Commit() error   { return nil }

func SetupPGWithMapper(t *testing.T, m *gorpmapper.Mapper, serviceType string, bootstrapFunc ...Bootstrapf) (*FakeTransaction, cache.Store) {
	log.SetLogger(t)
	db, _, cache, cancel := SetupPGToCancel(t, m, serviceType, bootstrapFunc...)
	t.Cleanup(cancel)
	return db, cache
}

// SetupPGToCancel setup PG DB for test
func SetupPGToCancel(t require.TestingT, m *gorpmapper.Mapper, serviceType string, bootstrapFunc ...Bootstrapf) (*FakeTransaction, *database.DBConnectionFactory, cache.Store, func()) {
	cfg := LoadTestingConf(t, serviceType)

	dbDriver := cfg["dbDriver"]
	require.NotEmpty(t, dbDriver, "db driver value should be given")

	dbUser := cfg["dbUser"]
	dbRole := cfg["dbRole"]
	dbPassword := cfg["dbPassword"]
	dbName := cfg["dbName"]
	dbSchema := cfg["dbSchema"]
	dbHost := cfg["dbHost"]
	dbPort, err := strconv.ParseInt(cfg["dbPort"], 10, 64)
	require.NoError(t, err, "error when unmarshal config")
	dbSSLMode := cfg["sslMode"]
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]

	sigKeys := database.RollingKeyConfig{
		Cipher: "hmac",
		Keys: []database.KeyConfig{
			{
				Timestamp: time.Now().Unix(),
				Key:       "8f17c90d5306028bdf6ef66cc6da387aca9dd57a11f44e5e2752228398b7d165",
			},
		},
	}

	encryptKeys := database.RollingKeyConfig{
		Cipher: "xchacha20-poly1305",
		Keys: []database.KeyConfig{
			{
				Timestamp: time.Now().Unix(),
				Key:       "fd27b8872bdefeb207bbefc1a82e94039b85d3ec68d891e22a5dcaa81542fc6b",
			},
		},
	}

	signatureKeyConfig := sigKeys.GetKeys(gorpmapper.KeySignIdentifier)
	encryptionKeyConfig := encryptKeys.GetKeys(gorpmapper.KeyEcnryptionIdentifier)
	require.NoError(t, m.ConfigureKeys(&signatureKeyConfig, &encryptionKeyConfig), "cannot setup database keys")

	dbConnectionConfigKey := dbUser + dbRole + dbPassword + dbName + dbSchema + dbHost + fmt.Sprintf("%d", dbPort)
	mDBConnectionFactoriesMutex.RLock()
	factory, ok := mDBConnectionFactories[dbConnectionConfigKey]
	mDBConnectionFactoriesMutex.RUnlock()
	if !ok {
		factory, err = database.Init(context.TODO(), dbUser, dbRole, dbPassword, dbName, dbSchema, dbHost, int(dbPort), dbSSLMode, 10, 2000, 100)
		require.NoError(t, err, "cannot open database")
		mDBConnectionFactoriesMutex.Lock()
		mDBConnectionFactories[dbConnectionConfigKey] = factory
		mDBConnectionFactoriesMutex.Unlock()
	}

	for _, f := range bootstrapFunc {
		require.NoError(t, f(context.TODO(), sdk.DefaultValues{}, factory.GetDBMap(m)))
	}

	store, err := cache.NewRedisStore(redisHost, redisPassword, 60)
	require.NoError(t, err, "unable to connect to redis")

	cancel := func() {
		store.Client.Close()
		store.Client = nil
	}

	dbMap := factory.GetDBMap(m)()
	require.NotNil(t, dbMap, "unable to init database connection")

	return &FakeTransaction{
		DbMap: dbMap,
	}, factory, store, cancel
}
