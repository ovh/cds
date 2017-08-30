package main

import (
	"compress/gzip"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	router    *Router
	panicked  bool
	nbPanic   int
	lastPanic *time.Time
)

const nbPanicsBeforeFail = 50

// Handler defines the HTTP handler used in CDS engine
type Handler func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error

// RouterConfigParam is the type of anonymous function returned by POST, GET and PUT functions
type RouterConfigParam func(rc *RouterConfig)

// RouterConfig contains a map of handler configuration. Key is the method of the http route
type RouterConfig struct {
	config map[string]*HandlerConfig
}

// HandlerConfig is the configuration for one handler
type HandlerConfig struct {
	auth                bool
	isExecution         bool
	needAdmin           bool
	needUsernameOrAdmin bool
	needHatchery        bool
	needWorker          bool
	method              string
	handler             Handler
	isDeprecated        bool
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
				log.Error("[PANIC_RECOVERY] Panic occurred on %s:%s, recover %s", req.Method, req.URL.String(), err)
				trace := make([]byte, 4096)
				count := runtime.Stack(trace, true)
				log.Error("[PANIC_RECOVERY] Stacktrace of %d bytes\n%s\n", count, trace)

				//Reinit database connection
				if _, e := database.Init(
					viper.GetString(viperDBUser),
					viper.GetString(viperDBPassword),
					viper.GetString(viperDBName),
					viper.GetString(viperDBHost),
					viper.GetString(viperDBPort),
					viper.GetString(viperDBSSLMode),
					viper.GetInt(viperDBTimeout),
					viper.GetInt(viperDBMaxConn),
				); e != nil {
					log.Error("[PANIC_RECOVERY] Unable to reinit db connection: %s", e)
				}

				//Checking if there are two much panics in two minutes
				//If last panic was more than 2 minutes ago, reinit the panic counter
				if lastPanic == nil {
					nbPanic = 0
				} else {
					dur := time.Since(*lastPanic)
					if dur.Minutes() > float64(2) {
						log.Info("[PANIC_RECOVERY] Last panic was %d seconds ago", int(dur.Seconds()))
						nbPanic = 0
					}
				}

				nbPanic++
				now := time.Now()
				lastPanic = &now
				//If two much panic, change the status of /mon/status with panicked = true
				if nbPanic > nbPanicsBeforeFail {
					panicked = true
					log.Error("[PANIC_RECOVERY] RESTART NEEDED")
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
	url        string
}

func newRouter(a auth.Driver, m *mux.Router, p string) *Router {
	return &Router{a, m, p, ""}
}

var mapRouterConfigs = map[string]*RouterConfig{}

// HandlerConfigParam is a type used in handler configuration, to set specific config on a route given a method
type HandlerConfigParam func(*HandlerConfig)

// HandlerConfigFunc is a type used in the router configuration fonction "Handle"
type HandlerConfigFunc func(Handler, ...HandlerConfigParam) *HandlerConfig

// Handle adds all handler for their specific verb in gorilla router for given uri
func (r *Router) Handle(uri string, handlers ...*HandlerConfig) {
	uri = r.prefix + uri
	cfg := &RouterConfig{
		config: map[string]*HandlerConfig{},
	}
	mapRouterConfigs[uri] = cfg

	for i := range handlers {
		cfg.config[handlers[i].method] = handlers[i]
	}

	f := func(w http.ResponseWriter, req *http.Request) {
		// Close indicates  to close the connection after replying to this request
		req.Close = true
		// Authorization
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", "GET,OPTIONS,PUT,POST,DELETE")
		w.Header().Add("Access-Control-Allow-Headers", "Accept, Origin, Referer, User-Agent, Content-Type, Authorization, Session-Token, Last-Event-Id, If-Modified-Since, Content-Disposition")
		w.Header().Add("Access-Control-Expose-Headers", "Accept, Origin, Referer, User-Agent, Content-Type, Authorization, Session-Token, Last-Event-Id, ETag, Content-Disposition")
		w.Header().Add("X-Api-Time", time.Now().Format(time.RFC3339))
		w.Header().Add("ETag", fmt.Sprintf("%d", time.Now().Unix()))

		if req.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		rc := cfg.config[req.Method]
		if rc == nil || rc.handler == nil {
			WriteError(w, req, sdk.ErrNotFound)
			return
		}

		start := time.Now()
		defer func() {
			end := time.Now()
			latency := end.Sub(start)
			if rc.isDeprecated {
				log.Error("%-7s | %13v | DEPRECATED ROUTE | %v", req.Method, latency, req.URL)
				w.Header().Add("X-CDS-WARNING", "deprecated route")
			} else {
				log.Debug("%-7s | %13v | %v", req.Method, latency, req.URL)
			}
		}()

		c := &businesscontext.Ctx{}

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
				WriteError(w, req, sdk.WrapError(sdk.ErrUnauthorized, "Router> Authorization denied on %s %s for %s agent %s : %s", req.Method, req.URL, req.RemoteAddr, c.Agent, err))
				return
			}
		}

		if c.User != nil {
			if err := loadUserPermissions(db, c.User); err != nil {
				WriteError(w, req, sdk.WrapError(sdk.ErrUnauthorized, "Router> Unable to load user %s permission: %s", c.User.ID, err))
				return
			}
		}

		if c.Hatchery != nil {
			g, err := loadGroupPermissions(db, c.Hatchery.GroupID)
			if err != nil {
				WriteError(w, req, sdk.WrapError(sdk.ErrUnauthorized, "Router> cannot load group permissions for GroupID %d err:%s", c.Hatchery.GroupID, err))
				return
			}
			c.User.Groups = append(c.User.Groups, *g)
		}

		if c.Worker != nil {
			if err := worker.RefreshWorker(db, c.Worker.ID); err != nil {
				WriteError(w, req, sdk.WrapError(err, "Router> Unable to refresh worker"))
				return
			}

			g, err := loadGroupPermissions(db, c.Worker.GroupID)
			if err != nil {
				WriteError(w, req, sdk.WrapError(sdk.ErrUnauthorized, "Router> cannot load group permissions: %s", err))
				return
			}
			c.User.Groups = append(c.User.Groups, *g)

			if c.Worker.Model != 0 {
				//Load model
				m, err := worker.LoadWorkerModelByID(db, c.Worker.Model)
				if err != nil {
					WriteError(w, req, sdk.WrapError(sdk.ErrUnauthorized, "Router> cannot load worker: %s", err))
					return
				}

				//If worker model is owned by shared.infra, let's add SharedInfraGroup in user's group
				if m.GroupID == group.SharedInfraGroup.ID {
					c.User.Groups = append(c.User.Groups, *group.SharedInfraGroup)
				} else {
					log.Debug("Router> loading groups permission for model %d", c.Worker.Model)
					modelGroup, errLoad2 := loadGroupPermissions(db, m.GroupID)
					if errLoad2 != nil {
						WriteError(w, req, sdk.WrapError(sdk.ErrUnauthorized, "Router> Cannot load group: %s", errLoad2))
						return
					}
					//Anyway, add the group of the model as a group of the user
					c.User.Groups = append(c.User.Groups, *modelGroup)
				}
			}
		}

		permissionOk := false
		if !rc.auth {
			permissionOk = true
		} else {
			if rc.needHatchery && c.Hatchery != nil {
				permissionOk = true
			}
			if rc.needWorker {
				permissionOk = checkWorkerPermission(db, rc, mux.Vars(req), c)
			}

			if rc.needUsernameOrAdmin && (c.User.Admin || (c.User.Username == mux.Vars(req)["username"])) {
				// get / update / delete user -> for admin or current user
				// if not admin and currentUser != username in request -> ko
				permissionOk = true
			}

			if rc.needAdmin && c.User.Admin {
				permissionOk = true
			}
			if !rc.needAdmin && !c.User.Admin {
				permissionOk = checkPermission(mux.Vars(req), c, getPermissionByMethod(req.Method, rc.isExecution))
			}

			// else case, just need auth
			if !rc.needAdmin && !rc.needHatchery && !rc.needWorker && !rc.needUsernameOrAdmin {
				permissionOk = true
			}
		}

		if !permissionOk {
			WriteError(w, req, sdk.ErrForbidden)
			return
		}

		if err := rc.handler(w, req, db, c); err != nil {
			WriteError(w, req, err)
			return
		}

		if req.Method == "POST" || req.Method == "PUT" || req.Method == "DELETE" {
			deleteUserPermissionCache(c)
		}
	}
	router.mux.HandleFunc(uri, compress(recoverWrap(f)))
}

