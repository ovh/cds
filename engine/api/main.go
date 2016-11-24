package main

import (
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	_ "github.com/proullon/ramsql/driver"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/api/archivist"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/repositoriesmanager/polling"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/stats"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
)

var startupTime time.Time
var b *Broker
var baseURL string
var localCLientAuthMode = auth.LocalClientBasicAuthMode

var mainCmd = &cobra.Command{
	Use:   "api",
	Short: "CDS Engine",
	Run: func(cmd *cobra.Command, args []string) {
		viper.SetEnvPrefix("cds")
		viper.AutomaticEnv()

		log.Initialize()
		log.Notice("Starting CDS server...\n")

		startupTime = time.Now()

		if err := mail.CheckMailConfiguration(); err != nil {
			log.Fatalf("SMTP configuration error: %s\n", err)
		}

		if err := objectstore.Initialize(
			viper.GetString("artifact_mode"),
			viper.GetString("artifact_address"),
			viper.GetString("artifact_user"),
			viper.GetString("artifact_password"),
			viper.GetString("artifact_basedir")); err != nil {
			log.Fatalf("Cannot initialize storage: %s\n", err)
		}

		db, err := database.Init()
		if err != nil {
			log.Warning("Cannot connect to database: %s\n", err)
		}

		if db != nil {
			if viper.GetBool("db_logging") {
				log.UseDatabaseLogger(db)
			}

			if err = bootstrap.InitiliazeDB(db); err != nil {
				log.Critical("Cannot setup databases: %s\n", err)
			}

			// Gracefully shutdown sql connections
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
			signal.Notify(c, syscall.SIGTERM)
			signal.Notify(c, syscall.SIGKILL)
			go func() {
				<-c
				log.Warning("Cleanup SQL connections\n")
				db.Close()
				os.Exit(0)
			}()
		}

		// Make a new Broker instance
		b = &Broker{
			make(map[chan string]bool),
			make(chan (chan string)),
			make(chan (chan string)),
			make(chan string),
		}

		b.Start()

		router = &Router{
			mux: mux.NewRouter(),
		}
		router.init()

		baseURL = viper.GetString("base_url")
		notification.Initialize(viper.GetString("notifs_urls"), viper.GetString("notifs_key"), baseURL)

		//Initialize secret driver
		secretBackend := viper.GetString("secret_backend")
		secretBackendOptions := viper.GetStringSlice("secret_backend_option")
		secretBackendOptionsMap := map[string]string{}
		for _, o := range secretBackendOptions {
			if !strings.Contains(o, "=") {
				log.Warning("Malformated options : %s", o)
				continue
			}
			t := strings.Split(o, "=")
			secretBackendOptionsMap[t[0]] = t[1]
		}
		if err := secret.Init(secretBackend, secretBackendOptionsMap); err != nil {
			log.Critical("Cannot initialize secret manager: %s\n", err)
		}

		//Intialize repositories manager
		if err := repositoriesmanager.Initialize(
			secret.Client,
			viper.GetString("keys_directory"),
			baseURL,
			viper.GetString("api_url"),
		); err != nil {
			log.Warning("Error initializing repositories manager connections: %s\n", err)
		}

		// Initialize the auth driver
		var authMode string
		var authOptions interface{}
		switch viper.GetBool("ldap_enable") {
		case true:
			authMode = "ldap"
			authOptions = auth.LDAPConfig{
				Host:         viper.GetString("ldap_host"),
				Port:         viper.GetInt("ldap_port"),
				Base:         viper.GetString("ldap_base"),
				DN:           viper.GetString("ldap_dn"),
				SSL:          viper.GetBool("ldap_ssl"),
				UserFullname: viper.GetString("ldap_user_fullname"),
			}
		default:
			authMode = "local"
		}

		storeOptions := sessionstore.Options{
			Mode:          viper.GetString("cache"),
			TTL:           viper.GetInt("session_ttl"),
			RedisHost:     viper.GetString("redis_host"),
			RedisPassword: viper.GetString("redis_password"),
		}

		router.authDriver, _ = auth.GetDriver(authMode, authOptions, storeOptions)

		cache.Initialize(viper.GetString("cache"), viper.GetString("redis_host"), viper.GetString("redis_password"), viper.GetInt("cache_ttl"))

		go archivist.Archive(viper.GetInt("interval_archive_seconds"), viper.GetInt("archived_build_hours"))
		go scheduler.Schedule()
		go pipeline.AWOLPipelineKiller()
		//go pipeline.HistoryCleaningRoutine(db)
		go worker.Heartbeat()
		go hatchery.Heartbeat()
		go log.RemovalRoutine()
		go auditCleanerRoutine()
		go repositoriesmanager.RepositoriesCacheLoader(30)
		go stats.StartRoutine()
		go worker.UpdateModelCapabilitiesCache()
		go worker.UpdateActionRequirementsCache()
		go hookRecoverer()
		go polling.Initialize()
		go polling.ExecutionCleaner()

		s := &http.Server{
			Addr:           ":" + viper.GetString("listen_port"),
			Handler:        router.mux,
			ReadTimeout:    10 * time.Minute,
			WriteTimeout:   10 * time.Minute,
			MaxHeaderBytes: 1 << 20,
		}

		log.Notice("Listening on :%s\n", viper.GetString("listen_port"))
		if err := s.ListenAndServe(); err != nil {
			log.Fatalf("Cannot start cds-server: %s\n", err)
		}
	},
}

