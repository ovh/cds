package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/pkg/errors"

	"golang.org/x/text/language"
)

// Existing CDS errors
// Note: the error id is useless except to ensure objects are different in map
var (
	ErrUnknownError                          = Error{ID: 1, Status: http.StatusInternalServerError}
	ErrActionAlreadyUpdated                  = Error{ID: 2, Status: http.StatusBadRequest}
	ErrNoAction                              = Error{ID: 3, Status: http.StatusNotFound}
	ErrActionLoop                            = Error{ID: 4, Status: http.StatusBadRequest}
	ErrInvalidID                             = Error{ID: 5, Status: http.StatusBadRequest}
	ErrInvalidProject                        = Error{ID: 6, Status: http.StatusBadRequest}
	ErrInvalidProjectKey                     = Error{ID: 7, Status: http.StatusBadRequest}
	ErrProjectHasPipeline                    = Error{ID: 8, Status: http.StatusConflict}
	ErrProjectHasApplication                 = Error{ID: 9, Status: http.StatusConflict}
	ErrUnauthorized                          = Error{ID: 10, Status: http.StatusUnauthorized}
	ErrForbidden                             = Error{ID: 11, Status: http.StatusForbidden}
	ErrPipelineNotFound                      = Error{ID: 12, Status: http.StatusBadRequest}
	ErrPipelineNotAttached                   = Error{ID: 13, Status: http.StatusBadRequest}
	ErrNoEnvironmentProvided                 = Error{ID: 14, Status: http.StatusBadRequest}
	ErrEnvironmentProvided                   = Error{ID: 15, Status: http.StatusBadRequest}
	ErrUnknownEnv                            = Error{ID: 16, Status: http.StatusBadRequest}
	ErrEnvironmentExist                      = Error{ID: 17, Status: http.StatusConflict}
	ErrNoPipelineBuild                       = Error{ID: 18, Status: http.StatusNotFound}
	ErrApplicationNotFound                   = Error{ID: 19, Status: http.StatusNotFound}
	ErrGroupNotFound                         = Error{ID: 20, Status: http.StatusNotFound}
	ErrInvalidUsername                       = Error{ID: 21, Status: http.StatusBadRequest}
	ErrInvalidEmail                          = Error{ID: 22, Status: http.StatusBadRequest}
	ErrGroupPresent                          = Error{ID: 23, Status: http.StatusBadRequest}
	ErrInvalidName                           = Error{ID: 24, Status: http.StatusBadRequest}
	ErrInvalidUser                           = Error{ID: 25, Status: http.StatusBadRequest}
	ErrBuildArchived                         = Error{ID: 26, Status: http.StatusBadRequest}
	ErrNoEnvironment                         = Error{ID: 27, Status: http.StatusNotFound}
	ErrModelNameExist                        = Error{ID: 28, Status: http.StatusConflict}
	ErrNoWorkerModel                         = Error{ID: 29, Status: http.StatusNotFound}
	ErrNoProject                             = Error{ID: 30, Status: http.StatusNotFound}
	ErrVariableExists                        = Error{ID: 31, Status: http.StatusConflict}
	ErrInvalidGroupPattern                   = Error{ID: 32, Status: http.StatusBadRequest}
	ErrGroupExists                           = Error{ID: 33, Status: http.StatusConflict}
	ErrNotEnoughAdmin                        = Error{ID: 34, Status: http.StatusBadRequest}
	ErrInvalidProjectName                    = Error{ID: 35, Status: http.StatusBadRequest}
	ErrInvalidApplicationPattern             = Error{ID: 36, Status: http.StatusBadRequest}
	ErrInvalidPipelinePattern                = Error{ID: 37, Status: http.StatusBadRequest}
	ErrNotFound                              = Error{ID: 38, Status: http.StatusNotFound}
	ErrNoWorkerModelCapa                     = Error{ID: 39, Status: http.StatusNotFound}
	ErrNoHook                                = Error{ID: 40, Status: http.StatusNotFound}
	ErrNoAttachedPipeline                    = Error{ID: 41, Status: http.StatusNotFound}
	ErrNoReposManager                        = Error{ID: 42, Status: http.StatusNotFound}
	ErrNoReposManagerAuth                    = Error{ID: 43, Status: http.StatusUnauthorized}
	ErrNoReposManagerClientAuth              = Error{ID: 44, Status: http.StatusForbidden}
	ErrRepoNotFound                          = Error{ID: 45, Status: http.StatusNotFound}
	ErrSecretStoreUnreachable                = Error{ID: 46, Status: http.StatusMethodNotAllowed}
	ErrSecretKeyFetchFailed                  = Error{ID: 47, Status: http.StatusMethodNotAllowed}
	ErrInvalidGoPath                         = Error{ID: 48, Status: http.StatusBadRequest}
	ErrCommitsFetchFailed                    = Error{ID: 49, Status: http.StatusNotFound}
	ErrInvalidSecretFormat                   = Error{ID: 50, Status: http.StatusInternalServerError}
	ErrNoPreviousSuccess                     = Error{ID: 52, Status: http.StatusNotFound}
	ErrNoEnvExecution                        = Error{ID: 53, Status: http.StatusForbidden}
	ErrSessionNotFound                       = Error{ID: 54, Status: http.StatusUnauthorized}
	ErrInvalidSecretValue                    = Error{ID: 55, Status: http.StatusBadRequest}
	ErrPipelineHasApplication                = Error{ID: 56, Status: http.StatusBadRequest}
	ErrNoDirectSecretUse                     = Error{ID: 57, Status: http.StatusForbidden}
	ErrNoBranch                              = Error{ID: 58, Status: http.StatusNotFound}
	ErrLDAPConn                              = Error{ID: 59, Status: http.StatusInternalServerError}
	ErrServiceUnavailable                    = Error{ID: 60, Status: http.StatusServiceUnavailable}
	ErrParseUserNotification                 = Error{ID: 61, Status: http.StatusBadRequest}
	ErrNotSupportedUserNotification          = Error{ID: 62, Status: http.StatusBadRequest}
	ErrGroupNeedAdmin                        = Error{ID: 63, Status: http.StatusBadRequest}
	ErrGroupNeedWrite                        = Error{ID: 64, Status: http.StatusBadRequest}
	ErrNoVariable                            = Error{ID: 65, Status: http.StatusNotFound}
	ErrPluginInvalid                         = Error{ID: 66, Status: http.StatusBadRequest}
	ErrConflict                              = Error{ID: 67, Status: http.StatusConflict}
	ErrPipelineAlreadyAttached               = Error{ID: 68, Status: http.StatusConflict}
	ErrApplicationExist                      = Error{ID: 69, Status: http.StatusConflict}
	ErrBranchNameNotProvided                 = Error{ID: 70, Status: http.StatusBadRequest}
	ErrInfiniteTriggerLoop                   = Error{ID: 71, Status: http.StatusBadRequest}
	ErrInvalidResetUser                      = Error{ID: 72, Status: http.StatusBadRequest}
	ErrUserConflict                          = Error{ID: 73, Status: http.StatusBadRequest}
	ErrWrongRequest                          = Error{ID: 74, Status: http.StatusBadRequest}
	ErrAlreadyExist                          = Error{ID: 75, Status: http.StatusConflict}
	ErrInvalidType                           = Error{ID: 76, Status: http.StatusBadRequest}
	ErrParentApplicationAndPipelineMandatory = Error{ID: 77, Status: http.StatusBadRequest}
	ErrNoParentBuildFound                    = Error{ID: 78, Status: http.StatusNotFound}
	ErrParameterExists                       = Error{ID: 79, Status: http.StatusConflict}
	ErrNoHatchery                            = Error{ID: 80, Status: http.StatusNotFound}
	ErrInvalidWorkerStatus                   = Error{ID: 81, Status: http.StatusNotFound}
	ErrInvalidToken                          = Error{ID: 82, Status: http.StatusUnauthorized}
	ErrAppBuildingPipelines                  = Error{ID: 83, Status: http.StatusForbidden}
	ErrInvalidTimezone                       = Error{ID: 84, Status: http.StatusBadRequest}
	ErrEnvironmentCannotBeDeleted            = Error{ID: 85, Status: http.StatusForbidden}
	ErrInvalidPipeline                       = Error{ID: 86, Status: http.StatusBadRequest}
	ErrKeyNotFound                           = Error{ID: 87, Status: http.StatusNotFound}
	ErrPipelineAlreadyExists                 = Error{ID: 88, Status: http.StatusConflict}
	ErrJobAlreadyBooked                      = Error{ID: 89, Status: http.StatusConflict}
	ErrPipelineBuildNotFound                 = Error{ID: 90, Status: http.StatusNotFound}
	ErrAlreadyTaken                          = Error{ID: 91, Status: http.StatusGone}
	ErrWorkflowNotFound                      = Error{ID: 92, Status: http.StatusNotFound}
	ErrWorkflowNodeNotFound                  = Error{ID: 93, Status: http.StatusNotFound}
	ErrWorkflowInvalidRoot                   = Error{ID: 94, Status: http.StatusBadRequest}
	ErrWorkflowNodeRef                       = Error{ID: 95, Status: http.StatusBadRequest}
	ErrWorkflowInvalid                       = Error{ID: 96, Status: http.StatusBadRequest}
	ErrWorkflowNodeJoinNotFound              = Error{ID: 97, Status: http.StatusNotFound}
	ErrInvalidJobRequirement                 = Error{ID: 98, Status: http.StatusBadRequest}
	ErrNotImplemented                        = Error{ID: 99, Status: http.StatusNotImplemented}
	ErrParameterNotExists                    = Error{ID: 100, Status: http.StatusNotFound}
	ErrUnknownKeyType                        = Error{ID: 101, Status: http.StatusBadRequest}
	ErrInvalidKeyPattern                     = Error{ID: 102, Status: http.StatusBadRequest}
	ErrWebhookConfigDoesNotMatch             = Error{ID: 103, Status: http.StatusBadRequest}
	ErrPipelineUsedByWorkflow                = Error{ID: 104, Status: http.StatusBadRequest}
	ErrMethodNotAllowed                      = Error{ID: 105, Status: http.StatusMethodNotAllowed}
	ErrInvalidNodeNamePattern                = Error{ID: 106, Status: http.StatusBadRequest}
	ErrWorkflowNodeParentNotRun              = Error{ID: 107, Status: http.StatusForbidden}
	ErrHookNotFound                          = Error{ID: 108, Status: http.StatusNotFound}
	ErrDefaultGroupPermission                = Error{ID: 109, Status: http.StatusBadRequest}
	ErrLastGroupWithWriteRole                = Error{ID: 110, Status: http.StatusForbidden}
	ErrInvalidEmailDomain                    = Error{ID: 111, Status: http.StatusForbidden}
	ErrWorkflowNodeRunJobNotFound            = Error{ID: 112, Status: http.StatusNotFound}
	ErrBuiltinKeyNotFound                    = Error{ID: 113, Status: http.StatusInternalServerError}
	ErrStepNotFound                          = Error{ID: 114, Status: http.StatusNotFound}
	ErrWorkerModelAlreadyBooked              = Error{ID: 115, Status: http.StatusConflict}
	ErrConditionsNotOk                       = Error{ID: 116, Status: http.StatusBadRequest}
	ErrDownloadInvalidOS                     = Error{ID: 117, Status: http.StatusNotFound}
	ErrDownloadInvalidArch                   = Error{ID: 118, Status: http.StatusNotFound}
	ErrDownloadInvalidName                   = Error{ID: 119, Status: http.StatusNotFound}
	ErrDownloadDoesNotExist                  = Error{ID: 120, Status: http.StatusNotFound}
	ErrTokenNotFound                         = Error{ID: 121, Status: http.StatusNotFound}
	ErrWorkflowNotificationNodeRef           = Error{ID: 122, Status: http.StatusBadRequest}
	ErrInvalidJobRequirementDuplicateModel   = Error{ID: 123, Status: http.StatusBadRequest}
)

