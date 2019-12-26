package test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/configstore"
)

//DBDriver is exported for testing purpose
var (
	DBDriver      string
	dbUser        string
	dbRole        string
	dbPassword    string
	dbName        string
	dbHost        string
	dbPort        int64
	dbSSLMode     string
	RedisHost     string
	RedisPassword string
)

func init() {
	log.Initialize(&log.Conf{Level: "debug"})
}

type Bootstrapf func(context.Context, sdk.DefaultValues, func() *gorp.DbMap) error

var DBConnectionFactory *database.DBConnectionFactory

// TestKey is a test key encoded in base64, do not use it in real life
var TestKey []byte

func init() {
	TestKey, _ = base64.StdEncoding.DecodeString(`LS0tLS1CRUdJTiBPUEVOU1NIIFBSSVZBVEUgS0VZLS0tLS0KYjNCbGJuTnphQzFyWlhrdGRqRUFBQUFBQkc1dmJtVUFBQUFFYm05dVpRQUFBQUFBQUFBQkFBQUNGd0FBQUFkemMyZ3RjbgpOaEFBQUFBd0VBQVFBQUFnRUFwNDN4L2U3S3B6TjJLMDc2Y1FhU3ppL3B0YTQyMVYvMnNPRGpuUE52dWpkamlQaEtYWW1kClFmNmtIZHFUeTVKdW5KVkJ5cTFFM0t6dWd0cCs1VUlGZ21tYVNWU092R1B0MkdZaERiemVsSUlsWTFhVlRvQ3BnL2hTb3gKRFhucXBlMkFRUjRIU1hHdEJTQUJVbXJYb1ZxNGRtZmFtYXloVzc5NmpTVEdMY0lMWFBJNXFUcFNlMjd3cGNGOGM5MXFGVgpUNFo5b0FMbzB2ZmlYQlF6c0tlMVNBbjZlOWtGUEtJZ01wak91Y3kyTkVuVDVLTTlJVFNSOTRCM0ptcDV5enBOMXpHdmRMCkxTQ2ptUGlxQnpCSUlncCt1T01WMDBZT2JiUHJOVTM5NHJhYzJWR2tOb2JYb3kwL05YUUxQY0hGeUxSMXoyVGxsWUNabWUKUVZKV1laelVtYm5CaWN4SXlDYVQ1bExGdmorTjZkbWd5cE90aTJXSEY1MHJlOG0xa0ZDTll4UlkrWjAzTDliNzFiS1UzcgpzOVM2RThHMDVZWVFQOER6MWdzUWRvSVRwT0w4cnROaDd5SWdCaG8weVozQ2RnQ09Pem1rM0hOaU5NQXdNUS9NWHZXMytLCjNhTFMxMG1tb3dJT0ZXdWtKT2cxWkJBRWVndnpNZ2xxcGZ1eElWNWljT3oxOUVST0Q1bkZjeGl5TkV0dTAza3hTVzhnRjUKYlBNTnFydkNFNytqSFVxSzlhRmpxeHFtOWZvSjJRZnlZcG5qQmZpd1JkWG9yZzBwSzcvRnNYeFhHMkZ6ZHF4RmNtK2FEbwpGbnlxZzExWTFhdlBYKzZ2WXk5bE4rRWRNQ1NNVUt3U3NSMkdaMkg1M1FDczFSQWhsaG10MDFtS2k2Y0Q1VTVLRVJGTkxWClVBQUFkZ1pyWlVLV2EyVkNrQUFBQUhjM05vTFhKellRQUFBZ0VBcDQzeC9lN0twek4ySzA3NmNRYVN6aS9wdGE0MjFWLzIKc09Eam5QTnZ1amRqaVBoS1hZbWRRZjZrSGRxVHk1SnVuSlZCeXExRTNLenVndHArNVVJRmdtbWFTVlNPdkdQdDJHWWhEYgp6ZWxJSWxZMWFWVG9DcGcvaFNveERYbnFwZTJBUVI0SFNYR3RCU0FCVW1yWG9WcTRkbWZhbWF5aFc3OTZqU1RHTGNJTFhQCkk1cVRwU2UyN3dwY0Y4YzkxcUZWVDRaOW9BTG8wdmZpWEJRenNLZTFTQW42ZTlrRlBLSWdNcGpPdWN5Mk5FblQ1S005SVQKU1I5NEIzSm1wNXl6cE4xekd2ZExMU0NqbVBpcUJ6QklJZ3ArdU9NVjAwWU9iYlByTlUzOTRyYWMyVkdrTm9iWG95MC9OWApRTFBjSEZ5TFIxejJUbGxZQ1ptZVFWSldZWnpVbWJuQmljeEl5Q2FUNWxMRnZqK042ZG1neXBPdGkyV0hGNTByZThtMWtGCkNOWXhSWStaMDNMOWI3MWJLVTNyczlTNkU4RzA1WVlRUDhEejFnc1Fkb0lUcE9MOHJ0Tmg3eUlnQmhvMHlaM0NkZ0NPT3oKbWszSE5pTk1Bd01RL01YdlczK0szYUxTMTBtbW93SU9GV3VrSk9nMVpCQUVlZ3Z6TWdscXBmdXhJVjVpY096MTlFUk9ENQpuRmN4aXlORXR1MDNreFNXOGdGNWJQTU5xcnZDRTcrakhVcUs5YUZqcXhxbTlmb0oyUWZ5WXBuakJmaXdSZFhvcmcwcEs3Ci9Gc1h4WEcyRnpkcXhGY20rYURvRm55cWcxMVkxYXZQWCs2dll5OWxOK0VkTUNTTVVLd1NzUjJHWjJINTNRQ3MxUkFobGgKbXQwMW1LaTZjRDVVNUtFUkZOTFZVQUFBQURBUUFCQUFBQ0FRQ1FNU0NTeGdBU09jQTA3d2VwWXQzTm9RQUFRTWVoZ3E4YQpjcjZPWUJURGJVMDBIM0JuNUxpM2hYc1kwZlNrbVFTbHJmRHJpWWNjWFpuNGRDNEYvM1ljVCtMZHZtNERnLys0WGRPT0xmCjVpVVVuNW5oWnBjMkh1VnpKT2NIME9aMUd0bG5zSDdXM29QbVNDKzdESVU2cjRiVkp2VEJrUVZmbm4zSm4xOEpHOWVKaWsKN0M2cFQyOG5jWVBsVnFwSjNaYzhFK0ppWkg2V3A0cGVjV2cyVzIwdmJKN3FHODVjNnF6SXZpWVJVVEZ2K0NUb3V1NHRlRAo4eGZwV0xNdEJUYTM1M2RhT255d2Zra3JxTHN4Nm9QNC80MGtjUkJrUEFMSXQ2L3Z0SW1McEZtQXo3aUEwRFFja2lDMlVJCklvQ0d5OEYwalhUTjRpZFlRNklrVnNaTnhKaFR1YkdCeUg2V0JVdjV3VlNxd2FQUk5yWTRtRE1ONUI2N04vWEZIdjVhejcKQjUwWUVwYUhKTmxTdlphc2JWSmQ5QU0vU0x0cERuNDArZ0gvUzdXVDVvTTRpQWtUcWxzdjZRRHQzTURJOVZ5NTluaGFsMgpNQXdZc3NzbnZldWNuYVR5OUFOQWdHNjMzb3E1N1BReGFGdEFVUkFxQ3ptNVhhbjY2UTVISkhKZHJQcWdXM3RrRzNxall6CmFoMnFGdW05bHFhMTJtWVRNdTZkNlRiUmtFQ1lkTk02QTRwdnIxeVhWQkY0M09ncjU0T1FXaVNKV0Jka3JObHNKcGl5RUwKcjlRSG9nNkpIRmc0ckJWT0xjZDkvVWV0R3NDSnpTZ0JNNWxwZFV0RmdlVGJVREUzSDhHbjdqZlZRd1VYcW8xVENjdnh6bAp5ZWdRaXFTSThtaHJxMzJUNllkUUFBQVFCNlY2a2gxSjA2ZVlveFBYMjZCNlIrVC9XWUM3Tkp3eXdLSzJjTE5FbjBDWE9pClFqWnFKOHdocWo0TVl2Q2xmTGJSNUtNd2dib3J5NU1yQXJpQ2NzVVRTRy9XSWlFTlJEM005VUo3ZnN0bGY5bWllQlNWKzUKVEovMkZ3VTFabU82ci9nbC9BNFRNMGJUVXR1aFJZc0x0SFZVS3pFaW5LVVpUQi83S3l1SWljNDhIL1ZNcFNUdlpuUzVSagpRVzQvN0dUQ2tFSERMUEJnNmJZSGVPM2YrT1hkTTJnSnlaL3JrTmwvVm1RSGIrYS84Z2FUL21uTEtGNDhBVlkxQmUxMFNTCkoxcHM5eEQ5bm1IMWJZMGFlUUo3dEZkTE1IZGhoV0Y3Q05lT21xSE5ZMVdMaHczeC90cEZmcVJjbTYyTUp6ODhhaFNickwKekxRbWVONVR0U2hYeHJlK0FBQUJBUURiVWxyWG9XeXRHOHFGWUVqeGMrQmZFM3ltYTJwVlpCWjg2TEtnRjk5Rmo0dkdkRApMNWd3dDBGMTNLblpndUZka0UycHJGczlrbmxGQlFMbHcyZWRiVnVhUERrcURncGdVOFlEaEdxSXNzUlZpR0hCNmJqZ1hpClFaeW1LM1B3K005eERKVmpEYnIyVUt2Wk5ZWnVUWkFvcGVTVkVyaW84bFJsVUw2d1dOMm9RMTVGcWJYVysvSzVXaStXWHgKb3lEZTlyVGFZKzU1UndFMTNHT2JlNG5ZanJmeVl5OHhaZGthU3FWMWJVU2NlVkNtaXc3WnJ2eUs3TDVIRnh3ME11eUg2RApsMlZCMElZL1RZU21OYlhyZVpnR21uUkFJbmk5VzFyS0J2K3JXWmllVzV3RzdxdlNlSm9NamU4aGNmeXZRWHFTakxXRlR4CjNWY2xybjVyMUo4bFQzQUFBQkFRRERrMUpNQ0xiQVpIYVVBYmh2MGVxSFlLRWFpWTBnKzZZakNmSnNoMi96eEs2NnZiYUQKak5NTmdxQXNqem4zUU9Kb1hBbmRCRmxKOTZqKzNqQlpaa3dwMVFyY2N3eUtrUUI1NGdXRjV2NHFXSjZjNkZSZUFpYzR4bwo1a2l4cTNqVUJNcG85TDNIQ3dGYlpxVzZqTkI3NDlHSXo2VEhCdEFZYWdaWHpMZ2haTDJLcGhqM2c0SXkzaW9TTkk3NFc1CllzQXFLSWhxNGFFTitJTERIdjhuaTRaWWM3OXpFNGJXRCtTRmp1aC9XazJweDJUM2RIaHRLaGg4eXRhamh0UlNhcVowY2MKRlIrSy93OE1PVDVzelpIbllzaVV4R1BnKzkvZVlZYlZ6azZGazJHL2xrN0RXZ1BweitPVi9QS29QMGs5RDhUZEZzbE5EWApvODBOYmdxYXNWa1RBQUFBS0daeVlXNWpiMmx6TG5OaGJXbHVLMmRwZEdoMVltWmhhMlZyWlhsQVpYaGhiWEJzWlM1amIyCjBCQWc9PQotLS0tLUVORCBPUEVOU1NIIFBSSVZBVEUgS0VZLS0tLS0K`)
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

// SetupPG setup PG DB for test
func SetupPG(t log.Logger, bootstrapFunc ...Bootstrapf) (*gorp.DbMap, cache.Store, context.CancelFunc) {
	log.SetLogger(t)
	cfg := LoadTestingConf(t)
	DBDriver = cfg["dbDriver"]
	dbUser = cfg["dbUser"]
	dbRole = cfg["dbRole"]
	dbPassword = cfg["dbPassword"]
	dbName = cfg["dbName"]
	dbHost = cfg["dbHost"]
	var err error
	dbPort, err = strconv.ParseInt(cfg["dbPort"], 10, 64)
	if err != nil {
		t.Errorf("Error when unmarshal config %s", err)
	}
	dbSSLMode = cfg["sslMode"]
	RedisHost = cfg["redisHost"]
	RedisPassword = cfg["redisPassword"]

	secret.Init("3dojuwevn94y7orh5e3t4ejtmbtstest")
	err = authentication.Init("cds_test", SigningKey) // nolint
	if err != nil {
		log.Fatalf("unable to init authentication layer: %v", err)
	}

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
	configstore.AllowProviderOverride()

	signatureKeyConfig := sigKeys.GetKeys(gorpmapping.KeySignIdentifier)
	encryptionKeyConfig := encryptKeys.GetKeys(gorpmapping.KeyEcnryptionIdentifier)
	if err := gorpmapping.ConfigureKeys(&signatureKeyConfig, &encryptionKeyConfig); err != nil {
		t.Fatalf("cannot setup database keys: %v", err)
		return nil, nil, func() {}
	}

	if DBDriver == "" {
		t.Fatalf("This should be run with a database")
		return nil, nil, func() {}
	}
	if DBConnectionFactory == nil {
		var err error
		DBConnectionFactory, err = database.Init(context.TODO(), dbUser, dbRole, dbPassword, dbName, dbHost, int(dbPort), dbSSLMode, 10, 2000, 100)
		if err != nil {
			t.Fatalf("Cannot open database: %s", err)
			return nil, nil, func() {}
		}
	}

	for _, f := range bootstrapFunc {
		if err := f(context.TODO(), sdk.DefaultValues{}, DBConnectionFactory.GetDBMap); err != nil {
			log.Error(context.TODO(), "Error: %v", err)
			return nil, nil, func() {}
		}
	}

	store, err := cache.NewRedisStore(RedisHost, RedisPassword, 60)
	if err != nil {
		t.Fatalf("Unable to connect to redis: %v", err)
	}

	cancel := func() {
		store.Client.Close()
		store.Client = nil
	}

	return DBConnectionFactory.GetDBMap(), store, cancel
}

// LoadTestingConf loads test configuration tests.cfg.json
func LoadTestingConf(t log.Logger) map[string]string {
	var f string
	u, _ := user.Current()
	if u != nil {
		f = path.Join(u.HomeDir, ".cds", "tests.cfg.json")
	}

	if _, err := os.Stat(f); err == nil {
		t.Logf("Tests configuration read from %s", f)
		btes, err := ioutil.ReadFile(f)
		if err != nil {
			t.Fatalf("Error reading %s: %v", f, err)
		}
		if len(btes) != 0 {
			cfg := map[string]string{}
			if err := json.Unmarshal(btes, &cfg); err != nil {
				t.Fatalf("Error reading %s: %v", f, err)
			}
			return cfg
		}
	} else {
		t.Fatalf("Error reading %s: %v", f, err)
	}
	return nil
}

//GetTestName returns the name the the test
func GetTestName(t *testing.T) string {
	return t.Name()
}

//FakeHTTPClient implements sdk.HTTPClient and returns always the same response
type FakeHTTPClient struct {
	T        *testing.T
	Response *http.Response
	Error    error
}

//Do implements sdk.HTTPClient and returns always the same response
func (f *FakeHTTPClient) Do(r *http.Request) (*http.Response, error) {
	b, err := ioutil.ReadAll(r.Body)
	if err == nil {
		r.Body.Close()
	}

	f.T.Logf("FakeHTTPClient> Do> %s %s: Payload %s", r.Method, r.URL.String(), string(b))
	return f.Response, f.Error
}
