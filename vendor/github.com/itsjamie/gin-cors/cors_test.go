package cors

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

var sHeaders = "TestOne, TestTwo, TestThree, TestFour, TestFive"
var config Config = Config{
	Origins:         "*",
	ValidateHeaders: true,
	Credentials:     true,
	RequestHeaders:  "Authorization, Content-Type, Accept",
	ExposedHeaders:  "Authorization",
	Methods:         "GET, POST",
	MaxAge:          1 * time.Minute,
}

func TestPrepare(t *testing.T) {
	config := Config{
		Origins:         "*",
		ValidateHeaders: true,
		Credentials:     true,
		RequestHeaders:  "Authorization, Content-Type, Accept",
		ExposedHeaders:  "Authorization",
		Methods:         "GET, POST",
		MaxAge:          1 * time.Minute,
	}
	config.prepare()

	if len(config.requestHeaders) != 3 {
		t.Fatal("Unexpected number of request headers")
	}

	if len(config.methods) != 2 {
		t.Fatal("Unexpected number of methods.")
	}

	if len(config.origins) != 1 {
		t.Fatal("Unexpected number of origins")
	}

	if config.credentials != "true" {
		t.Fatal("String conversion bad?")
	}

	if config.maxAge != "60" {
		t.Fatalf("One minute should be sixty seconds, and it should be stored as a string. Instead it is %s", config.maxAge)
	}
}

func TestBadConfiguration(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil {
			t.Fatal("With no origin set, we should panic.")
		}
	}()
	Middleware(Config{Origins: ""})
}

func TestNoOrigin(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	router := gin.New()

	router.Use(Middleware(Config{
		Origins: "http://testing.com",
	}))

	router.ServeHTTP(w, req)

	if w.Header().Get(AllowOriginKey) != "" {
		t.Fatal("This should not match.")
	}
}

func TestMismatchOrigin(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	req.Header.Set("Origin", "http://files.testing.com")

	router := gin.New()

	router.Use(Middleware(Config{
		Origins: "http://testing.com",
	}))

	router.ServeHTTP(w, req)

	if w.Header().Get(AllowOriginKey) != "" {
		t.Fatal("This should not match.")
	}
}

func TestPreflightRequest(t *testing.T) {
	req, _ := http.NewRequest("OPTIONS", "/", nil)
	w := httptest.NewRecorder()

	req.Header.Set(OriginKey, "http://files.testing.com")
	req.Header.Set(RequestMethodKey, "GET")
	req.Header.Set(RequestHeadersKey, "Content-Type")
	req.Header.Set(RequestHeadersKey, "accept")

	router := gin.New()

	router.Use(Middleware(config))

	router.ServeHTTP(w, req)

	if w.Header().Get(AllowMethodsKey) != "GET, POST" {
		t.Fatal("Mismatch of methods.")
	}

	if w.Header().Get(AllowHeadersKey) != "Authorization, Content-Type, Accept" {
		t.Fatal("Mismatch of headers.")
	}

	if w.Header().Get(MaxAgeKey) != "60" {
		t.Fatal("Incorrect max age.")
	}

	if w.Header().Get(AllowOriginKey) != "http://files.testing.com" {
		t.Fatal("Incorrect origin.")
	}

	if w.Header().Get(AllowCredentialsKey) != "true" {
		t.Fatal("Incorrect credentials value")
	}
}

func TestPreflightMethodMismatch(t *testing.T) {
	req, _ := http.NewRequest("OPTIONS", "/", nil)
	w := httptest.NewRecorder()

	req.Header.Set(OriginKey, "http://files.testing.com")
	req.Header.Set(RequestMethodKey, "PUT")

	router := gin.New()

	router.Use(Middleware(config))

	router.ServeHTTP(w, req)

	if w.Header().Get(AllowOriginKey) != "" {
		t.Fatal("Cors headers should not be set.")
	}
}