func (router *Router) init() {
	router.Handle("/login", Auth(false), POST(LoginUser))

	// Action
	router.Handle("/action", GET(getActionsHandler))
	router.Handle("/action/import", NeedAdmin(true), POST(importActionHandler))
	router.Handle("/action/requirement", Auth(false), GET(getActionsRequirements))
	router.Handle("/action/{permActionName}", GET(getActionHandler), POST(addActionHandler), PUT(updateActionHandler), DELETE(deleteActionHandler))
	router.Handle("/action/{actionName}/using", NeedAdmin(true), GET(getPipelinesUsingActionHandler))
	router.Handle("/action/{actionID}/audit", NeedAdmin(true), GET(getActionAuditHandler))

	// Action plugin
	router.Handle("/plugin", NeedAdmin(true), POST(addPluginHandler), PUT(updatePluginHandler))
	router.Handle("/plugin/{name}", NeedAdmin(true), DELETE(deletePluginHandler))
	router.Handle("/plugin/download/{name}", GET(downloadPluginHandler))

	// Download file
	router.ServeAbsoluteFile("/download/cli/x86_64", path.Join(viper.GetString("download_directory"), "cds"), "cds")
	router.ServeAbsoluteFile("/download/worker/x86_64", path.Join(viper.GetString("download_directory"), "worker"), "worker")
	router.ServeAbsoluteFile("/download/worker/windows_x86_64", path.Join(viper.GetString("download_directory"), "worker.exe"), "worker.exe")
	router.ServeAbsoluteFile("/download/hatchery/x86_64", path.Join(viper.GetString("download_directory"), "hatchery", "x86_64"), "hatchery")

	// Group
	router.Handle("/group", GET(getGroups), POST(addGroupHandler))
	router.Handle("/group/{permGroupName}", GET(getGroupHandler), PUT(updateGroupHandler), DELETE(deleteGroupHandler))
	router.Handle("/group/{permGroupName}/user", POST(addUserInGroup))
	router.Handle("/group/{permGroupName}/user/{user}", DELETE(removeUserFromGroupHandler))
	router.Handle("/group/{permGroupName}/user/{user}/admin", POST(setUserGroupAdminHandler), DELETE(removeUserGroupAdminHandler))
	router.Handle("/group/{permGroupName}/token/{expiration}", POST(generateTokenHandler))

	// Hatchery
	router.Handle("/hatchery", Auth(false), POST(registerHatchery))
	router.Handle("/hatchery/{id}", PUT(refreshHatcheryHandler))

	// Hooks
	router.Handle("/hook", Auth(false) /* Public handler called by third parties */, POST(receiveHook))

	// Overall health
	router.Handle("/mon/status", Auth(false), GET(statusHandler))
	router.Handle("/mon/status/polling", Auth(false), GET(pollinStatusHandler))
	router.Handle("/mon/smtp/ping", Auth(true), GET(smtpPingHandler))
	router.Handle("/mon/log/{level}", Auth(false), POST(setEngineLogLevel))
	router.Handle("/mon/sla/{date}", POST(slaHandler))
	router.Handle("/mon/version", Auth(false), GET(getVersionHandler))
	router.Handle("/mon/error", Auth(false), GET(getError))
	router.Handle("/mon/stats", Auth(false), GET(getStats))
	router.Handle("/mon/models", Auth(false), GET(getWorkerModelsStatsHandler))
	router.Handle("/mon/building", GET(getBuildingPipelines))
	router.Handle("/mon/building/{hash}", GET(getPipelineBuildingCommit))
	router.Handle("/mon/warning", GET(getUserWarnings))
	router.Handle("/mon/lastupdates", GET(getUserLastUpdates))

	// Notif builtin from worker
	router.Handle("/notif/{actionBuildId}", POST(notifHandler))

	// Project
	router.Handle("/project", GET(getProjects), POST(addProject))
	router.Handle("/project/{permProjectKey}", GET(getProject), PUT(updateProject), DELETE(deleteProject))
	router.Handle("/project/{permProjectKey}/group", POST(addGroupInProject), PUT(updateGroupsInProject))
	router.Handle("/project/{permProjectKey}/group/{group}", PUT(updateGroupRoleOnProjectHandler), DELETE(deleteGroupFromProjectHandler))
	router.Handle("/project/{permProjectKey}/variable", GET(getVariablesInProjectHandler), PUT(updateVariablesInProjectHandler))
	router.Handle("/project/{key}/variable/audit", GET(getVariablesAuditInProjectnHandler))
	router.Handle("/project/{key}/variable/audit/{auditID}", PUT(restoreProjectVariableAuditHandler))
	router.Handle("/project/{permProjectKey}/variable/{name}", GET(getVariableInProjectHandler), POST(addVariableInProjectHandler), PUT(updateVariableInProjectHandler), DELETE(deleteVariableFromProjectHandler))
	router.Handle("/project/{permProjectKey}/applications", GET(getApplicationsHandler), POST(addApplicationHandler))

	// Application
	router.Handle("/project/{key}/application/{permApplicationName}", GET(getApplicationHandler), PUT(updateApplicationHandler), DELETE(deleteApplicationHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/branches", GET(getApplicationBranchHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/version", GET(getApplicationBranchVersionHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/clone", POST(cloneApplicationHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/group", POST(addGroupInApplicationHandler), PUT(updateGroupsInApplicationHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/group/{group}", PUT(updateGroupRoleOnApplicationHandler), DELETE(deleteGroupFromApplicationHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/history", GET(getApplicationHistoryHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/history/branch", GET(getPipelineBuildBranchHistoryHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/history/env/deploy", GET(getApplicationDeployHistoryHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline", GET(getPipelinesInApplicationHandler), PUT(updatePipelinesToApplicationHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}", POST(attachPipelineToApplicationHandler), PUT(updatePipelineToApplicationHandler), DELETE(removePipelineFromApplicationHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/notification", GET(getUserNotificationApplicationPipelineHandler), PUT(updateUserNotificationApplicationPipelineHandler), DELETE(deleteUserNotificationApplicationPipelineHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/tree", GET(getApplicationTreeHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/variable", GET(getVariablesInApplicationHandler), PUT(updateVariablesInApplicationHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/variable/audit", GET(getVariablesAuditInApplicationHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/variable/audit/{auditID}", PUT(restoreAuditHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/variable/{name}", GET(getVariableInApplicationHandler), POST(addVariableInApplicationHandler), PUT(updateVariableInApplicationHandler), DELETE(deleteVariableFromApplicationHandler))

	// Pipeline
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/history", GET(getPipelineHistoryHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/log", GET(getBuildLogsHandler))
	router.Handle("/project/{key}/application/{app}/pipeline/{permPipelineKey}/build/{build}/test", POSTEXECUTE(addBuildTestResultsHandler), GET(getBuildTestResultsHandler))
	router.Handle("/project/{key}/application/{app}/pipeline/{permPipelineKey}/build/{build}/variable", POSTEXECUTE(addBuildVariableHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/action/{actionID}/log", GET(getActionBuildLogsHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}", GET(getBuildStateHandler), DELETE(deleteBuildHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/triggered", GET(getPipelineBuildTriggeredHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/stop", POSTEXECUTE(stopPipelineBuildHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/restart", POSTEXECUTE(restartPipelineBuildHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/commits", GET(getPipelineBuildCommitsHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/commits", GET(getPipelineCommitsHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/run", POSTEXECUTE(runPipelineHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/runwithlastparent", POSTEXECUTE(runPipelineWithLastParentHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/rollback", POSTEXECUTE(rollbackPipelineHandler))
	router.Handle("/project/{permProjectKey}/pipeline", GET(getPipelinesHandler), POST(addPipeline))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/application", GET(getApplicationUsingPipelineHandler))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/group", POST(addGroupInPipelineHandler), PUT(updateGroupsOnPipelineHandler))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/group/{group}", PUT(updateGroupRoleOnPipelineHandler), DELETE(deleteGroupFromPipelineHandler))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/parameter", GET(getParametersInPipelineHandler), PUT(updateParametersInPipelineHandler))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/parameter/{name}", POST(addParameterInPipelineHandler), PUT(updateParameterInPipelineHandler), DELETE(deleteParameterFromPipelineHandler))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}", GET(getPipelineHandler), PUT(updatePipelineHandler), DELETE(deletePipeline))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/stage", POST(addStageHandler))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/move", POST(moveStageHandler))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}", GET(getStageHandler), PUT(updateStageHandler), DELETE(deleteStageHandler))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/job", POST(addJobToStageHandler))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/job/{jobID}", PUT(updateJobHandler), DELETE(deleteJobHandler))

	// DEPRECATED
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/action/{pipelineActionID}", PUT(updatePipelineActionHandler), DELETE(deletePipelineActionHandler))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/joined", POST(addJoinedActionToPipelineHandler))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/joined/{actionID}", GET(getJoinedAction), PUT(updateJoinedAction), DELETE(deleteJoinedAction))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/joined/{actionID}/audit", GET(getJoinedActionAudithandler))

	// Triggers
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/trigger", GET(getTriggersHandler), POST(addTriggerHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/trigger/source", GET(getTriggersAsSourceHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/trigger/{id}", GET(getTriggerHandler), DELETE(deleteTriggerHandler), PUT(updateTriggerHandler))

	// Environment
	router.Handle("/project/{permProjectKey}/environment", GET(getEnvironmentsHandler), POST(addEnvironmentHandler), PUT(updateEnvironmentsHandler))
	router.Handle("/project/{key}/environment/{permEnvironmentName}", GET(getEnvironmentHandler), PUT(updateEnvironmentHandler), DELETE(deleteEnvironmentHandler))
	router.Handle("/project/{key}/environment/{permEnvironmentName}/audit", GET(getEnvironmentsAuditHandler))
	router.Handle("/project/{key}/environment/{permEnvironmentName}/audit/{auditID}", PUT(restoreEnvironmentAuditHandler))
	router.Handle("/project/{key}/environment/{permEnvironmentName}/group", POST(addGroupInEnvironmentHandler))
	router.Handle("/project/{key}/environment/{permEnvironmentName}/group/{group}", PUT(updateGroupRoleOnEnvironmentHandler), DELETE(deleteGroupFromEnvironmentHandler))
	router.Handle("/project/{key}/environment/{permEnvironmentName}/variable", GET(getVariablesInEnvironmentHandler))
	router.Handle("/project/{key}/environment/{permEnvironmentName}/variable/{name}", POST(addVariableInEnvironmentHandler), PUT(updateVariableInEnvironmentHandler), DELETE(deleteVariableFromEnvironmentHandler))

	// Artifacts
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/artifact/{tag}", GET(listArtifactsHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/{buildNumber}/artifact", GET(listArtifactsBuildHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/{buildNumber}/artifact/{tag}", POSTEXECUTE(uploadArtifactHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/artifact/download/{id}", GET(downloadArtifactHandler))
	router.Handle("/artifact/{hash}", Auth(false), GET(downloadArtifactDirectHandler))

	// Hooks
	router.Handle("/project/{key}/application/{permApplicationName}/hook", GET(getApplicationHooksHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/hook", POST(addHook), GET(getHooks))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/hook/{id}", PUT(updateHookHandler), DELETE(deleteHook))

	// Pollers
	router.Handle("/project/{key}/application/{permApplicationName}/polling", GET(getApplicationPollersHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/polling", POST(addPollerHandler), GET(getPollersHandler), PUT(updatePollerHandler), DELETE(deletePollerHandler))

	// Build queue
	router.Handle("/queue", GET(getQueueHandler))
	router.Handle("/queue/requirements/errors", POST(requirementsErrorHandler))
	router.Handle("/queue/{id}/take", POST(takeActionBuildHandler))
	router.Handle("/queue/{id}/result", POST(addQueueResultHandler))
	router.Handle("/build/{id}/log", POST(addBuildLogHandler))

	router.Handle("/variable/type", GET(getVariableTypeHandler))
	router.Handle("/parameter/type", GET(getParameterTypeHandler))
	router.Handle("/pipeline/type", GET(getPipelineTypeHandler))
	router.Handle("/notification/type", GET(getUserNotificationTypeHandler))
	router.Handle("/notification/state", GET(getUserNotificationStateValueHandler))

	// RepositoriesManager
	router.Handle("/repositories_manager", GET(getRepositoriesManagerHandler))
	router.Handle("/repositories_manager/add", NeedAdmin(true), POST(addRepositoriesManagerHandler))
	router.Handle("/repositories_manager/oauth2/callback", Auth(false), GET(repositoriesManagerOAuthCallbackHandler))
	// RepositoriesManager for projects
	router.Handle("/project/{permProjectKey}/repositories_manager", GET(getRepositoriesManagerForProjectHandler))
	router.Handle("/project/{permProjectKey}/repositories_manager/{name}/authorize", POST(repositoriesManagerAuthorize))
	router.Handle("/project/{permProjectKey}/repositories_manager/{name}/authorize/callback", POST(repositoriesManagerAuthorizeCallback))
	router.Handle("/project/{permProjectKey}/repositories_manager/{name}", DELETE(deleteRepositoriesManagerHandler))
	router.Handle("/project/{permProjectKey}/repositories_manager/{name}/repo", GET(getRepoFromRepositoriesManagerHandler))
	router.Handle("/project/{permProjectKey}/repositories_manager/{name}/repo/commits", GET(getCommitsHandler))
	router.Handle("/project/{permProjectKey}/repositories_manager/{name}/repos", GET(getReposFromRepositoriesManagerHandler))

	// RepositoriesManager for applications
	router.Handle("/project/{permProjectKey}/repositories_manager/{name}/application", POST(addApplicationFromRepositoriesManagerHandler))
	router.Handle("/project/{key}/repositories_manager/{name}/application/{permApplicationName}/attach", POST(attachRepositoriesManager))
	router.Handle("/project/{key}/repositories_manager/{name}/application/{permApplicationName}/detach", POST(detachRepositoriesManager))
	router.Handle("/project/{key}/application/{permApplicationName}/repositories_manager", GET(getRepositoriesManagerForApplicationsHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/repositories_manager/{name}/commits", GET(getApplicationCommitsHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/repositories_manager/{name}/hook", POST(addHookOnRepositoriesManagerHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/repositories_manager/{name}/hook/{hookId}", DELETE(deleteHookOnRepositoriesManagerHandler))

	//Suggest
	router.Handle("/suggest/variable/{permProjectKey}", GET(getVariablesHandler))

	// Templates
	router.Handle("/template", Auth(false), GET(getTemplatesHandler))
	router.Handle("/template/add", NeedAdmin(true), POST(addTemplateHandler))
	router.Handle("/template/build", Auth(false), GET(getBuildTemplatesHandler))
	router.Handle("/template/deploy", Auth(false), GET(getDeployTemplatesHandler))
	router.Handle("/template/{id}", NeedAdmin(true), PUT(updateTemplateHandler), DELETE(deleteTemplateHandler))
	router.Handle("/project/{permProjectKey}/template", POST(applyTemplateHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/template", POST(applyTemplateOnApplicationHandler))

	// Users
	router.Handle("/user", GET(GetUsers))
	router.Handle("/user/signup", Auth(false), POST(AddUser))
	router.Handle("/user/{name}", NeedAdmin(true), GET(GetUserHandler), PUT(UpdateUserHandler), DELETE(DeleteUserHandler))
	router.Handle("/user/{name}/confirm/{token}", Auth(false), GET(ConfirmUser))
	router.Handle("/user/{name}/reset", Auth(false), POST(ResetUser))
	router.Handle("/auth/mode", Auth(false), GET(AuthModeHandler))

	// Workers
	router.Handle("/worker", Auth(false), GET(getWorkersHandler), POST(registerWorkerHandler))
	router.Handle("/worker/status", GET(getWorkerModelStatus))
	router.Handle("/worker/refresh", POST(refreshWorkerHandler))
	router.Handle("/worker/unregister", POST(unregisterWorkerHandler))
	router.Handle("/worker/{id}/disable", POST(disableWorkerHandler))
	router.Handle("/worker/model", POST(addWorkerModel), GET(getWorkerModels))
	router.Handle("/worker/model/type", GET(getWorkerModelTypes))
	router.Handle("/worker/model/{permModelID}", PUT(updateWorkerModel), DELETE(deleteWorkerModel))
	router.Handle("/worker/model/{permModelID}/capability", POST(addWorkerModelCapa))
	router.Handle("/worker/model/capability/type", GET(getWorkerModelCapaTypes))
	router.Handle("/worker/model/{permModelID}/capability/{capa}", PUT(updateWorkerModelCapa), DELETE(deleteWorkerModelCapa))
}

func init() {
	pflags := mainCmd.PersistentFlags()
	pflags.String("db-user", "cds", "DB User")
	pflags.String("db-password", "", "DB Password")
	pflags.String("db-name", "cds", "DB Name")
	pflags.String("db-host", "localhost", "DB Host")
	pflags.String("db-port", "5432", "DB Port")
	pflags.String("db-sslmode", "require", "DB SSL Mode: require (default), verify-full, or disable")
	pflags.Int("db-maxconn", 20, "DB Max connection")
	pflags.Int("db-timeout", 3000, "Statement timeout value")
	viper.BindPFlag("db_user", pflags.Lookup("db-user"))
	viper.BindPFlag("db_password", pflags.Lookup("db-password"))
	viper.BindPFlag("db_name", pflags.Lookup("db-name"))
	viper.BindPFlag("db_host", pflags.Lookup("db-host"))
	viper.BindPFlag("db_port", pflags.Lookup("db-port"))
	viper.BindPFlag("db_sslmode", pflags.Lookup("db-sslmode"))
	viper.BindPFlag("db_maxconn", pflags.Lookup("db-maxconn"))
	viper.BindPFlag("db_timeout", pflags.Lookup("db-timeout"))

	flags := mainCmd.Flags()

	flags.String("log-level", "notice", "Log Level : debug, info, notice, warning, critical")
	viper.BindPFlag("log_level", flags.Lookup("log-level"))

	flags.Bool("db-logging", false, "Logging in Database: true of false")
	viper.BindPFlag("db_logging", flags.Lookup("db-logging"))

	flags.String("base-url", "", "CDS UI Base URL")
	viper.BindPFlag("base_url", flags.Lookup("base-url"))

	flags.String("api-url", "", "CDS API Base URL")
	viper.BindPFlag("api_url", flags.Lookup("api-url"))

	flags.String("listen-port", "8081", "CDS Engine Listen Port")
	viper.BindPFlag("listen_port", flags.Lookup("listen-port"))

	flags.String("artifact-mode", "filesystem", "Artifact Mode: openstack or filesystem")
	flags.String("artifact-address", "", "Artifact Adress: used with --artifact-mode=openstask")
	flags.String("artifact-user", "", "Artifact User: used with --artifact-mode=openstask")
	flags.String("artifact-password", "", "Artifact Password: used with --artifact-mode=openstask")
	flags.String("artifact-basedir", "/tmp", "Artifact Basedir: used with --artifact-mode=filesystem")
	viper.BindPFlag("artifact_mode", flags.Lookup("artifact-mode"))
	viper.BindPFlag("artifact_address", flags.Lookup("artifact-address"))
	viper.BindPFlag("artifact_user", flags.Lookup("artifact-user"))
	viper.BindPFlag("artifact_password", flags.Lookup("artifact-password"))
	viper.BindPFlag("artifact_basedir", flags.Lookup("artifact-basedir"))

	flags.Bool("no-smtp", true, "No SMTP mode: true or false")
	flags.String("smtp-host", "", "SMTP Host")
	flags.String("smtp-port", "", "SMTP Port")
	flags.Bool("smtp-tls", false, "SMTP TLS")
	flags.String("smtp-user", "", "SMTP Username")
	flags.String("smtp-password", "", "SMTP Password")
	flags.String("smtp-from", "", "SMTP From")
	viper.BindPFlag("no_smtp", flags.Lookup("no-smtp"))
	viper.BindPFlag("smtp_host", flags.Lookup("smtp-host"))
	viper.BindPFlag("smtp_port", flags.Lookup("smtp-port"))
	viper.BindPFlag("smtp_tls", flags.Lookup("smtp-tls"))
	viper.BindPFlag("smtp_user", flags.Lookup("smtp-user"))
	viper.BindPFlag("smtp_password", flags.Lookup("smtp-password"))
	viper.BindPFlag("smtp_from", flags.Lookup("smtp-from"))

	flags.Int("interval-archive-seconds", 3600, "Interval of archive routine, in seconds")
	viper.BindPFlag("interval_archive_seconds", flags.Lookup("interval-archive-seconds"))

	flags.Int("archived-build-hours", 24, "After n hours, build is archived")
	viper.BindPFlag("archived_build_hours", flags.Lookup("archived-build-hours"))

	flags.String("download-directory", "/app", "Directory prefix for cds binaries")
	viper.BindPFlag("download_directory", flags.Lookup("download-directory"))

	flags.String("keys-directory", "/app/keys", "Directory keys for repositories managers")
	viper.BindPFlag("keys_directory", flags.Lookup("keys-directory"))

	flags.String("notifs-urls", "", "URLs of CDS Notifications: tat:http://<cds2tat>>,stash:http://<cds2stash>,jabber:http://<cds2xmpp>")
	viper.BindPFlag("notifs_urls", flags.Lookup("notifs-urls"))

	flags.String("notifs-key", "", "Key of CDS Notifications. Use Key of your deployed CDS Notifications microservices")
	viper.BindPFlag("notifs_key", flags.Lookup("notifs-key"))

	flags.Bool("ldap-enable", false, "Enable LDAP Auth mode : true|false")
	viper.BindPFlag("ldap_enable", flags.Lookup("ldap-enable"))

	flags.String("ldap-host", "", "LDAP Host")
	viper.BindPFlag("ldap_host", flags.Lookup("ldap-host"))

	flags.Int("ldap-port", 636, "LDAP Post")
	viper.BindPFlag("ldap_port", flags.Lookup("ldap-port"))

	flags.Bool("ldap-ssl", true, "LDAP SSL mode")
	viper.BindPFlag("ldap_ssl", flags.Lookup("ldap-ssl"))

	flags.String("ldap-base", "", "LDAP Base")
	viper.BindPFlag("ldap_base", flags.Lookup("ldap-base"))

	flags.String("ldap-dn", "uid=%s,ou=people,{{.ldap-base}}", "LDAP Bind DN")
	viper.BindPFlag("ldap_dn", flags.Lookup("ldap-dn"))

	flags.String("ldap-user-fullname", "{{.givenName}} {{.sn}}", "LDAP User fullname")
	viper.BindPFlag("ldap_user_fullname", flags.Lookup("ldap-user-fullname"))

	flags.String("secret-backend", "", "Secret Backend plugin")
	viper.BindPFlag("secret_backend", flags.Lookup("secret-backend"))

	flags.StringSlice("secret-backend-option", []string{}, "Secret Backend plugin options")
	viper.BindPFlag("secret_backend_option", flags.Lookup("secret-backend-option"))

	flags.String("redis-host", "localhost:6379", "Redis hostname")
	viper.BindPFlag("redis_host", flags.Lookup("redis-host"))

	flags.String("redis-password", "", "Redis password")
	viper.BindPFlag("redis_password", flags.Lookup("redis-password"))

	flags.String("cache", "local", "Cache : local|redis")
	viper.BindPFlag("cache", flags.Lookup("cache"))

	flags.Int("cache-ttl", 600, "Cache Time to Live (seconds)")
	viper.BindPFlag("cache_ttl", flags.Lookup("cache-ttl"))

	flags.Int("session-ttl", 60, "Session Time to Live (minutes)")
	viper.BindPFlag("session_ttl", flags.Lookup("session-ttl"))

	mainCmd.AddCommand(database.DBCmd)

}

func main() {
	mainCmd.Execute()
}
