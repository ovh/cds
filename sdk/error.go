package sdk

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/text/language"
)

// Existing CDS errors
// Note: the error id is useless except to ensure objects are different in map
var (
	ErrUnknownError                                  = Error{ID: 1, Status: http.StatusInternalServerError}
	ErrActionAlreadyUpdated                          = Error{ID: 2, Status: http.StatusBadRequest}
	ErrNoAction                                      = Error{ID: 3, Status: http.StatusNotFound}
	ErrActionLoop                                    = Error{ID: 4, Status: http.StatusBadRequest}
	ErrInvalidID                                     = Error{ID: 5, Status: http.StatusBadRequest}
	ErrInvalidProject                                = Error{ID: 6, Status: http.StatusBadRequest}
	ErrInvalidProjectKey                             = Error{ID: 7, Status: http.StatusBadRequest}
	ErrProjectHasPipeline                            = Error{ID: 8, Status: http.StatusConflict}
	ErrProjectHasApplication                         = Error{ID: 9, Status: http.StatusConflict}
	ErrUnauthorized                                  = Error{ID: 10, Status: http.StatusUnauthorized}
	ErrForbidden                                     = Error{ID: 11, Status: http.StatusForbidden}
	ErrPipelineNotFound                              = Error{ID: 12, Status: http.StatusBadRequest}
	ErrPipelineNotAttached                           = Error{ID: 13, Status: http.StatusBadRequest}
	ErrNoEnvironmentProvided                         = Error{ID: 14, Status: http.StatusBadRequest}
	ErrEnvironmentProvided                           = Error{ID: 15, Status: http.StatusBadRequest}
	ErrUnknownEnv                                    = Error{ID: 16, Status: http.StatusBadRequest}
	ErrEnvironmentExist                              = Error{ID: 17, Status: http.StatusConflict}
	ErrNoPipelineBuild                               = Error{ID: 18, Status: http.StatusNotFound}
	ErrApplicationNotFound                           = Error{ID: 19, Status: http.StatusNotFound}
	ErrInvalidUsername                               = Error{ID: 21, Status: http.StatusBadRequest}
	ErrInvalidEmail                                  = Error{ID: 22, Status: http.StatusBadRequest}
	ErrGroupPresent                                  = Error{ID: 23, Status: http.StatusBadRequest}
	ErrInvalidName                                   = Error{ID: 24, Status: http.StatusBadRequest}
	ErrInvalidUser                                   = Error{ID: 25, Status: http.StatusBadRequest}
	ErrBuildArchived                                 = Error{ID: 26, Status: http.StatusBadRequest}
	ErrNoEnvironment                                 = Error{ID: 27, Status: http.StatusNotFound}
	ErrModelNameExist                                = Error{ID: 28, Status: http.StatusConflict}
	ErrNoWorkerModel                                 = Error{ID: 29, Status: http.StatusNotFound}
	ErrNoProject                                     = Error{ID: 30, Status: http.StatusNotFound}
	ErrVariableExists                                = Error{ID: 31, Status: http.StatusConflict}
	ErrInvalidGroupPattern                           = Error{ID: 32, Status: http.StatusBadRequest}
	ErrGroupExists                                   = Error{ID: 33, Status: http.StatusConflict}
	ErrNotEnoughAdmin                                = Error{ID: 34, Status: http.StatusBadRequest}
	ErrInvalidProjectName                            = Error{ID: 35, Status: http.StatusBadRequest}
	ErrInvalidApplicationPattern                     = Error{ID: 36, Status: http.StatusBadRequest}
	ErrInvalidPipelinePattern                        = Error{ID: 37, Status: http.StatusBadRequest}
	ErrNotFound                                      = Error{ID: 38, Status: http.StatusNotFound}
	ErrNoWorkerModelCapa                             = Error{ID: 39, Status: http.StatusNotFound}
	ErrNoHook                                        = Error{ID: 40, Status: http.StatusNotFound}
	ErrNoAttachedPipeline                            = Error{ID: 41, Status: http.StatusNotFound}
	ErrNoReposManager                                = Error{ID: 42, Status: http.StatusNotFound}
	ErrNoReposManagerAuth                            = Error{ID: 43, Status: http.StatusUnauthorized}
	ErrNoReposManagerClientAuth                      = Error{ID: 44, Status: http.StatusForbidden}
	ErrRepoNotFound                                  = Error{ID: 45, Status: http.StatusNotFound}
	ErrSecretStoreUnreachable                        = Error{ID: 46, Status: http.StatusMethodNotAllowed}
	ErrSecretKeyFetchFailed                          = Error{ID: 47, Status: http.StatusMethodNotAllowed}
	ErrInvalidGoPath                                 = Error{ID: 48, Status: http.StatusBadRequest}
	ErrCommitsFetchFailed                            = Error{ID: 49, Status: http.StatusNotFound}
	ErrInvalidSecretFormat                           = Error{ID: 50, Status: http.StatusInternalServerError}
	ErrNoPreviousSuccess                             = Error{ID: 52, Status: http.StatusNotFound}
	ErrNoPermExecution                               = Error{ID: 53, Status: http.StatusForbidden}
	ErrSessionNotFound                               = Error{ID: 54, Status: http.StatusUnauthorized}
	ErrInvalidSecretValue                            = Error{ID: 55, Status: http.StatusBadRequest}
	ErrPipelineHasApplication                        = Error{ID: 56, Status: http.StatusBadRequest}
	ErrNoDirectSecretUse                             = Error{ID: 57, Status: http.StatusForbidden}
	ErrNoBranch                                      = Error{ID: 58, Status: http.StatusNotFound}
	ErrLDAPConn                                      = Error{ID: 59, Status: http.StatusInternalServerError}
	ErrServiceUnavailable                            = Error{ID: 60, Status: http.StatusServiceUnavailable}
	ErrParseUserNotification                         = Error{ID: 61, Status: http.StatusBadRequest}
	ErrNotSupportedUserNotification                  = Error{ID: 62, Status: http.StatusBadRequest}
	ErrGroupNeedAdmin                                = Error{ID: 63, Status: http.StatusBadRequest}
	ErrGroupNeedWrite                                = Error{ID: 64, Status: http.StatusBadRequest}
	ErrNoVariable                                    = Error{ID: 65, Status: http.StatusNotFound}
	ErrPluginInvalid                                 = Error{ID: 66, Status: http.StatusBadRequest}
	ErrConflict                                      = Error{ID: 67, Status: http.StatusConflict}
	ErrPipelineAlreadyAttached                       = Error{ID: 68, Status: http.StatusConflict}
	ErrApplicationExist                              = Error{ID: 69, Status: http.StatusConflict}
	ErrBranchNameNotProvided                         = Error{ID: 70, Status: http.StatusBadRequest}
	ErrInfiniteTriggerLoop                           = Error{ID: 71, Status: http.StatusBadRequest}
	ErrInvalidResetUser                              = Error{ID: 72, Status: http.StatusBadRequest}
	ErrUserConflict                                  = Error{ID: 73, Status: http.StatusBadRequest}
	ErrWrongRequest                                  = Error{ID: 74, Status: http.StatusBadRequest}
	ErrAlreadyExist                                  = Error{ID: 75, Status: http.StatusConflict}
	ErrInvalidType                                   = Error{ID: 76, Status: http.StatusBadRequest}
	ErrParentApplicationAndPipelineMandatory         = Error{ID: 77, Status: http.StatusBadRequest}
	ErrNoParentBuildFound                            = Error{ID: 78, Status: http.StatusNotFound}
	ErrParameterExists                               = Error{ID: 79, Status: http.StatusConflict}
	ErrNoHatchery                                    = Error{ID: 80, Status: http.StatusNotFound}
	ErrInvalidWorkerStatus                           = Error{ID: 81, Status: http.StatusNotFound}
	ErrInvalidToken                                  = Error{ID: 82, Status: http.StatusUnauthorized}
	ErrAppBuildingPipelines                          = Error{ID: 83, Status: http.StatusForbidden}
	ErrInvalidTimezone                               = Error{ID: 84, Status: http.StatusBadRequest}
	ErrEnvironmentCannotBeDeleted                    = Error{ID: 85, Status: http.StatusForbidden}
	ErrInvalidPipeline                               = Error{ID: 86, Status: http.StatusBadRequest}
	ErrKeyNotFound                                   = Error{ID: 87, Status: http.StatusNotFound}
	ErrPipelineAlreadyExists                         = Error{ID: 88, Status: http.StatusConflict}
	ErrJobAlreadyBooked                              = Error{ID: 89, Status: http.StatusConflict}
	ErrPipelineBuildNotFound                         = Error{ID: 90, Status: http.StatusNotFound}
	ErrAlreadyTaken                                  = Error{ID: 91, Status: http.StatusGone}
	ErrWorkflowNotFound                              = Error{ID: 92, Status: http.StatusNotFound}
	ErrWorkflowNodeNotFound                          = Error{ID: 93, Status: http.StatusNotFound}
	ErrWorkflowInvalidRoot                           = Error{ID: 94, Status: http.StatusBadRequest}
	ErrWorkflowNodeRef                               = Error{ID: 95, Status: http.StatusBadRequest}
	ErrWorkflowInvalid                               = Error{ID: 96, Status: http.StatusBadRequest}
	ErrWorkflowNodeJoinNotFound                      = Error{ID: 97, Status: http.StatusNotFound}
	ErrInvalidJobRequirement                         = Error{ID: 98, Status: http.StatusBadRequest}
	ErrNotImplemented                                = Error{ID: 99, Status: http.StatusNotImplemented}
	ErrParameterNotExists                            = Error{ID: 100, Status: http.StatusNotFound}
	ErrUnknownKeyType                                = Error{ID: 101, Status: http.StatusBadRequest}
	ErrInvalidKeyPattern                             = Error{ID: 102, Status: http.StatusBadRequest}
	ErrWebhookConfigDoesNotMatch                     = Error{ID: 103, Status: http.StatusBadRequest}
	ErrPipelineUsedByWorkflow                        = Error{ID: 104, Status: http.StatusBadRequest}
	ErrMethodNotAllowed                              = Error{ID: 105, Status: http.StatusMethodNotAllowed}
	ErrInvalidNodeNamePattern                        = Error{ID: 106, Status: http.StatusBadRequest}
	ErrWorkflowNodeParentNotRun                      = Error{ID: 107, Status: http.StatusForbidden}
	ErrHookNotFound                                  = Error{ID: 108, Status: http.StatusNotFound}
	ErrDefaultGroupPermission                        = Error{ID: 109, Status: http.StatusBadRequest}
	ErrLastGroupWithWriteRole                        = Error{ID: 110, Status: http.StatusForbidden}
	ErrInvalidEmailDomain                            = Error{ID: 111, Status: http.StatusForbidden}
	ErrWorkflowNodeRunJobNotFound                    = Error{ID: 112, Status: http.StatusNotFound}
	ErrBuiltinKeyNotFound                            = Error{ID: 113, Status: http.StatusInternalServerError}
	ErrStepNotFound                                  = Error{ID: 114, Status: http.StatusNotFound}
	ErrWorkerModelAlreadyBooked                      = Error{ID: 115, Status: http.StatusConflict}
	ErrConditionsNotOk                               = Error{ID: 116, Status: http.StatusBadRequest}
	ErrDownloadInvalidOS                             = Error{ID: 117, Status: http.StatusNotFound}
	ErrDownloadInvalidArch                           = Error{ID: 118, Status: http.StatusNotFound}
	ErrDownloadInvalidName                           = Error{ID: 119, Status: http.StatusNotFound}
	ErrDownloadDoesNotExist                          = Error{ID: 120, Status: http.StatusNotFound}
	ErrTokenNotFound                                 = Error{ID: 121, Status: http.StatusNotFound}
	ErrWorkflowNotificationNodeRef                   = Error{ID: 122, Status: http.StatusBadRequest}
	ErrInvalidJobRequirementDuplicateModel           = Error{ID: 123, Status: http.StatusBadRequest}
	ErrInvalidJobRequirementDuplicateHostname        = Error{ID: 124, Status: http.StatusBadRequest}
	ErrInvalidKeyName                                = Error{ID: 125, Status: http.StatusBadRequest}
	ErrRepoOperationTimeout                          = Error{ID: 126, Status: http.StatusRequestTimeout}
	ErrInvalidGitBranch                              = Error{ID: 127, Status: http.StatusBadRequest}
	ErrInvalidFavoriteType                           = Error{ID: 128, Status: http.StatusBadRequest}
	ErrUnsupportedOSArchPlugin                       = Error{ID: 129, Status: http.StatusNotFound}
	ErrNoBroadcast                                   = Error{ID: 130, Status: http.StatusNotFound}
	ErrBroadcastNotFound                             = Error{ID: 131, Status: http.StatusNotFound}
	ErrInvalidPatternModel                           = Error{ID: 132, Status: http.StatusBadRequest}
	ErrWorkerModelNoAdmin                            = Error{ID: 133, Status: http.StatusForbidden}
	ErrWorkerModelNoPattern                          = Error{ID: 134, Status: http.StatusForbidden}
	ErrJobNotBooked                                  = Error{ID: 135, Status: http.StatusBadRequest}
	ErrUserNotFound                                  = Error{ID: 136, Status: http.StatusNotFound}
	ErrInvalidNumber                                 = Error{ID: 137, Status: http.StatusBadRequest}
	ErrKeyAlreadyExist                               = Error{ID: 138, Status: http.StatusConflict}
	ErrPipelineNameImport                            = Error{ID: 139, Status: http.StatusBadRequest}
	ErrWorkflowNameImport                            = Error{ID: 140, Status: http.StatusBadRequest}
	ErrIconBadFormat                                 = Error{ID: 141, Status: http.StatusBadRequest}
	ErrIconBadSize                                   = Error{ID: 142, Status: http.StatusBadRequest}
	ErrWorkflowConditionBadOperator                  = Error{ID: 143, Status: http.StatusBadRequest}
	ErrColorBadFormat                                = Error{ID: 144, Status: http.StatusBadRequest}
	ErrInvalidHookConfiguration                      = Error{ID: 145, Status: http.StatusBadRequest}
	ErrWorkerModelDeploymentFailed                   = Error{ID: 146, Status: http.StatusBadRequest}
	ErrJobLocked                                     = Error{ID: 147, Status: http.StatusConflict}
	ErrWorkflowNodeRunLocked                         = Error{ID: 148, Status: http.StatusConflict}
	ErrInvalidData                                   = Error{ID: 149, Status: http.StatusBadRequest}
	ErrInvalidGroupAdmin                             = Error{ID: 150, Status: http.StatusForbidden}
	ErrInvalidGroupMember                            = Error{ID: 151, Status: http.StatusForbidden}
	ErrWorkflowNotGenerated                          = Error{ID: 152, Status: http.StatusForbidden}
	ErrAlreadyLatestTemplate                         = Error{ID: 153, Status: http.StatusForbidden}
	ErrInvalidNodeDefaultPayload                     = Error{ID: 154, Status: http.StatusBadRequest}
	ErrInvalidApplicationRepoStrategy                = Error{ID: 155, Status: http.StatusBadRequest}
	ErrWorkflowNodeRootUpdate                        = Error{ID: 156, Status: http.StatusBadRequest}
	ErrWorkflowAlreadyAsCode                         = Error{ID: 157, Status: http.StatusBadRequest}
	ErrNoDBMigrationID                               = Error{ID: 158, Status: http.StatusNotFound}
	ErrCannotParseTemplate                           = Error{ID: 159, Status: http.StatusBadRequest}
	ErrGroupNotFoundInProject                        = Error{ID: 160, Status: http.StatusBadRequest}
	ErrGroupNotFoundInWorkflow                       = Error{ID: 161, Status: http.StatusBadRequest}
	ErrWorkflowPermInsufficient                      = Error{ID: 162, Status: http.StatusBadRequest}
	ErrApplicationUsedByWorkflow                     = Error{ID: 163, Status: http.StatusBadRequest}
	ErrLocked                                        = Error{ID: 164, Status: http.StatusConflict}
	ErrInvalidJobRequirementWorkerModelPermission    = Error{ID: 165, Status: http.StatusBadRequest}
	ErrInvalidJobRequirementWorkerModelCapabilitites = Error{ID: 166, Status: http.StatusBadRequest}
	ErrMalformattedStep                              = Error{ID: 167, Status: http.StatusBadRequest}
	ErrVCSUsedByApplication                          = Error{ID: 168, Status: http.StatusBadRequest}
	ErrApplicationAsCodeOverride                     = Error{ID: 169, Status: http.StatusForbidden}
	ErrPipelineAsCodeOverride                        = Error{ID: 170, Status: http.StatusForbidden}
	ErrEnvironmentAsCodeOverride                     = Error{ID: 171, Status: http.StatusForbidden}
	ErrWorkflowAsCodeOverride                        = Error{ID: 172, Status: http.StatusForbidden}
	ErrProjectSecretDataUnknown                      = Error{ID: 173, Status: http.StatusBadRequest}
	ErrApplicationMandatoryOnWorkflowAsCode          = Error{ID: 174, Status: http.StatusBadRequest}
	ErrInvalidPassword                               = Error{ID: 175, Status: http.StatusBadRequest}
	ErrInvalidPayloadVariable                        = Error{ID: 176, Status: http.StatusBadRequest}
	ErrRepositoryUsedByHook                          = Error{ID: 177, Status: http.StatusForbidden}
	ErrResourceNotInProject                          = Error{ID: 178, Status: http.StatusForbidden}
	ErrEnvironmentNotFound                           = Error{ID: 179, Status: http.StatusBadRequest}
	ErrIntegrationtNotFound                          = Error{ID: 180, Status: http.StatusBadRequest}
	ErrSignupDisabled                                = Error{ID: 181, Status: http.StatusForbidden}
)

