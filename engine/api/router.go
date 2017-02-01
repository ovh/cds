package main

import (
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

var (
	router    *Router
	panicked  bool
	nbPanic   int
	lastPanic *time.Time
)

const nbPanicsBeforeFail = 50

// Handler defines the HTTP handler used in CDS engine
type Handler func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error

// RouterConfigParam is the type of anonymous function returned by POST, GET and PUT functions
type RouterConfigParam func(rc *routerConfig)

type routerConfig struct {
	get           Handler
	post          Handler
	put           Handler
	deleteHandler Handler
	auth          bool
	isExecution   bool
	needAdmin     bool
}

// ServeAbsoluteFile Serve file to download
func (r *Router) ServeAbsoluteFile(uri, path, filename string) {
	f := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=\"%s\"", filename))
		http.ServeFile(w, r, path)
	}
	router.mux.HandleFunc(r.prefix+uri, f)
}

func compress(fn http.HandlerFunc) http.HandlerFunc {
	return handlers.CompressHandlerLevel(fn, gzip.DefaultCompression).ServeHTTP
}

func recoverWrap(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var err error
		defer func() {
			if re := recover(); re != nil {
				switch t := re.(type) {
				case string:
					err = errors.New(t)
				case error:
					err = re.(error)
				case sdk.Error:
					err = re.(sdk.Error)
				default:
					err = sdk.ErrUnknownError
				}
				log.Critical("[PANIC_RECOVERY] Panic occured on %s:%s, recover %s", req.Method, req.URL.String(), err)
				trace := make([]byte, 4096)
				count := runtime.Stack(trace, true)
				log.Critical("[PANIC_RECOVERY] Stacktrace of %d bytes\n%s\n", count, trace)

				//Reinit database connection
				if _, e := database.Init(); e != nil {
					log.Critical("[PANIC_RECOVERY] Unable to reinit db connection : %s", e)
				}

				//Checking if there are two much panics in two minutes
				//If last panic was more than 2 minutes ago, reinit the panic counter
				if lastPanic == nil {
					nbPanic = 0
				} else {
					dur := time.Since(*lastPanic)
					if dur.Minutes() > float64(2) {
						log.Notice("[PANIC_RECOVERY] Last panic was %d seconds ago", int(dur.Seconds()))
						nbPanic = 0
					}
				}

				nbPanic++
				now := time.Now()
				lastPanic = &now
				//If two much panic, change the status of /mon/status with panicked = true
				if nbPanic > nbPanicsBeforeFail {
					panicked = true
					log.Critical("[PANIC_RECOVERY] RESTART NEEDED")
				}

				WriteError(w, req, err)
			}
		}()
		h.ServeHTTP(w, req)
	})
}

//Router is our base router struct
type Router struct {
	authDriver auth.Driver
	mux        *mux.Router
	prefix     string
}

var mapRouterConfigs = map[string]*routerConfig{}

// Handle adds all handler for their specific verb in gorilla router for given uri
func (r *Router) Handle(uri string, handlers ...RouterConfigParam) {
	uri = r.prefix + uri
	rc := &routerConfig{auth: true, isExecution: false, needAdmin: false}
	mapRouterConfigs[uri] = rc

	for _, h := range handlers {
		h(rc)
	}

	f := func(w http.ResponseWriter, req *http.Request) {
		// Close indicates  to close the connection after replying to this request
		req.Close = true
		// Authorization ?
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", "GET,OPTIONS,PUT,POST,DELETE")
		w.Header().Add("Access-Control-Allow-Headers", "Accept, Origin, Referer, User-Agent, Content-Type, Authorization, Session-Token, Last-Event-Id")
		w.Header().Add("Access-Control-Expose-Headers", "Accept, Origin, Referer, User-Agent, Content-Type, Authorization, Session-Token, Last-Event-Id")

		c := &context.Ctx{}

		if req.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		//Check DB connection
		db := database.DBMap(database.DB())
		if db == nil {
			//We can handle database loss with hook.recovery
			if req.URL.Path != "/hook" {
				WriteError(w, req, sdk.ErrServiceUnavailable)
				return
			}
		}

		if rc.auth {
			if err := r.checkAuthentication(db, req.Header, c); err != nil {
				log.Warning("Authorization denied on %s %s for %s: %s\n", req.Method, req.URL, req.RemoteAddr, err)
				WriteError(w, req, sdk.ErrUnauthorized)
				return
			}
		}

		permissionOk := true
		if rc.auth && rc.needAdmin && !c.User.Admin {
			permissionOk = false
		} else if rc.auth && !rc.needAdmin && !c.User.Admin {
			permissionOk = checkPermission(mux.Vars(req), c, getPermissionByMethod(req.Method, rc.isExecution))
		}
		if permissionOk {
			start := time.Now()
			defer func() {
				end := time.Now()
				latency := end.Sub(start)
				log.Info("%-7s | %13v | %v", req.Method, latency, req.URL)
			}()

			if req.Method == "GET" && rc.get != nil {
				if err := rc.get(w, req, db, c); err != nil {
					log.Debug("Error : %s %v : %T %s", req.Method, req.URL, err, err)
					WriteError(w, req, err)
				}
				return
			}

			if req.Method == "POST" && rc.post != nil {
				if err := rc.post(w, req, db, c); err != nil {
					log.Debug("Error : %s %v : %T %s", req.Method, req.URL, err, err)
					WriteError(w, req, err)
				}
				return
			}
			if req.Method == "PUT" && rc.put != nil {
				if err := rc.put(w, req, db, c); err != nil {
					log.Debug("Error : %s %v : %T %s", req.Method, req.URL, err, err)
					WriteError(w, req, err)
				}
				return
			}

			if req.Method == "DELETE" && rc.deleteHandler != nil {
				if err := rc.deleteHandler(w, req, db, c); err != nil {
					log.Debug("Error : %s %v : %T %s", req.Method, req.URL, err, err)
					WriteError(w, req, err)
				}
				return
			}
			WriteError(w, req, sdk.ErrNotFound)
			return
		}
		WriteError(w, req, sdk.ErrForbidden)
		return
	}
	router.mux.HandleFunc(uri, compress(recoverWrap(f)))
}