var errorsAmericanEnglish = map[int]string{
	ErrUnknownError.ID:                          "internal server error",
	ErrActionAlreadyUpdated.ID:                  "action status already updated",
	ErrNoAction.ID:                              "action does not exist",
	ErrActionLoop.ID:                            "action definition contains a recursive loop",
	ErrInvalidID.ID:                             "ID must be an integer",
	ErrInvalidProject.ID:                        "project not provided",
	ErrInvalidProjectKey.ID:                     "project key must contain only upper-case alphanumerical characters",
	ErrProjectHasPipeline.ID:                    "project contains a pipeline",
	ErrProjectHasApplication.ID:                 "project contains an application",
	ErrUnauthorized.ID:                          "not authenticated",
	ErrForbidden.ID:                             "forbidden",
	ErrPipelineNotFound.ID:                      "pipeline does not exist",
	ErrPipelineNotAttached.ID:                   "pipeline is not attached to application",
	ErrNoEnvironmentProvided.ID:                 "deployment and testing pipelines require an environnement",
	ErrEnvironmentProvided.ID:                   "build pipeline are not compatible with environment usage",
	ErrUnknownEnv.ID:                            "unknown environment",
	ErrEnvironmentExist.ID:                      "environment already exists",
	ErrNoPipelineBuild.ID:                       "this pipeline build does not exist",
	ErrApplicationNotFound.ID:                   "application does not exist",
	ErrGroupNotFound.ID:                         "group does not exist",
	ErrInvalidUsername.ID:                       "invalid username",
	ErrInvalidEmail.ID:                          "invalid email",
	ErrGroupPresent.ID:                          "group already present",
	ErrInvalidName.ID:                           "invalid name",
	ErrInvalidUser.ID:                           "invalid user or password",
	ErrBuildArchived.ID:                         "Cannot restart this build because it has been archived",
	ErrNoEnvironment.ID:                         "environment does not exist",
	ErrModelNameExist.ID:                        "worker model name already used",
	ErrNoWorkerModel.ID:                         "worker model does not exist",
	ErrNoProject.ID:                             "project does not exist",
	ErrVariableExists.ID:                        "variable already exists",
	ErrInvalidGroupPattern.ID:                   "group name must respect '^[a-zA-Z0-9.-_-]{1,}$'",
	ErrGroupExists.ID:                           "group already exists",
	ErrNotEnoughAdmin.ID:                        "not enough group admin left",
	ErrInvalidProjectName.ID:                    "project name must not be empty",
	ErrInvalidApplicationPattern.ID:             "application name must respect '^[a-zA-Z0-9.-_-]{1,}$'",
	ErrInvalidPipelinePattern.ID:                "pipeline name must respect '^[a-zA-Z0-9.-_-]{1,}$'",
	ErrNotFound.ID:                              "resource not found",
	ErrNoWorkerModelCapa.ID:                     "capability not found",
	ErrNoHook.ID:                                "hook not found",
	ErrNoAttachedPipeline.ID:                    "pipeline not attached to the application",
	ErrNoReposManager.ID:                        "repositories manager not found",
	ErrNoReposManagerAuth.ID:                    "CDS authentication error, please contact CDS administrator",
	ErrNoReposManagerClientAuth.ID:              "Repository manager authentication error, please check your credentials",
	ErrRepoNotFound.ID:                          "repository not found",
	ErrSecretStoreUnreachable.ID:                "could not reach secret backend to fetch secret key",
	ErrSecretKeyFetchFailed.ID:                  "error while fetching key from secret backend",
	ErrCommitsFetchFailed.ID:                    "unable to retrieves commits",
	ErrInvalidGoPath.ID:                         "invalid gopath",
	ErrInvalidSecretFormat.ID:                   "cannot decrypt secret, invalid format",
	ErrSessionNotFound.ID:                       "invalid session",
	ErrNoPreviousSuccess.ID:                     "there is no previous success version for this pipeline",
	ErrNoEnvExecution.ID:                        "you don't have execution right on this environment",
	ErrInvalidSecretValue.ID:                    "secret value not specified",
	ErrPipelineHasApplication.ID:                "pipeline still used by an application",
	ErrNoDirectSecretUse.ID:                     "usage of 'password' parameter is not allowed",
	ErrNoBranch.ID:                              "branch not found in repository",
	ErrLDAPConn.ID:                              "LDAP server connection error",
	ErrServiceUnavailable.ID:                    "service currently unavailable or down for maintenance",
	ErrParseUserNotification.ID:                 "unrecognized user notification settings",
	ErrNotSupportedUserNotification.ID:          "unsupported user notification",
	ErrGroupNeedAdmin.ID:                        "need at least 1 administrator",
	ErrGroupNeedWrite.ID:                        "need at least 1 group with write permission",
	ErrNoVariable.ID:                            "variable not found",
	ErrPluginInvalid.ID:                         "invalid plugin",
	ErrConflict.ID:                              "object conflict",
	ErrPipelineAlreadyAttached.ID:               "pipeline already attached to this application",
	ErrApplicationExist.ID:                      "application already exists",
	ErrBranchNameNotProvided.ID:                 "branchName parameter must be provided",
	ErrInfiniteTriggerLoop.ID:                   "infinite trigger loop are forbidden",
	ErrInvalidResetUser.ID:                      "invalid user or email",
	ErrUserConflict.ID:                          "this user already exists",
	ErrWrongRequest.ID:                          "wrong request",
	ErrAlreadyExist.ID:                          "already exists",
	ErrInvalidType.ID:                           "invalid type",
	ErrParentApplicationAndPipelineMandatory.ID: "parent application and pipeline are mandatory",
	ErrNoParentBuildFound.ID:                    "no parent build found",
	ErrParameterExists.ID:                       "parameter already exists",
	ErrNoHatchery.ID:                            "No hatchery found",
	ErrInvalidWorkerStatus.ID:                   "Worker status is invalid",
	ErrInvalidToken.ID:                          "Invalid token",
	ErrAppBuildingPipelines.ID:                  "Cannot delete application, there are building pipelines",
	ErrInvalidTimezone.ID:                       "Invalid timezone",
	ErrEnvironmentCannotBeDeleted.ID:            "Environment cannot be deleted. It is still in used",
	ErrInvalidPipeline.ID:                       "Invalid pipeline",
	ErrKeyNotFound.ID:                           "Key not found",
	ErrPipelineAlreadyExists.ID:                 "Pipeline already exists",
	ErrJobAlreadyBooked.ID:                      "Job already booked",
	ErrPipelineBuildNotFound.ID:                 "Pipeline build not found",
	ErrAlreadyTaken.ID:                          "This job is already taken by another worker",
	ErrWorkflowNotFound.ID:                      "Workflow not found",
	ErrWorkflowNodeNotFound.ID:                  "Workflow node not found",
	ErrWorkflowInvalidRoot.ID:                   "Invalid workflow root",
	ErrWorkflowNodeRef.ID:                       "Invalid workflow node reference",
	ErrWorkflowInvalid.ID:                       "Invalid workflow",
	ErrWorkflowNodeJoinNotFound.ID:              "Workflow node join not found",
	ErrInvalidJobRequirement.ID:                 "Invalid job requirement",
	ErrNotImplemented.ID:                        "This functionality isn't implemented",
	ErrParameterNotExists.ID:                    "This parameter doesn't exist",
	ErrUnknownKeyType.ID:                        "Unknown key type",
	ErrInvalidKeyPattern.ID:                     "key name must respect the following pattern: '^[a-zA-Z0-9.-_-]{1,}$'",
	ErrWebhookConfigDoesNotMatch.ID:             "Webhook config does not match",
	ErrPipelineUsedByWorkflow.ID:                "pipeline still used by a workflow",
	ErrMethodNotAllowed.ID:                      "Method not allowed",
	ErrInvalidNodeNamePattern.ID:                "Node name must respect the following pattern: '^[a-zA-Z0-9.-_-]{1,}$'",
	ErrWorkflowNodeParentNotRun.ID:              "Cannot run a node if their parents have never been launched",
	ErrDefaultGroupPermission.ID:                "Only read permission is allowed to default group",
	ErrLastGroupWithWriteRole.ID:                "The last group must have the write permission",
	ErrInvalidEmailDomain.ID:                    "Invalid domain",
	ErrWorkflowNodeRunJobNotFound.ID:            "Job not found",
	ErrBuiltinKeyNotFound.ID:                    "Encryption Key not found",
	ErrStepNotFound.ID:                          "Step not found",
	ErrWorkerModelAlreadyBooked.ID:              "Worker Model already booked",
	ErrConditionsNotOk.ID:                       "Cannot run this pipeline because launch conditions aren't ok",
	ErrDownloadInvalidOS.ID:                     "OS Invalid. Should be linux, darwin, freebsd or windows",
	ErrDownloadInvalidArch.ID:                   "Architecture invalid. Should be 386, i386, i686, amd64, x86_64 or arm (depends on OS)",
	ErrDownloadInvalidName.ID:                   "Invalid name",
	ErrDownloadDoesNotExist.ID:                  "File does not exist",
	ErrTokenNotFound.ID:                         "Token does not exist",
	ErrWorkflowNotificationNodeRef.ID:           "An invalid workflow node reference has been found, if you want to delete a pipeline from your workflow check if this pipeline isn't referenced in your notifications list",
	ErrInvalidJobRequirementDuplicateModel.ID:   "Invalid job requirements: you can't select multiple worker models",
}