var errorsAmericanEnglish = map[int]string{
	ErrUnknownError.ID:                                  "internal server error",
	ErrActionAlreadyUpdated.ID:                          "action status already updated",
	ErrNoAction.ID:                                      "action does not exist",
	ErrActionLoop.ID:                                    "action definition contains a recursive loop",
	ErrInvalidID.ID:                                     "ID must be an integer",
	ErrInvalidProject.ID:                                "project not provided",
	ErrInvalidProjectKey.ID:                             "project key must contain only upper-case alphanumerical characters",
	ErrProjectHasPipeline.ID:                            "project contains a pipeline",
	ErrProjectHasApplication.ID:                         "project contains an application",
	ErrUnauthorized.ID:                                  "not authenticated",
	ErrForbidden.ID:                                     "forbidden",
	ErrPipelineNotFound.ID:                              "pipeline does not exist",
	ErrPipelineNotAttached.ID:                           "pipeline is not attached to application",
	ErrNoEnvironmentProvided.ID:                         "deployment and testing pipelines require an environnement",
	ErrEnvironmentProvided.ID:                           "build pipeline are not compatible with environment usage",
	ErrUnknownEnv.ID:                                    "unknown environment",
	ErrEnvironmentExist.ID:                              "environment already exists",
	ErrNoPipelineBuild.ID:                               "this pipeline build does not exist",
	ErrApplicationNotFound.ID:                           "application does not exist",
	ErrInvalidUsername.ID:                               "invalid username",
	ErrInvalidEmail.ID:                                  "invalid email",
	ErrGroupPresent.ID:                                  "group already present",
	ErrInvalidName.ID:                                   "invalid name",
	ErrInvalidUser.ID:                                   "invalid user or password",
	ErrBuildArchived.ID:                                 "Cannot restart this build because it has been archived",
	ErrNoEnvironment.ID:                                 "environment does not exist",
	ErrModelNameExist.ID:                                "worker model name already used",
	ErrNoWorkerModel.ID:                                 "worker model does not exist",
	ErrNoProject.ID:                                     "project does not exist",
	ErrVariableExists.ID:                                "variable already exists",
	ErrInvalidGroupPattern.ID:                           "group name must respect '^[a-zA-Z0-9.-_-]{1,}$'",
	ErrGroupExists.ID:                                   "group already exists",
	ErrNotEnoughAdmin.ID:                                "not enough group admin left",
	ErrInvalidProjectName.ID:                            "project name must not be empty",
	ErrInvalidApplicationPattern.ID:                     "application name must respect '^[a-zA-Z0-9.-_-]{1,}$'",
	ErrInvalidPipelinePattern.ID:                        "pipeline name must respect '^[a-zA-Z0-9.-_-]{1,}$'",
	ErrNotFound.ID:                                      "resource not found",
	ErrNoWorkerModelCapa.ID:                             "capability not found",
	ErrNoHook.ID:                                        "hook not found",
	ErrNoAttachedPipeline.ID:                            "pipeline not attached to the application",
	ErrNoReposManager.ID:                                "repositories manager not found",
	ErrNoReposManagerAuth.ID:                            "CDS authentication error, please contact CDS administrator",
	ErrNoReposManagerClientAuth.ID:                      "Repository manager authentication error, please unlink and relink your CDS Project to the repository manager",
	ErrRepoNotFound.ID:                                  "repository not found",
	ErrSecretStoreUnreachable.ID:                        "could not reach secret backend to fetch secret key",
	ErrSecretKeyFetchFailed.ID:                          "error while fetching key from secret backend",
	ErrCommitsFetchFailed.ID:                            "unable to retrieves commits",
	ErrInvalidGoPath.ID:                                 "invalid gopath",
	ErrInvalidSecretFormat.ID:                           "cannot decrypt secret, invalid format",
	ErrSessionNotFound.ID:                               "invalid session",
	ErrNoPreviousSuccess.ID:                             "there is no previous success version for this pipeline",
	ErrNoPermExecution.ID:                               "you don't have execution right",
	ErrInvalidSecretValue.ID:                            "secret value not specified",
	ErrPipelineHasApplication.ID:                        "pipeline still used by an application",
	ErrNoDirectSecretUse.ID:                             "usage of 'password' parameter is not allowed",
	ErrNoBranch.ID:                                      "branch not found in repository",
	ErrLDAPConn.ID:                                      "LDAP server connection error",
	ErrServiceUnavailable.ID:                            "service currently unavailable or down for maintenance",
	ErrParseUserNotification.ID:                         "unrecognized user notification settings",
	ErrNotSupportedUserNotification.ID:                  "unsupported user notification",
	ErrGroupNeedAdmin.ID:                                "need at least 1 administrator",
	ErrGroupNeedWrite.ID:                                "need at least 1 group with write permission",
	ErrNoVariable.ID:                                    "variable not found",
	ErrPluginInvalid.ID:                                 "invalid plugin",
	ErrConflict.ID:                                      "object conflict",
	ErrPipelineAlreadyAttached.ID:                       "pipeline already attached to this application",
	ErrApplicationExist.ID:                              "application already exists",
	ErrBranchNameNotProvided.ID:                         "git.branch or git.tag parameter must be provided",
	ErrInfiniteTriggerLoop.ID:                           "infinite trigger loop are forbidden",
	ErrInvalidResetUser.ID:                              "invalid user or email",
	ErrUserConflict.ID:                                  "this user already exists",
	ErrWrongRequest.ID:                                  "wrong request",
	ErrAlreadyExist.ID:                                  "already exists",
	ErrInvalidType.ID:                                   "invalid type",
	ErrParentApplicationAndPipelineMandatory.ID:         "parent application and pipeline are mandatory",
	ErrNoParentBuildFound.ID:                            "no parent build found",
	ErrParameterExists.ID:                               "parameter already exists",
	ErrNoHatchery.ID:                                    "No hatchery found",
	ErrInvalidWorkerStatus.ID:                           "Worker status is invalid",
	ErrInvalidToken.ID:                                  "Invalid token",
	ErrAppBuildingPipelines.ID:                          "Cannot delete application, there are building pipelines",
	ErrInvalidTimezone.ID:                               "Invalid timezone",
	ErrEnvironmentCannotBeDeleted.ID:                    "Environment cannot be deleted. It is still in used",
	ErrInvalidPipeline.ID:                               "Invalid pipeline",
	ErrKeyNotFound.ID:                                   "Key not found",
	ErrPipelineAlreadyExists.ID:                         "Pipeline already exists",
	ErrJobAlreadyBooked.ID:                              "Job already booked",
	ErrPipelineBuildNotFound.ID:                         "Pipeline build not found",
	ErrAlreadyTaken.ID:                                  "This job is already taken by another worker",
	ErrWorkflowNotFound.ID:                              "Workflow not found",
	ErrWorkflowNodeNotFound.ID:                          "Workflow node not found",
	ErrWorkflowInvalidRoot.ID:                           "Invalid workflow root",
	ErrWorkflowNodeRef.ID:                               "Invalid workflow node reference",
	ErrWorkflowInvalid.ID:                               "Invalid workflow",
	ErrWorkflowNodeJoinNotFound.ID:                      "Workflow node join not found",
	ErrInvalidJobRequirement.ID:                         "Invalid job requirement",
	ErrNotImplemented.ID:                                "This functionality isn't implemented",
	ErrParameterNotExists.ID:                            "This parameter doesn't exist",
	ErrUnknownKeyType.ID:                                "Unknown key type",
	ErrInvalidKeyPattern.ID:                             "key name must respect the following pattern: '^[a-zA-Z0-9.-_-]{1,}$'",
	ErrWebhookConfigDoesNotMatch.ID:                     "Webhook config does not match",
	ErrPipelineUsedByWorkflow.ID:                        "Pipeline still used by a workflow",
	ErrMethodNotAllowed.ID:                              "Method not allowed",
	ErrInvalidNodeNamePattern.ID:                        "Node name must respect the following pattern: '^[a-zA-Z0-9.-_-]{1,}$'",
	ErrWorkflowNodeParentNotRun.ID:                      "Cannot run a node if their parents have never been launched",
	ErrDefaultGroupPermission.ID:                        "Only read permission is allowed to default group",
	ErrLastGroupWithWriteRole.ID:                        "The last group must have the write permission",
	ErrInvalidEmailDomain.ID:                            "Invalid domain",
	ErrWorkflowNodeRunJobNotFound.ID:                    "Job not found",
	ErrBuiltinKeyNotFound.ID:                            "Encryption Key not found",
	ErrStepNotFound.ID:                                  "Step not found",
	ErrWorkerModelAlreadyBooked.ID:                      "Worker Model already booked",
	ErrConditionsNotOk.ID:                               "Cannot run this pipeline because launch conditions aren't ok",
	ErrDownloadInvalidOS.ID:                             "OS Invalid. Should be linux, darwin, freebsd or windows",
	ErrDownloadInvalidArch.ID:                           "Architecture invalid. Should be 386, i386, i686, amd64, x86_64 or arm (depends on OS)",
	ErrDownloadInvalidName.ID:                           "Invalid name",
	ErrDownloadDoesNotExist.ID:                          "File does not exist",
	ErrTokenNotFound.ID:                                 "Token does not exist",
	ErrWorkflowNotificationNodeRef.ID:                   "An invalid workflow node reference has been found, if you want to delete a pipeline from your workflow check if this pipeline isn't referenced in your notifications list",
	ErrInvalidJobRequirementDuplicateModel.ID:           "Invalid job requirements: you can't select multiple worker models",
	ErrInvalidJobRequirementDuplicateHostname.ID:        "Invalid job requirements: you can't select multiple hostname",
	ErrInvalidKeyName.ID:                                "Invalid key name. Application key must have prefix 'app-'; environment key must have prefix 'env-'",
	ErrRepoOperationTimeout.ID:                          "Analyzing repository took too much time",
	ErrInvalidGitBranch.ID:                              "Invalid git.branch value, you cannot have an empty git.branch value in your default payload",
	ErrInvalidFavoriteType.ID:                           "Invalid favorite type: must be 'project' or 'workflow'",
	ErrUnsupportedOSArchPlugin.ID:                       "Unsupported os/architecture for this plugin",
	ErrNoBroadcast.ID:                                   "Invalid broadcast",
	ErrBroadcastNotFound.ID:                             "Broadcast not found",
	ErrInvalidPatternModel.ID:                           "Invalid worker model pattern: name, type and main command are mandatory",
	ErrWorkerModelNoAdmin.ID:                            "Forbidden: you are neither a CDS administrator or the administrator for the group in which you want to create the worker model",
	ErrWorkerModelNoPattern.ID:                          "Forbidden: you must select a pattern of configuration scripts. If you have specific needs, please contact a CDS administrator",
	ErrJobNotBooked.ID:                                  "Job already released",
	ErrUserNotFound.ID:                                  "User not found",
	ErrInvalidNumber.ID:                                 "Invalid number",
	ErrKeyAlreadyExist.ID:                               "Key already exists",
	ErrPipelineNameImport.ID:                            "Pipeline name doesn't correspond in your code",
	ErrWorkflowNameImport.ID:                            "Workflow name doesn't correspond in your code",
	ErrIconBadFormat.ID:                                 "Bad icon format. Must be an image",
	ErrIconBadSize.ID:                                   "Bad icon size. Must be lower than 100Ko",
	ErrWorkflowConditionBadOperator.ID:                  "Your run conditions have bad operator",
	ErrColorBadFormat.ID:                                "The format of color isn't correct. You must use hexadecimal format (example: #FFFF)",
	ErrInvalidHookConfiguration.ID:                      "Invalid hook configuration",
	ErrInvalidData.ID:                                   "Cannot validate given data",
	ErrInvalidGroupAdmin.ID:                             "User is not a group's admin",
	ErrInvalidGroupMember.ID:                            "User is not a group's member",
	ErrWorkflowNotGenerated.ID:                          "Workflow was not generated by a template",
	ErrInvalidNodeDefaultPayload.ID:                     "Workflow node which isn't a root node cannot have a default payload",
	ErrInvalidApplicationRepoStrategy.ID:                "The repository strategy for your application is not correct",
	ErrWorkflowNodeRootUpdate.ID:                        "Unable to update/delete the root node of your workflow",
	ErrWorkflowAlreadyAsCode.ID:                         "Workflow is already as-code or there is already a pull-request to transform it",
	ErrNoDBMigrationID.ID:                               "ID does not exist in table gorp_migration",
	ErrCannotParseTemplate.ID:                           "Cannot parse workflow template",
	ErrGroupNotFoundInProject.ID:                        "Cannot add this permission group on your workflow because this group is not already in the project's permissions",
	ErrGroupNotFoundInWorkflow.ID:                       "Cannot add this permission group on your workflow node because this group is not already your workflow's permissions",
	ErrWorkflowPermInsufficient.ID:                      "Cannot add this permission group on your workflow because you can't have less rights than rights in your project when you are in RWX",
	ErrApplicationUsedByWorkflow.ID:                     "Application still used by a workflow",
	ErrLocked.ID:                                        "Resource locked",
	ErrInvalidJobRequirementWorkerModelPermission.ID:    "Invalid job requirements: unable to use worker model due to permissions",
	ErrInvalidJobRequirementWorkerModelCapabilitites.ID: "Invalid job requirements: the worker model doesn't match with the binary requirements",
	ErrMalformattedStep.ID:                              "Malformatted step",
	ErrVCSUsedByApplication.ID:                          "Repository manager still used by an application",
	ErrApplicationAsCodeOverride.ID:                     "You cannot override application from this repository",
	ErrPipelineAsCodeOverride.ID:                        "You cannot override pipeline from this repository",
	ErrEnvironmentAsCodeOverride.ID:                     "You cannot override environment from this repository",
	ErrWorkflowAsCodeOverride.ID:                        "You cannot override workflow from this repository",
	ErrProjectSecretDataUnknown.ID:                      "Invalid encrypted data",
	ErrApplicationMandatoryOnWorkflowAsCode.ID:          "An application linked to a git repository is mandatory on the workflow root",
	ErrInvalidPayloadVariable.ID:                        "Your payload cannot contain keys like cds.*",
	ErrInvalidPassword.ID:                               "Your value of type password isn't correct",
	ErrRepositoryUsedByHook.ID:                          "There is still a hook on this repository",
	ErrResourceNotInProject.ID:                          "The resource is not attached to the project",
	ErrEnvironmentNotFound.ID:                           "environment not found",
	ErrIntegrationtNotFound.ID:                          "integration not found",
	ErrSignupDisabled.ID:                                "Sign up is disabled for target consumer type",
}