// GET will set given handler only for GET request
func GET(h Handler) RouterConfigParam {
	f := func(rc *routerConfig) {
		rc.get = h
	}

	return f
}

// POST will set given handler only for POST request
func POST(h Handler) RouterConfigParam {
	f := func(rc *routerConfig) {
		rc.post = h
	}
	return f
}

// POSTEXECUTE will set given handler only for POST request and add a flag for execution permission
func POSTEXECUTE(h Handler) RouterConfigParam {
	f := func(rc *routerConfig) {
		rc.post = h
		rc.isExecution = true
	}

	return f
}

// PUT will set given handler only for PUT request
func PUT(h Handler) RouterConfigParam {
	f := func(rc *routerConfig) {
		rc.put = h
	}
	return f
}

// NeedAdmin set the route for cds admin only (or not)
func NeedAdmin(admin bool) RouterConfigParam {
	f := func(rc *routerConfig) {
		rc.needAdmin = admin
	}
	return f
}

// DELETE will set given handler only for DELETE request
func DELETE(h Handler) RouterConfigParam {
	f := func(rc *routerConfig) {
		rc.deleteHandler = h
	}
	return f
}

func (r *Router) getRoute(method string, handler Handler, vars map[string]string) string {
	sf1 := reflect.ValueOf(handler)
	var url string
	for uri, routerConfig := range mapRouterConfigs {
		if strings.HasPrefix(uri, r.prefix) {
			switch method {
			case "GET":
				sf2 := reflect.ValueOf(routerConfig.get)
				if sf1.Pointer() == sf2.Pointer() {
					url = uri
					break
				}
			case "POST":
				sf2 := reflect.ValueOf(routerConfig.post)
				if sf1.Pointer() == sf2.Pointer() {
					url = uri
					break
				}
			case "PUT":
				sf2 := reflect.ValueOf(routerConfig.put)
				if sf1.Pointer() == sf2.Pointer() {
					url = uri
					break
				}
			case "DELETE":
				sf2 := reflect.ValueOf(routerConfig.deleteHandler)
				if sf1.Pointer() == sf2.Pointer() {
					url = uri
					break
				}
			}
		}
	}

	for k, v := range vars {
		url = strings.Replace(url, "{"+k+"}", v, -1)
	}

	if url == "" {
		log.Debug("Cant find route for Handler %s %v", method, handler)
	}

	return url
}

// Auth set manually whether authorisation layer should be applied
// Authorization is enabled by default
func Auth(v bool) RouterConfigParam {
	f := func(rc *routerConfig) {
		rc.auth = v
	}
	return f
}

func (r *Router) checkAuthHeader(db *gorp.DbMap, headers http.Header, c *context.Ctx) error {
	return r.authDriver.GetCheckAuthHeaderFunc(localCLientAuthMode)(db, headers, c)
}

func (r *Router) checkAuthentication(db *gorp.DbMap, headers http.Header, c *context.Ctx) error {

	c.Agent = sdk.Agent(headers.Get("User-Agent"))

	switch headers.Get("User-Agent") {
	// TODO: case sdk.WorkerAgent should be moved here
	case sdk.HatcheryAgent:
		return r.checkHatcheryAuth(db, headers, c)
	default:
		return r.checkAuthHeader(db, headers, c)
	}
}

func (r *Router) checkHatcheryAuth(db *gorp.DbMap, headers http.Header, c *context.Ctx) error {
	id, err := base64.StdEncoding.DecodeString(headers.Get(sdk.AuthHeader))
	if err != nil {
		return fmt.Errorf("bad worker key syntax: %s", err)
	}
	log.Debug("HatcheryAuth> Hatchery looking for auth (%s)\n", id)

	h, err := hatchery.LoadHatchery(db, string(id))
	if err != nil {
		return err
	}

	log.Debug("HatcheryAuth> Loading permissions for group %d\n", h.GroupID)
	c.User = &sdk.User{Username: h.Name}
	g, err := user.LoadGroupPermissions(db, h.GroupID)
	if err != nil {
		return fmt.Errorf("cannot load group permissions: %s", err)
	}
	c.User.Groups = append(c.User.Groups, *g)
	return nil
}

// WriteJSON is a helper function to marshal json, handle errors and set Content-Type for the best
func WriteJSON(w http.ResponseWriter, r *http.Request, data interface{}, status int) error {
	b, e := json.Marshal(data)
	if e != nil {
		log.Warning("return WriteJSON> unable to marshal : %s", e)
		return sdk.ErrUnknownError

	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(b)
	return nil
}

func notFoundHandler(w http.ResponseWriter, req *http.Request) {
	start := time.Now()
	defer func() {
		end := time.Now()
		latency := end.Sub(start)
		log.Warning("%-7s | %13v | %v", req.Method, latency, req.URL)
	}()
	WriteError(w, req, sdk.ErrNotFound)
}