// DEPRECATED marks the handler as deprecated
var DEPRECATED = func(rc *HandlerConfig) {
	rc.isDeprecated = true
}

// GET will set given handler only for GET request
func GET(h Handler, cfg ...HandlerConfigParam) *HandlerConfig {
	rc := new(HandlerConfig)
	rc.handler = h
	rc.auth = true
	rc.method = "GET"
	for _, c := range cfg {
		c(rc)
	}
	return rc
}

// POST will set given handler only for POST request
func POST(h Handler, cfg ...HandlerConfigParam) *HandlerConfig {
	rc := new(HandlerConfig)
	rc.handler = h
	rc.auth = true
	rc.method = "POST"
	for _, c := range cfg {
		c(rc)
	}
	return rc
}

// POSTEXECUTE will set given handler only for POST request and add a flag for execution permission
func POSTEXECUTE(h Handler, cfg ...HandlerConfigParam) *HandlerConfig {
	rc := new(HandlerConfig)
	rc.handler = h
	rc.auth = true
	rc.method = "POST"
	rc.isExecution = true
	for _, c := range cfg {
		c(rc)
	}
	return rc
}

// PUT will set given handler only for PUT request
func PUT(h Handler, cfg ...HandlerConfigParam) *HandlerConfig {
	rc := new(HandlerConfig)
	rc.handler = h
	rc.auth = true
	rc.method = "PUT"
	for _, c := range cfg {
		c(rc)
	}
	return rc
}