var errorsFrench = map[int]string{
	ErrUnknownError.ID:                                  "erreur interne",
	ErrActionAlreadyUpdated.ID:                          "le status de l'action a déjà été mis à jour",
	ErrNoAction.ID:                                      "l'action n'existe pas",
	ErrActionLoop.ID:                                    "la définition de l'action contient une boucle récursive",
	ErrInvalidID.ID:                                     "l'ID doit être un nombre entier",
	ErrInvalidProject.ID:                                "projet manquant",
	ErrInvalidProjectKey.ID:                             "la clef de project doit uniquement contenir des lettres majuscules et des chiffres",
	ErrProjectHasPipeline.ID:                            "le project contient un pipeline",
	ErrProjectHasApplication.ID:                         "le project contient une application",
	ErrUnauthorized.ID:                                  "authentification invalide",
	ErrForbidden.ID:                                     "accès refusé",
	ErrPipelineNotFound.ID:                              "le pipeline n'existe pas",
	ErrPipelineNotAttached.ID:                           "le pipeline n'est pas lié à l'application",
	ErrNoEnvironmentProvided.ID:                         "les pipelines de déploiement et de tests requièrent un environnement",
	ErrEnvironmentProvided.ID:                           "une pipeline de build ne nécessite pas d'environnement",
	ErrUnknownEnv.ID:                                    "environnement inconnu",
	ErrEnvironmentExist.ID:                              "l'environnement existe",
	ErrNoPipelineBuild.ID:                               "ce build n'existe pas",
	ErrApplicationNotFound.ID:                           "l'application n'existe pas",
	ErrInvalidUsername.ID:                               "nom d'utilisateur invalide",
	ErrInvalidEmail.ID:                                  "addresse email invalide",
	ErrGroupPresent.ID:                                  "le groupe est déjà présent",
	ErrInvalidName.ID:                                   "le nom est invalide",
	ErrInvalidUser.ID:                                   "mauvaise combinaison compte/mot de passe utilisateur",
	ErrBuildArchived.ID:                                 "impossible de relancer ce build car il a été archivé",
	ErrNoEnvironment.ID:                                 "l'environement n'existe pas",
	ErrModelNameExist.ID:                                "le nom du modèle de worker est déjà utilisé",
	ErrNoWorkerModel.ID:                                 "le modèle de worker n'existe pas",
	ErrNoProject.ID:                                     "le projet n'existe pas",
	ErrVariableExists.ID:                                "la variable existe déjà",
	ErrInvalidGroupPattern.ID:                           "nom de groupe invalide '^[a-zA-Z0-9.-_-]{1,}$'",
	ErrGroupExists.ID:                                   "le groupe existe déjà",
	ErrNotEnoughAdmin.ID:                                "pas assez d'admin restant",
	ErrInvalidProjectName.ID:                            "nom de project vide non autorisé",
	ErrInvalidApplicationPattern.ID:                     "nom de l'application invalide '^[a-zA-Z0-9.-_-]{1,}$'",
	ErrInvalidPipelinePattern.ID:                        "nom du pipeline invalide '^[a-zA-Z0-9.-_-]{1,}$'",
	ErrNotFound.ID:                                      "la ressource n'existe pas",
	ErrNoWorkerModelCapa.ID:                             "la capacité n'existe pas",
	ErrNoHook.ID:                                        "le hook n'existe pas",
	ErrNoAttachedPipeline.ID:                            "le pipeline n'est pas lié à l'application",
	ErrNoReposManager.ID:                                "le gestionnaire de dépôt n'existe pas",
	ErrNoReposManagerAuth.ID:                            "connexion de CDS au gestionnaire de dépôt refusée, merci de contacter l'administrateur",
	ErrNoReposManagerClientAuth.ID:                      "connexion au gestionnaire de dépôts refusée, merci de détacher et ré-attacher le repository manager sur votre projet CDS",
	ErrRepoNotFound.ID:                                  "le dépôt n'existe pas",
	ErrSecretStoreUnreachable.ID:                        "impossible de contacter vault",
	ErrSecretKeyFetchFailed.ID:                          "erreur pendnat la récuperation de la clef de chiffrement",
	ErrCommitsFetchFailed.ID:                            "impossible de retrouver les changements",
	ErrInvalidGoPath.ID:                                 "le gopath n'est pas valide",
	ErrInvalidSecretFormat.ID:                           "impossibe de dechiffrer le secret, format invalide",
	ErrSessionNotFound.ID:                               "session invalide",
	ErrNoPreviousSuccess.ID:                             "il n'y a aucune précédente version en succès pour ce pipeline",
	ErrNoPermExecution.ID:                               "vous n'avez pas les droits d'éxécution",
	ErrInvalidSecretValue.ID:                            "valeur du secret non specifiée",
	ErrPipelineHasApplication.ID:                        "le pipeline est utilisé par une application",
	ErrNoDirectSecretUse.ID:                             "l'utilisation du type de paramêtre 'password' est impossible",
	ErrNoBranch.ID:                                      "la branche est introuvable dans le dépôt",
	ErrLDAPConn.ID:                                      "erreur de connexion au serveur LDAP",
	ErrServiceUnavailable.ID:                            "service temporairement indisponible ou en maintenance",
	ErrParseUserNotification.ID:                         "notification non reconnue",
	ErrNotSupportedUserNotification.ID:                  "notification non supportée",
	ErrGroupNeedAdmin.ID:                                "il faut au moins 1 administrateur",
	ErrGroupNeedWrite.ID:                                "il faut au moins 1 groupe avec les droits d'écriture",
	ErrNoVariable.ID:                                    "la variable n'existe pas",
	ErrPluginInvalid.ID:                                 "plugin non valide",
	ErrConflict.ID:                                      "l'objet est en conflit",
	ErrPipelineAlreadyAttached.ID:                       "le pipeline est déjà attaché à cette application",
	ErrApplicationExist.ID:                              "une application du même nom existe déjà",
	ErrBranchNameNotProvided.ID:                         "le paramètre git.branch ou git.tag est obligatoire",
	ErrInfiniteTriggerLoop.ID:                           "création d'une boucle de trigger infinie interdite",
	ErrInvalidResetUser.ID:                              "mauvaise combinaison compte/mail utilisateur",
	ErrUserConflict.ID:                                  "cet utilisateur existe deja",
	ErrWrongRequest.ID:                                  "la requête est incorrecte",
	ErrAlreadyExist.ID:                                  "conflit",
	ErrInvalidType.ID:                                   "type non valide",
	ErrParentApplicationAndPipelineMandatory.ID:         "application et pipeline parents obligatoires",
	ErrNoParentBuildFound.ID:                            "aucun build parent n'a pu être trouvé",
	ErrParameterExists.ID:                               "le paramètre existe déjà",
	ErrNoHatchery.ID:                                    "La hatchery n'existe pas",
	ErrInvalidWorkerStatus.ID:                           "Le status du worker est incorrect",
	ErrInvalidToken.ID:                                  "Token non valide",
	ErrAppBuildingPipelines.ID:                          "Impossible de supprimer l'application, il y a pipelines en cours",
	ErrInvalidTimezone.ID:                               "Fuseau horaire invalide",
	ErrEnvironmentCannotBeDeleted.ID:                    "L'environement ne peut etre supprimé. Il est encore utilisé.",
	ErrInvalidPipeline.ID:                               "Pipeline invalide",
	ErrKeyNotFound.ID:                                   "Clé introuvable",
	ErrPipelineAlreadyExists.ID:                         "Le pipeline existe déjà",
	ErrJobAlreadyBooked.ID:                              "Le job est déjà réservé",
	ErrPipelineBuildNotFound.ID:                         "Le pipeline build n'a pu être trouvé",
	ErrAlreadyTaken.ID:                                  "Ce job est déjà en cours de traitement par un autre worker",
	ErrWorkflowNotFound.ID:                              "Workflow introuvable",
	ErrWorkflowNodeNotFound.ID:                          "Noeud de Workflow introuvable",
	ErrWorkflowInvalidRoot.ID:                           "Racine de Workflow invalide",
	ErrWorkflowNodeRef.ID:                               "Référence de noeud de workflow invalide",
	ErrWorkflowInvalid.ID:                               "Workflow invalide",
	ErrWorkflowNodeJoinNotFound.ID:                      "Jointure introuvable",
	ErrInvalidJobRequirement.ID:                         "Pré-requis de Job invalide",
	ErrNotImplemented.ID:                                "La fonctionnalité n'est pas implémentée",
	ErrParameterNotExists.ID:                            "Ce paramètre n'existe pas",
	ErrUnknownKeyType.ID:                                "Le type de clé n'est pas connu",
	ErrInvalidKeyPattern.ID:                             "le nom de la clé doit respecter le pattern suivant; '^[a-zA-Z0-9.-_-]{1,}$'",
	ErrWebhookConfigDoesNotMatch.ID:                     "la configuration du webhook ne correspond pas",
	ErrPipelineUsedByWorkflow.ID:                        "Le pipeline est utilisé par un workflow",
	ErrMethodNotAllowed.ID:                              "La méthode n'est pas autorisée",
	ErrInvalidNodeNamePattern.ID:                        "Le nom du noeud du workflow doit respecter le pattern suivant; '^[a-zA-Z0-9.-_-]{1,}$'",
	ErrWorkflowNodeParentNotRun.ID:                      "Il est interdit de lancer un noeuds si ses parents n'ont jamais été lancés",
	ErrDefaultGroupPermission.ID:                        "Le groupe par défaut ne peut être utilisé qu'en lecture seule",
	ErrLastGroupWithWriteRole.ID:                        "Le dernier groupe doit avoir les droits d'écriture",
	ErrInvalidEmailDomain.ID:                            "Domaine invalide",
	ErrWorkflowNodeRunJobNotFound.ID:                    "Job non trouvé",
	ErrBuiltinKeyNotFound.ID:                            "Clé de chiffrage introuvable",
	ErrStepNotFound.ID:                                  "Step introuvable",
	ErrWorkerModelAlreadyBooked.ID:                      "Le modèle de worker est déjà réservé",
	ErrConditionsNotOk.ID:                               "Impossible de démarrer ce pipeline car les conditions de lancement ne sont pas respectées",
	ErrDownloadInvalidOS.ID:                             "OS invalide. L'OS doit être linux, darwin, freebsd ou windows",
	ErrDownloadInvalidArch.ID:                           "Architecture invalide. L'architecture doit être 386, i386, i686, amd64, x86_64 ou arm (dépendant de l'OS)",
	ErrDownloadInvalidName.ID:                           "Nom invalide",
	ErrDownloadDoesNotExist.ID:                          "Le fichier n'existe pas",
	ErrTokenNotFound.ID:                                 "Le token n'existe pas",
	ErrWorkflowNotificationNodeRef.ID:                   "Une référence de noeud de workflow est invalide dans vos notifications (si vous souhaitez supprimer un pipeline vérifiez qu'il ne soit plus référencé dans la liste de vos notifications)",
	ErrInvalidJobRequirementDuplicateModel.ID:           "Pré-requis de job invalides: vous ne pouvez pas séléctionnez plusieurs modèles de worker",
	ErrInvalidJobRequirementDuplicateHostname.ID:        "Pré-requis de job invalides: vous ne pouvez pas séléctionnez plusieurs hostname",
	ErrInvalidKeyName.ID:                                "Nom de clé invalide. Les clés d'application doivent être préfixées par 'app-', les clés d'environnement doivent être préfixées par 'env-'",
	ErrRepoOperationTimeout.ID:                          "L'analyse du dépôt a pris trop de temps",
	ErrInvalidGitBranch.ID:                              "Valeur git.branch invalide, vous ne pouvez pas avoir de valeur git.branch avec une string vide dans votre payload par défaut",
	ErrInvalidFavoriteType.ID:                           "Type de favori invalide: doit être 'projet' ou 'workflow'",
	ErrUnsupportedOSArchPlugin.ID:                       "OS/Architecture non supporté pour ce plugin",
	ErrNoBroadcast.ID:                                   "Information invalide",
	ErrBroadcastNotFound.ID:                             "Information non trouvée",
	ErrInvalidPatternModel.ID:                           "Pattern de modèle de worker invalide: le nom, type et commande principale sont requis",
	ErrWorkerModelNoAdmin.ID:                            "Accès refusé: vous n'êtes ni un administrateur CDS ni un administrateur du groupe pour lequel vous tentez de créer votre modèle",
	ErrWorkerModelNoPattern.ID:                          "Accès refusé: vous devez obligatoirement sélectionner un pattern de script de configuration. Si vous souhaitez ajouter un pattern particulier, veuillez contacter un administrateur CDS",
	ErrJobNotBooked.ID:                                  "Le job est déjà libéré",
	ErrUserNotFound.ID:                                  "Utilisateur non trouvé",
	ErrInvalidNumber.ID:                                 "Nombre non valide",
	ErrKeyAlreadyExist.ID:                               "La clé existe déjà",
	ErrPipelineNameImport.ID:                            "Le nom du pipeline dans le code ne correspond pas au nom du pipeline que vous voulez éditer",
	ErrWorkflowNameImport.ID:                            "Le nom du workflow dans le code ne correspond pas au nom du workflow que vous voulez éditer",
	ErrIconBadFormat.ID:                                 "Mauvais format d'icône, doit être une image",
	ErrIconBadSize.ID:                                   "Taille de l'icône trop importante. (max 100Ko)",
	ErrWorkflowConditionBadOperator.ID:                  "Opérateur de condition de lancement incorrect",
	ErrColorBadFormat.ID:                                "Format de la couleur incorrect. Vous devez utiliser le format hexadécimal (exemple: #FFFF)",
	ErrInvalidHookConfiguration.ID:                      "Configuration de hook invalide",
	ErrInvalidData.ID:                                   "Impossible de valider les données",
	ErrInvalidGroupAdmin.ID:                             "L'utilisateur n'est pas administrateur du groupe",
	ErrInvalidGroupMember.ID:                            "L'utilisateur n'est pas membre du groupe",
	ErrWorkflowNotGenerated.ID:                          "Le workflow n'a pas été généré par un template",
	ErrInvalidNodeDefaultPayload.ID:                     "Le workflow est incorrect. Un payload par défaut ne peut pas être sur un pipeline autre que le premier du workflow",
	ErrInvalidApplicationRepoStrategy.ID:                "La stratégie de dépôt (vcs) de l'application n'est pas correcte",
	ErrWorkflowNodeRootUpdate.ID:                        "Impossible de mettre à jour ou supprimer le noeud racine du workflow",
	ErrWorkflowAlreadyAsCode.ID:                         "Le workflow est déjà as-code ou il y a déjà une pull-request pour le transformer",
	ErrNoDBMigrationID.ID:                               "Cet id n'existe pas dans la table gorp_migrations",
	ErrCannotParseTemplate.ID:                           "Impossible de parser le modèle de workflow",
	ErrGroupNotFoundInProject.ID:                        "Impossible d'ajouter ce groupe dans vos permissions de workflow car ce groupe n'est pas présent dans les permissions de votre projet",
	ErrGroupNotFoundInWorkflow.ID:                       "Impossible d'ajouter ce groupe dans vos permissions de noeud du workflow car ce groupe n'est pas présent dans les permissions de votre workflow",
	ErrWorkflowPermInsufficient.ID:                      "Impossible d'ajouter ce groupe dans vos permissions du workflow car ce groupe a des droits inférieurs (< RWX) à celui du workflow",
	ErrApplicationUsedByWorkflow.ID:                     "L'application est utilisée par un workflow",
	ErrLocked.ID:                                        "La ressource est verrouillée",
	ErrInvalidJobRequirementWorkerModelPermission.ID:    "Pré-requis de job invalide: Modèle de worker inutilisable en raison des permissions",
	ErrInvalidJobRequirementWorkerModelCapabilitites.ID: "Pré-requis de job invalide: Le modèle de worker ne dispose pas de binaires suffisants",
	ErrMalformattedStep.ID:                              "Étape malformée",
	ErrVCSUsedByApplication.ID:                          "Le gestionnaire de dépot est encore utilisé par une application",
	ErrApplicationAsCodeOverride.ID:                     "Vous ne pouvez pas importer l'application depuis ce dépôt",
	ErrPipelineAsCodeOverride.ID:                        "Vous ne pouvez pas importer le pipeline depuis ce dépôt",
	ErrEnvironmentAsCodeOverride.ID:                     "Vous ne pouvez pas importer l'environment depuis ce dépôt",
	ErrWorkflowAsCodeOverride.ID:                        "Vous ne pouvez pas importer le workflow depuis ce dépôt",
	ErrProjectSecretDataUnknown.ID:                      "Donnée chiffrée non valide",
	ErrApplicationMandatoryOnWorkflowAsCode.ID:          "Une application liée à un dépôt git est obligatoire à la racine du workflow",
	ErrInvalidPayloadVariable.ID:                        "Le payload du workflow ne peut pas contenir de clés nommées cds.*",
	ErrInvalidPassword.ID:                               "Votre valeur de type mot de passe n'est pas correct",
	ErrRepositoryUsedByHook.ID:                          "Il y a encore un hook sur ce dépôt",
	ErrResourceNotInProject.ID:                          "La ressource n'est pas lié au projet",
	ErrEnvironmentNotFound.ID:                           "l'environnement n'existe pas",
	ErrSignupDisabled.ID:                                "La création de compte est désactivée pour ce mode d'authentification.",
}