var errorsFrench = map[int]string{
	ErrUnknownError.ID:                          "erreur interne",
	ErrActionAlreadyUpdated.ID:                  "le status de l'action a déjà été mis à jour",
	ErrNoAction.ID:                              "l'action n'existe pas",
	ErrActionLoop.ID:                            "la définition de l'action contient une boucle récursive",
	ErrInvalidID.ID:                             "l'ID doit être un nombre entier",
	ErrInvalidProject.ID:                        "projet manquant",
	ErrInvalidProjectKey.ID:                     "la clef de project doit uniquement contenir des lettres majuscules et des chiffres",
	ErrProjectHasPipeline.ID:                    "le project contient un pipeline",
	ErrProjectHasApplication.ID:                 "le project contient une application",
	ErrUnauthorized.ID:                          "authentification invalide",
	ErrForbidden.ID:                             "accès refusé",
	ErrPipelineNotFound.ID:                      "le pipeline n'existe pas",
	ErrPipelineNotAttached.ID:                   "le pipeline n'est pas lié à l'application",
	ErrNoEnvironmentProvided.ID:                 "les pipelines de déploiement et de tests requièrent un environnement",
	ErrEnvironmentProvided.ID:                   "une pipeline de build ne nécessite pas d'environnement",
	ErrUnknownEnv.ID:                            "environnement inconnu",
	ErrEnvironmentExist.ID:                      "l'environnement existe",
	ErrNoPipelineBuild.ID:                       "ce build n'existe pas",
	ErrApplicationNotFound.ID:                   "l'application n'existe pas",
	ErrGroupNotFound.ID:                         "le groupe n'existe pas",
	ErrInvalidUsername.ID:                       "nom d'utilisateur invalide",
	ErrInvalidEmail.ID:                          "addresse email invalide",
	ErrGroupPresent.ID:                          "le groupe est déjà présent",
	ErrInvalidName.ID:                           "le nom est invalide",
	ErrInvalidUser.ID:                           "mauvaise combinaison compte/mot de passe utilisateur",
	ErrBuildArchived.ID:                         "impossible de relancer ce build car il a été archivé",
	ErrNoEnvironment.ID:                         "l'environement n'existe pas",
	ErrModelNameExist.ID:                        "le nom du modèle de worker est déjà utilisé",
	ErrNoWorkerModel.ID:                         "le modèle de worker n'existe pas",
	ErrNoProject.ID:                             "le projet n'existe pas",
	ErrVariableExists.ID:                        "la variable existe déjà",
	ErrInvalidGroupPattern.ID:                   "nom de groupe invalide '^[a-zA-Z0-9.-_-]{1,}$'",
	ErrGroupExists.ID:                           "le groupe existe déjà",
	ErrNotEnoughAdmin.ID:                        "pas assez d'admin restant",
	ErrInvalidProjectName.ID:                    "nom de project vide non autorisé",
	ErrInvalidApplicationPattern.ID:             "nom de l'application invalide '^[a-zA-Z0-9.-_-]{1,}$'",
	ErrInvalidPipelinePattern.ID:                "nom du pipeline invalide '^[a-zA-Z0-9.-_-]{1,}$'",
	ErrNotFound.ID:                              "la ressource n'existe pas",
	ErrNoWorkerModelCapa.ID:                     "la capacité n'existe pas",
	ErrNoHook.ID:                                "le hook n'existe pas",
	ErrNoAttachedPipeline.ID:                    "le pipeline n'est pas lié à l'application",
	ErrNoReposManager.ID:                        "le gestionnaire de dépôt n'existe pas",
	ErrNoReposManagerAuth.ID:                    "connexion de CDS au gestionnaire de dépôt refusée, merci de contacter l'administrateur",
	ErrNoReposManagerClientAuth.ID:              "connexion au gestionnaire de dépôts refusée, vérifier vos accréditations",
	ErrRepoNotFound.ID:                          "le dépôt n'existe pas",
	ErrSecretStoreUnreachable.ID:                "impossible de contacter vault",
	ErrSecretKeyFetchFailed.ID:                  "erreur pendnat la récuperation de la clef de chiffrement",
	ErrCommitsFetchFailed.ID:                    "impossible de retrouver les changements",
	ErrInvalidGoPath.ID:                         "le gopath n'est pas valide",
	ErrInvalidSecretFormat.ID:                   "impossibe de dechiffrer le secret, format invalide",
	ErrSessionNotFound.ID:                       "session invalide",
	ErrNoPreviousSuccess.ID:                     "il n'y a aucune précédente version en succès pour ce pipeline",
	ErrNoEnvExecution.ID:                        "vous n'avez pas les droits d'éxécution sur cet environnement",
	ErrInvalidSecretValue.ID:                    "valeur du secret non specifiée",
	ErrPipelineHasApplication.ID:                "le pipeline est utilisé par une application",
	ErrNoDirectSecretUse.ID:                     "l'utilisation du type de paramêtre 'password' est impossible",
	ErrNoBranch.ID:                              "la branche est introuvable dans le dépôt",
	ErrLDAPConn.ID:                              "erreur de connexion au serveur LDAP",
	ErrServiceUnavailable.ID:                    "service temporairement indisponible ou en maintenance",
	ErrParseUserNotification.ID:                 "notification non reconnue",
	ErrNotSupportedUserNotification.ID:          "notification non supportée",
	ErrGroupNeedAdmin.ID:                        "il faut au moins 1 administrateur",
	ErrGroupNeedWrite.ID:                        "il faut au moins 1 groupe avec les droits d'écriture",
	ErrNoVariable.ID:                            "la variable n'existe pas",
	ErrPluginInvalid.ID:                         "plugin non valide",
	ErrConflict.ID:                              "l'objet est en conflit",
	ErrPipelineAlreadyAttached.ID:               "le pipeline est déjà attaché à cette application",
	ErrApplicationExist.ID:                      "une application du même nom existe déjà",
	ErrBranchNameNotProvided.ID:                 "le paramètre branchName doit être envoyé",
	ErrInfiniteTriggerLoop.ID:                   "création d'une boucle de trigger infinie interdite",
	ErrInvalidResetUser.ID:                      "mauvaise combinaison compte/mail utilisateur",
	ErrUserConflict.ID:                          "cet utilisateur existe deja",
	ErrWrongRequest.ID:                          "la requête est incorrecte",
	ErrAlreadyExist.ID:                          "conflit",
	ErrInvalidType.ID:                           "type non valide",
	ErrParentApplicationAndPipelineMandatory.ID: "application et pipeline parents obligatoires",
	ErrNoParentBuildFound.ID:                    "aucun build parent n'a pu être trouvé",
	ErrParameterExists.ID:                       "le paramètre existe déjà",
	ErrNoHatchery.ID:                            "La hatchery n'existe pas",
	ErrInvalidWorkerStatus.ID:                   "Le status du worker est incorrect",
	ErrInvalidToken.ID:                          "Token non valide",
	ErrAppBuildingPipelines.ID:                  "Impossible de supprimer l'application, il y a pipelines en cours",
	ErrInvalidTimezone.ID:                       "Fuseau horaire invalide",
	ErrEnvironmentCannotBeDeleted.ID:            "L'environement ne peut etre supprimé. Il est encore utilisé.",
	ErrInvalidPipeline.ID:                       "Pipeline invalide",
	ErrKeyNotFound.ID:                           "Clé introuvable",
	ErrPipelineAlreadyExists.ID:                 "Le pipeline existe déjà",
	ErrJobAlreadyBooked.ID:                      "Le job est déjà réservé",
	ErrPipelineBuildNotFound.ID:                 "Le pipeline build n'a pu être trouvé",
	ErrAlreadyTaken.ID:                          "Ce job est déjà en cours de traitement par un autre worker",
	ErrWorkflowNotFound.ID:                      "Workflow introuvable",
	ErrWorkflowNodeNotFound.ID:                  "Noeud de Workflow introuvable",
	ErrWorkflowInvalidRoot.ID:                   "Racine de Workflow invalide",
	ErrWorkflowNodeRef.ID:                       "Référence de noeud de workflow invalide",
	ErrWorkflowInvalid.ID:                       "Workflow invalide",
	ErrWorkflowNodeJoinNotFound.ID:              "Jointure introuvable",
	ErrInvalidJobRequirement.ID:                 "Pré-requis de Job invalide",
	ErrNotImplemented.ID:                        "La fonctionnalité n'est pas implémentée",
	ErrParameterNotExists.ID:                    "Ce paramètre n'existe pas",
	ErrUnknownKeyType.ID:                        "Le type de clé n'est pas connu",
	ErrInvalidKeyPattern.ID:                     "le nom de la clé doit respecter le pattern suivant; '^[a-zA-Z0-9.-_-]{1,}$'",
	ErrWebhookConfigDoesNotMatch.ID:             "la configuration du webhook ne correspond pas",
	ErrPipelineUsedByWorkflow.ID:                "le pipeline est utilisé par un workflow",
	ErrMethodNotAllowed.ID:                      "La méthode n'est pas autorisée",
	ErrInvalidNodeNamePattern.ID:                "Le nom du noeud du workflow doit respecter le pattern suivant; '^[a-zA-Z0-9.-_-]{1,}$'",
	ErrWorkflowNodeParentNotRun.ID:              "Il est interdit de lancer un noeuds si ses parents n'ont jamais été lancés",
	ErrDefaultGroupPermission.ID:                "Le groupe par défaut ne peut être utilisé qu'en lecture seule",
	ErrLastGroupWithWriteRole.ID:                "Le dernier groupe doit avoir les droits d'écriture",
	ErrInvalidEmailDomain.ID:                    "Domaine invalide",
	ErrWorkflowNodeRunJobNotFound.ID:            "Job non trouvé",
	ErrBuiltinKeyNotFound.ID:                    "Clé de chiffrage introuvable",
	ErrStepNotFound.ID:                          "Step introuvable",
	ErrWorkerModelAlreadyBooked.ID:              "Le modèle de worker est déjà réservé",
	ErrConditionsNotOk.ID:                       "Impossible de démarrer ce pipeline car les conditions de lancement ne sont pas respectées",
	ErrDownloadInvalidOS.ID:                     "OS invalide. L'OS doit être linux, darwin, freebsd ou windows",
	ErrDownloadInvalidArch.ID:                   "Architecture invalide. L'architecture doit être 386, i386, i686, amd64, x86_64 ou arm (dépendant de l'OS)",
	ErrDownloadInvalidName.ID:                   "Nom invalide",
	ErrDownloadDoesNotExist.ID:                  "Le fichier n'existe pas",
	ErrTokenNotFound.ID:                         "Le token n'existe pas",
	ErrWorkflowNotificationNodeRef.ID:           "Une référence de noeud de workflow est invalide dans vos notifications (si vous souhaitez supprimer un pipeline vérifiez qu'il ne soit plus référencé dans la liste de vos notifications)",
	ErrInvalidJobRequirementDuplicateModel.ID:   "Pré-requis de job invalides: vous ne pouvez pas séléctionnez plusieurs modèles de worker",
}