// DELETE will set given handler only for DELETE request
func DELETE(h Handler, cfg ...HandlerConfigParam) *HandlerConfig {
	rc := new(HandlerConfig)
	rc.handler = h
	rc.auth = true
	rc.method = "DELETE"
	for _, c := range cfg {
		c(rc)
	}
	return rc
}

// NeedAdmin set the route for cds admin only (or not)
func NeedAdmin(admin bool) HandlerConfigParam {
	f := func(rc *HandlerConfig) {
		rc.needAdmin = admin
	}
	return f
}

// NeedUsernameOrAdmin set the route for cds admin or current user = username called on route
func NeedUsernameOrAdmin(need bool) HandlerConfigParam {
	f := func(rc *HandlerConfig) {
		rc.needUsernameOrAdmin = need
	}
	return f
}

// NeedHatchery set the route for hatchery only
func NeedHatchery() HandlerConfigParam {
	f := func(rc *HandlerConfig) {
		rc.needHatchery = true
	}
	return f
}

// NeedWorker set the route for worker only
func NeedWorker() HandlerConfigParam {
	f := func(rc *HandlerConfig) {
		rc.needWorker = true
	}
	return f
}

// Auth set manually whether authorisation layer should be applied
// Authorization is enabled by default
func Auth(v bool) HandlerConfigParam {
	f := func(rc *HandlerConfig) {
		rc.auth = v
	}
	return f
}

func (r *Router) checkAuthentication(db *gorp.DbMap, headers http.Header, c *businesscontext.Ctx) error {
	c.Agent = headers.Get("User-Agent")

	switch headers.Get("User-Agent") {
	case sdk.HatcheryAgent:
		return auth.CheckHatcheryAuth(db, headers, c)
	case sdk.WorkerAgent:
		return auth.CheckWorkerAuth(db, headers, c)
	default:
		return r.authDriver.CheckAuthHeader(db, headers, c)
	}
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

/*
// businessContext completes the business context with functionnal stuff
func businessContext(db gorp.SqlExecutor, c *businesscontext.Ctx, req *http.Request, w http.ResponseWriter) error {

	if c.Hatchery != nil {
		g, err := loadGroupPermissions(db, c.Hatchery.GroupID)
		if err != nil {
			log.Warning("Router> cannot load group permissions for GroupID %d err:%s", c.Hatchery.GroupID, err)
			return sdk.ErrUnauthorized
		}
		c.User.Groups = append(c.User.Groups, *g)
	}

	if c.Worker != nil {
		if err := worker.RefreshWorker(db, c.Worker.ID); err != nil {
			log.Warning("Router> Unable to refresh worker: %s", err)
			return err
		}

		g, err := loadGroupPermissions(db, c.Worker.GroupID)
		if err != nil {
			log.Warning("Router> cannot load group permissions: %s", err)
			return sdk.ErrUnauthorized
		}
		c.User.Groups = append(c.User.Groups, *g)

		if c.Worker.Model != 0 {
			//Load model
			m, err := worker.LoadWorkerModelByID(db, c.Worker.Model)
			if err != nil {
				log.Warning("Router> cannot load worker: %s", err)
				return sdk.ErrUnauthorized
			}

			//If worker model is owned by shared.infra, let's add SharedInfraGroup in user's group
			if m.GroupID == group.SharedInfraGroup.ID {
				c.User.Groups = append(c.User.Groups, *group.SharedInfraGroup)
			} else {
				log.Debug("Router> loading groups permission for model %d", c.Worker.Model)
				modelGroup, errLoad2 := loadGroupPermissions(db, m.GroupID)
				if errLoad2 != nil {
					log.Warning("Router> Cannot load group: %s", errLoad2)
					return sdk.ErrUnauthorized
				}
				//Anyway, add the group of the model as a group of the user
				c.User.Groups = append(c.User.Groups, *modelGroup)
			}
		}
	}

	if c.User != nil {
		vars := mux.Vars(req)
		key := vars["key"]
		if key == "" {
			key = vars["permProjectKey"]
		}

		if key != "" {
			proj, errproj := project.Load(db, key, c.User, project.LoadOptions.Default)
			if errproj != nil {
				return errproj
			}
			c.Project = proj
		}

		app := vars["permApplicationName"]
		if app == "" {
			app = vars["app"]
		}

		if app != "" {
			app, errapp := application.LoadByName(db, key, app, c.User, application.LoadOptions.Default)
			if errapp != nil {
				return errapp
			}
			c.Application = app
		}
	}

	return nil
}
*/