var errorsLanguages = []map[int]string{
	errorsAmericanEnglish,
	errorsFrench,
}

// Error type.
type Error struct {
	ID         int         `json:"id"`
	Status     int         `json:"-"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data"`
	UUID       string      `json:"uuid,omitempty"`
	StackTrace string      `json:"stack_trace,omitempty"`
	from       string
}

func (e Error) Error() string {
	if e.Message != "" {
		return e.Message
	}

	if en, ok := errorsAmericanEnglish[e.ID]; ok {
		return en
	}

	return errorsAmericanEnglish[ErrUnknownError.ID]
}

func (e Error) Translate(al string) string {
	acceptedLanguages, _, err := language.ParseAcceptLanguage(al)
	if err != nil {
		acceptedLanguages = []language.Tag{language.AmericanEnglish}
	}

	// try to get error message for accepted language and error ID, else use unknown error message
	tag, _, _ := matcher.Match(acceptedLanguages...)
	var msg string
	var ok bool
	switch tag {
	case language.French:
		msg, ok = errorsFrench[e.ID]
		break
	case language.AmericanEnglish:
		msg, ok = errorsAmericanEnglish[e.ID]
		break
	default:
		msg, ok = errorsAmericanEnglish[e.ID]
		break
	}
	if !ok {
		return errorsAmericanEnglish[ErrUnknownError.ID]
	}

	return msg
}