func TestPreflightHeaderMismatch(t *testing.T) {
	req, _ := http.NewRequest("OPTIONS", "/", nil)
	w := httptest.NewRecorder()

	req.Header.Set(OriginKey, "http://files.testing.com")
	req.Header.Set(RequestMethodKey, "GET")
	req.Header.Set(RequestHeadersKey, "Range")

	router := gin.New()

	router.Use(Middleware(config))

	router.ServeHTTP(w, req)

	if w.Header().Get(AllowOriginKey) != "" {
		t.Fatal("Cors headers should not be set.")
	}
}

func TestMatchOrigin(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	req.Header.Set("Origin", "http://files.testing.com")

	router := gin.New()
	router.Use(Middleware(Config{
		Origins: "http://files.testing.com",
	}))
	router.ServeHTTP(w, req)

	if w.Header().Get(AllowOriginKey) == "" {
		t.Fatal("Origin matches, this header should be set.")
	}
}

func TestForceOrigin(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	req.Header.Set("Origin", "http://localhost")

	router := gin.New()
	router.Use(Middleware(config))
	router.ServeHTTP(w, req)

	if w.Header().Get(AllowOriginKey) == "" {
		t.Fatal("Origin always matches, this header should be set.")
	}
}

func TestForceOriginCredentails(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	
	req.Header.Set("Origin", "http://localhost")
	
	router := gin.New()
	router.Use(Middleware(Config{
		Origins:         "http://localhost",
		ValidateHeaders: true,
		Credentials:     false,
		RequestHeaders:  "Authorization, Content-Type, Accept",
		ExposedHeaders:  "Authorization",
		Methods:         "GET, POST",
		MaxAge:          1 * time.Minute,
	}))
	router.ServeHTTP(w, req)
	
	if w.Header().Get(AllowOriginKey) != "http://localhost" {
		t.Fatal("Improper Origin is set.")
	}
}

func TestValidateMethods(t *testing.T) {
	testFailMethod := "PUT"
	testPassMethod := "GET"

	config := Config{
		Origins:         "*",
		ValidateHeaders: true,
		Methods:         "GET, POST",
	}
	config.prepare()

	if test := validateRequestMethod(testFailMethod, config); test {
		t.Fatal("Expected to return false")
	}

	if test := validateRequestMethod(testPassMethod, config); !test {
		t.Fatal("Expected to return true")
	}

	config.ValidateHeaders = false

	if test := validateRequestMethod(testFailMethod, config); !test {
		t.Fatal("Expected to return true, since validate is off.")
	}
}

func TestValidateHeaders(t *testing.T) {
	testFailHeader := "Authorization, MissingHeader"
	testPassHeader := "Authorization, Accept"

	config := Config{
		Origins:         "*",
		ValidateHeaders: true,
		RequestHeaders:  "Authorization, Content-Type, Accept",
	}
	config.prepare()

	if test := validateRequestHeaders(testFailHeader, config); test {
		t.Fatal("Expected headers to not match, return false.")
	}

	if test := validateRequestHeaders(testPassHeader, config); !test {
		t.Fatal("Expected headers to match, should return true.")
	}

	config.ValidateHeaders = false

	if test := validateRequestHeaders(testFailHeader, config); !test {
		t.Fatal("Expect it to always return true when not validating.")
	}
}

func BenchmarkSortFive(b *testing.B) {
	ssHeaders := sort.StringSlice(strings.Split(sHeaders, ", "))
	sort.Sort(ssHeaders)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := rand.Intn(10)
		search := ""

		switch {
		case index >= 5 && index < 10:
			search = "NotFound"
		default:
			search = ssHeaders[index]
		}

		if idx := sort.SearchStrings(ssHeaders, search); ssHeaders[idx] == search {

		}
	}
}

func BenchmarkRangeFive(b *testing.B) {
	ssHeaders := strings.Split(sHeaders, ", ")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := rand.Intn(10)
		search := ""

		switch {
		case index >= 5 && index < 10:
			search = "NotFound"
		default:
			search = ssHeaders[index]
		}

		for _, value := range ssHeaders {
			if value == search {
				break
			}
		}
	}
}