var errorsLanguages = []map[int]string{
	errorsAmericanEnglish,
	errorsFrench,
}

// NewError just set an error with a root cause
func NewError(target Error, root error) Error {
	target.Root = root
	return target
}

// WrapError constructs a stack of errors, adding context to the preceding error.
func WrapError(err error, format string, args ...interface{}) error {
	return errors.Wrap(err, fmt.Sprintf(format, args...))
}

// SetError completes an error
func SetError(err error, format string, args ...interface{}) error {
	msg := fmt.Errorf("%s: %v", fmt.Sprintf(format, args...), err)
	cdsErr, ok := err.(Error)
	if !ok {
		return NewError(cdsErr, msg)
	}
	return msg
}

// ProcessError tries to recognize given error and return error message in a language matching Accepted-Language
func ProcessError(target error, al string) (string, Error) {
	// will recursively retrieve the topmost error which does not implement causer, which is assumed to be the original cause
	target = errors.Cause(target)
	cdsErr, ok := target.(Error)
	if !ok {
		return errorsAmericanEnglish[ErrUnknownError.ID], ErrUnknownError
	}
	acceptedLanguages, _, err := language.ParseAcceptLanguage(al)
	if err != nil {
		return errorsAmericanEnglish[ErrUnknownError.ID], ErrUnknownError
	}

	tag, _, _ := matcher.Match(acceptedLanguages...)
	var msg string
	switch tag {
	case language.French:
		msg, ok = errorsFrench[cdsErr.ID]
		break
	case language.AmericanEnglish:
		msg, ok = errorsAmericanEnglish[cdsErr.ID]
		break
	default:
		msg, ok = errorsAmericanEnglish[cdsErr.ID]
		break
	}

	if !ok {
		return errorsAmericanEnglish[ErrUnknownError.ID], ErrUnknownError
	}
	if cdsErr.Root != nil {
		cdsErrRoot, ok := cdsErr.Root.(*Error)
		var rootMsg string
		if ok {
			rootMsg, _ = ProcessError(cdsErrRoot, al)
		} else {
			rootMsg = fmt.Sprintf("%v", cdsErr.Root.Error())
		}

		msg = fmt.Sprintf("%s (caused by: %s)", msg, rootMsg)
	}
	return msg, cdsErr
}