// NewErrorWithStack returns an error with stack.
func NewErrorWithStack(root error, err error) error {
	errWithStack := WithStack(root).(errorWithStack)
	errWithStack.httpError = ExtractHTTPError(err, "")
	return errWithStack
}

type errorWithStack struct {
	root      error  // root error should be wrapped with stack
	stack     *stack // used to generate inline call stack
	httpError Error
}

func (w errorWithStack) Error() string {
	var cause string
	if w.root.Error() != "" {
		cause = fmt.Sprintf(" (caused by: %s)", w.root)
	}
	return fmt.Sprintf("%s: %s%s", w.stack.String(), w.httpError, cause)
}

func (w errorWithStack) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			// print the root stack trace
			fmt.Fprintf(s, "%+v", w.root)
			return
		}
		fallthrough
	default:
		_, _ = io.WriteString(s, w.Error())
	}
}

// ErrorWithFallback returns the current error if it's not a ErrUnknownError else it returns a new ErrorWithStack
func ErrorWithFallback(err error, httpError Error, from string, args ...interface{}) error {
	if ErrorIs(err, ErrUnknownError) {
		return NewErrorWithStack(err, NewError(ErrWrongRequest, fmt.Errorf(from, args...)))
	}
	return err
}

// IsErrorWithStack returns true if given error is an errorWithStack.
func IsErrorWithStack(err error) bool {
	_, ok := err.(errorWithStack)
	return ok
}

type stack []uintptr

func (s *stack) String() string {
	var names []string
	for _, pc := range *s {
		name := runtime.FuncForPC(pc).Name()
		if strings.HasPrefix(name, "github.com/ovh/cds/vendor") {
			continue
		}
		if strings.HasPrefix(name, "github.com/ovh/cds") {
			sp := strings.Split(name, "/")
			sp = strings.Split(sp[len(sp)-1], ".")
			// check if it's a struct or package func
			if strings.HasPrefix(sp[1], "(") {
				names = append(names, sp[2])
			} else {
				names = append(names, sp[1])
			}
		}
	}
	reverse(names)
	return strings.Join(names, ">")
}

func reverse(ss []string) {
	last := len(ss) - 1
	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}
}

func callers() *stack {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	var st stack = pcs[0:n]
	return &st
}

// NewError returns a merge of given err with new http error.
func NewError(httpError Error, err error) error {
	// if the given error is nil do nothing
	if err == nil {
		return nil
	}

	// if it's already an error with stack, override the http error and set from value with err cause
	if e, ok := err.(errorWithStack); ok {
		httpError.from = Cause(e).Error()
		e.httpError = httpError
		return e
	}

	// if it's a library error create a new error with stack
	httpError.from = err.Error()
	return errorWithStack{
		root:      errors.WithStack(err),
		stack:     callers(),
		httpError: httpError,
	}
}