// Exit func display an error message on stderr and exit 1
func Exit(format string, args ...interface{}) {
	if len(format) > 0 && format[:len(format)-1] != "\n" {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

// Error type
type Error struct {
	ID      int    `json:"-"`
	Status  int    `json:"-"`
	Message string `json:"message"`
	Root    error  `json:"-"`
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

func (e Error) String() string {
	var msg = e.Message
	if e.Message == "" {
		msg1, ok := errorsAmericanEnglish[e.ID]
		if ok {
			msg = msg1
		}
	}

	if e.Root != nil {
		cdsErrRoot, ok := e.Root.(*Error)
		var rootMsg string
		if ok {
			rootMsg, _ = ProcessError(cdsErrRoot, "en-US")
		} else {
			rootMsg = fmt.Sprintf("%s [%T]", e.Root.Error(), e.Root)
		}

		msg = fmt.Sprintf("%s (caused by: %s)", msg, rootMsg)
	}

	return msg
}

func (e Error) Error() string {
	return e.String()
}

// ErrorIs returns true if error is same as and sdk.Error Message
// this func checks msg in all languages
func ErrorIs(err error, t Error) bool {
	if err == nil {
		return false
	}
	for _, l := range errorsLanguages {
		if l[t.ID] == err.Error() {
			return true
		}
	}
	return false
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
	*e = append(*e, err)
}

// IsEmpty return true if MultiError is empty, false otherwise
func (e *MultiError) IsEmpty() bool {
	return len(*e) == 0
}