// NewErrorFrom returns the given http error with from details.
func NewErrorFrom(httpError Error, from string, args ...interface{}) error {
	return NewError(httpError, fmt.Errorf(from, args...))
}

// WrapError returns an error with stack and message.
func WrapError(err error, format string, args ...interface{}) error {
	// the wrap should ignore nil err like in pkg/errors lib
	if err == nil {
		return nil
	}

	m := fmt.Sprintf(format, args...)

	// if it's already a CDS error then only wrap with message
	if e, ok := err.(errorWithStack); ok {
		e.root = errors.WithMessage(e.root, m)
		return e
	}

	// if it's a Error wrap it in error with stack
	if e, ok := err.(Error); ok {
		return errorWithStack{
			root:      errors.WithStack(fmt.Errorf(format, args...)),
			stack:     callers(),
			httpError: e,
		}
	}

	return errorWithStack{
		root:      errors.Wrap(err, m),
		stack:     callers(),
		httpError: ErrUnknownError,
	}
}

// WithStack returns an error with stack from given error.
func WithStack(err error) error {
	// the wrap should ignore nil err like in pkg/errors lib
	if err == nil {
		return nil
	}

	// if it's already a CDS error do not override the stack
	if e, ok := err.(errorWithStack); ok {
		return e
	}

	// if it's a Error wrap it in error with stack
	if e, ok := err.(Error); ok {
		return errorWithStack{
			root:      errors.New(e.Error()),
			stack:     callers(),
			httpError: e,
		}
	}

	return errorWithStack{
		root:      errors.WithStack(err),
		stack:     callers(),
		httpError: ErrUnknownError,
	}
}

// ErrorWithData returns an error with data from given error.
func ErrorWithData(err Error, data interface{}) error {
	err.Data = data
	return err
}

// ExtractHTTPError tries to recognize given error and return http error
// with message in a language matching Accepted-Language.
func ExtractHTTPError(source error, al string) Error {
	var httpError Error

	// try to recognize http error from source
	switch e := source.(type) {
	case errorWithStack:
		httpError = e.httpError
	case Error:
		httpError = e
	default:
		httpError = ErrUnknownError
	}

	// if it's a custom err with no status use unknown error status
	if httpError.Status == 0 {
		httpError.Status = ErrUnknownError.Status
	}

	// if error's message is not empty do not override (custom message)
	// else set message for given accepted languages.
	if httpError.Message == "" {
		httpError.Message = httpError.Translate(al)
	}

	if httpError.from != "" {
		httpError.Message = fmt.Sprintf("%s (from: %s)", httpError.Message, httpError.from)
	}

	return httpError
}

// Exit func display an error message on stderr and exit 1
func Exit(format string, args ...interface{}) {
	if len(format) > 0 && format[:len(format)-1] != "\n" {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

// DecodeError return an Error struct from json
func DecodeError(data []byte) error {
	var e Error

	if err := json.Unmarshal(data, &e); err != nil {
		return nil
	}

	if e.Message == "" {
		return nil
	}

	return e
}

// ErrorIs returns true if error match the target error.
func ErrorIs(err error, target Error) bool {
	if err == nil {
		return false
	}

	switch e := err.(type) {
	case errorWithStack:
		return e.httpError.ID == target.ID
	case Error:
		return e.ID == target.ID
	default:
		return target.ID == ErrUnknownError.ID
	}
}

// Cause returns recursively the root error from given error.
func Cause(err error) error {
	if e, ok := err.(errorWithStack); ok {
		return errors.Cause(e.root)
	}
	return errors.Cause(err)
}

// ErrorIsUnknown returns true the error is unknown (sdk.ErrUnknownError or lib error).
func ErrorIsUnknown(err error) bool {
	return ErrorIs(err, ErrUnknownError)
}

// MultiError is just an array of error
type MultiError []error

func (e *MultiError) Error() string {
	var s string
	for i := range *e {
		if i > 0 {
			s += ", "
		}
		s += (*e)[i].Error()
	}
	return s
}

// Join joins errors from MultiError to another errors MultiError
func (e *MultiError) Join(j MultiError) {
	for _, err := range j {
		*e = append(*e, err)
	}
}

// Append appends an error to a MultiError
func (e *MultiError) Append(err error) {
	if err == nil {
		return
	}
	if mError, ok := err.(*MultiError); ok {
		for i := range *mError {
			e.Append((*mError)[i])
		}
	} else {
		*e = append(*e, err)
	}
}

// IsEmpty return true if MultiError is empty, false otherwise
func (e *MultiError) IsEmpty() bool { return len(*e) == 0 }
