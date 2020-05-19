package sdk

import (
	"fmt"

	"github.com/pkg/errors"
	"golang.org/x/text/language"
)

type (
	lang language.Tag
	trad map[lang]string
)

//Supported API language
var (
	FR = lang(language.French)
	EN = lang(language.AmericanEnglish)
)

//Message list
var (
	MsgAppCreated                           = &Message{"MsgAppCreated", trad{FR: "L'application %s a été créée avec succès", EN: "Application %s successfully created"}, nil, RunInfoTypInfo}
	MsgAppUpdated                           = &Message{"MsgAppUpdated", trad{FR: "L'application %s a été mise à jour avec succès", EN: "Application %s successfully updated"}, nil, RunInfoTypInfo}
	MsgPipelineCreated                      = &Message{"MsgPipelineCreated", trad{FR: "Le pipeline %s a été créé avec succès", EN: "Pipeline %s successfully created"}, nil, RunInfoTypInfo}
	MsgPipelineCreationAborted              = &Message{"MsgPipelineCreationAborted", trad{FR: "La création du pipeline %s a été abandonnée", EN: "Pipeline %s creation aborted"}, nil, RunInfoTypeError}
	MsgPipelineExists                       = &Message{"MsgPipelineExists", trad{FR: "Le pipeline %s existe déjà", EN: "Pipeline %s already exists"}, nil, RunInfoTypInfo}
	MsgAppVariablesCreated                  = &Message{"MsgAppVariablesCreated", trad{FR: "Les variables ont été ajoutées avec succès sur l'application %s", EN: "Application variables for %s are successfully created"}, nil, RunInfoTypInfo}
	MsgAppKeyCreated                        = &Message{"MsgAppKeyCreated", trad{FR: "La clé %s %s a été créée sur l'application %s", EN: "%s key %s created on application %s"}, nil, RunInfoTypInfo}
	MsgEnvironmentExists                    = &Message{"MsgEnvironmentExists", trad{FR: "L'environnement %s existe déjà", EN: "Environment %s already exists"}, nil, RunInfoTypInfo}
	MsgEnvironmentCreated                   = &Message{"MsgEnvironmentCreated", trad{FR: "L'environnement %s a été créé avec succès", EN: "Environment %s successfully created"}, nil, RunInfoTypInfo}
	MsgEnvironmentVariableUpdated           = &Message{"MsgEnvironmentVariableUpdated", trad{FR: "La variable %s de l'environnement %s a été mise à jour", EN: "Variable %s on environment %s has been updated"}, nil, RunInfoTypInfo}
	MsgEnvironmentVariableCannotBeUpdated   = &Message{"MsgEnvironmentVariableCannotBeUpdated", trad{FR: "La variable %s de l'environnement %s n'a pu être mise à jour : %s", EN: "Variable %s on environment %s cannot be updated: %s"}, nil, RunInfoTypeError}
	MsgEnvironmentVariableCreated           = &Message{"MsgEnvironmentVariableCreated", trad{FR: "La variable %s de l'environnement %s a été ajoutée", EN: "Variable %s on environment %s has been added"}, nil, RunInfoTypInfo}
	MsgEnvironmentVariableCannotBeCreated   = &Message{"MsgEnvironmentVariableCannotBeCreated", trad{FR: "La variable %s de l'environnement %s n'a pu être ajoutée : %s", EN: "Variable %s on environment %s cannot be added: %s"}, nil, RunInfoTypeError}
	MsgEnvironmentGroupUpdated              = &Message{"MsgEnvironmentGroupUpdated", trad{FR: "Le groupe %s de l'environnement %s a été mis à jour", EN: "Group %s on environment %s has been updated"}, nil, RunInfoTypInfo}
	MsgEnvironmentGroupCannotBeUpdated      = &Message{"MsgEnvironmentGroupCannotBeUpdated", trad{FR: "Le groupe %s de l'environnement %s n'a pu être mis à jour : %s", EN: "Group %s on environment %s cannot be updated: %s"}, nil, RunInfoTypeError}
	MsgEnvironmentGroupCreated              = &Message{"MsgEnvironmentGroupCreated", trad{FR: "Le groupe %s de l'environnement %s a été ajouté", EN: "Group %s on environment %s has been added"}, nil, RunInfoTypInfo}
	MsgEnvironmentGroupCannotBeCreated      = &Message{"MsgEnvironmentGroupCannotBeCreated", trad{FR: "Le groupe %s de l'environnement %s n'a pu être ajouté : %s", EN: "Group %s on environment %s cannot be added: %s"}, nil, RunInfoTypeError}
	MsgEnvironmentGroupDeleted              = &Message{"MsgEnvironmentGroupDeleted", trad{FR: "Le groupe %s de l'environnement %s a été supprimé", EN: "Group %s on environment %s has been deleted"}, nil, RunInfoTypInfo}
	MsgEnvironmentGroupCannotBeDeleted      = &Message{"MsgEnvironmentGMsgEnvironmentGroupCannotBeDeletedroupCannotBeCreated", trad{FR: "Le groupe %s de l'environnement %s n'a pu être supprimé : %s", EN: "Group %s on environment %s cannot be deleted: %s"}, nil, RunInfoTypeError}
	MsgEnvironmentKeyCreated                = &Message{"MsgEnvironmentKeyCreated", trad{FR: "La clé %s %s a été créée sur l'environnement %s", EN: "%s key %s created on environment %s"}, nil, RunInfoTypInfo}
	MsgJobNotValidActionNotFound            = &Message{"MsgJobNotValidActionNotFound", trad{FR: "Erreur de validation du Job %s : L'action %s à l'étape %d n'a pas été trouvée", EN: "Job %s validation Failure: Unknown action %s on step #%d"}, nil, RunInfoTypeError}
	MsgJobNotValidInvalidActionParameter    = &Message{"MsgJobNotValidInvalidActionParameter", trad{FR: "Erreur de validation du Job %s : Le paramètre %s de l'étape %d - %s est invalide", EN: "Job %s validation Failure: Invalid parameter %s on step #%d %s"}, nil, RunInfoTypeError}
	MsgPipelineGroupUpdated                 = &Message{"MsgPipelineGroupUpdated", trad{FR: "Les permissions du groupe %s sur le pipeline %s on été mises à jour", EN: "Permission for group %s on pipeline %s has been updated"}, nil, RunInfoTypInfo}
	MsgPipelineGroupAdded                   = &Message{"MsgPipelineGroupAdded", trad{FR: "Les permissions du groupe %s sur le pipeline %s on été ajoutées", EN: "Permission for group %s on pipeline %s has been added"}, nil, RunInfoTypInfo}
	MsgPipelineGroupDeleted                 = &Message{"MsgPipelineGroupDeleted", trad{FR: "Les permissions du groupe %s sur le pipeline %s on été supprimées", EN: "Permission for group %s on pipeline %s has been deleted"}, nil, RunInfoTypInfo}
	MsgPipelineStageUpdated                 = &Message{"MsgPipelineStageUpdated", trad{FR: "Le stage %s a été mis à jour", EN: "Stage %s updated"}, nil, RunInfoTypInfo}
	MsgPipelineStageUpdating                = &Message{"MsgPipelineStageUpdating", trad{FR: "Mise à jour du stage %s en cours...", EN: "Updating stage %s ..."}, nil, RunInfoTypInfo}
	MsgPipelineStageDeletingOldJobs         = &Message{"MsgPipelineStageDeletingOldJobs", trad{FR: "Suppression des anciens jobs du stage %s en cours...", EN: "Deleting old jobs in stage %s ..."}, nil, RunInfoTypInfo}
	MsgPipelineStageInsertingNewJobs        = &Message{"MsgPipelineStageInsertingNewJobs", trad{FR: "Insertion des nouveaux jobs dans le stage %s en cours...", EN: "Inserting new jobs in stage %s ..."}, nil, RunInfoTypInfo}
	MsgPipelineStageAdded                   = &Message{"MsgPipelineStageAdded", trad{FR: "Le stage %s a été ajouté", EN: "Stage %s added"}, nil, RunInfoTypInfo}
	MsgPipelineStageDeleted                 = &Message{"MsgPipelineStageDeleted", trad{FR: "Le stage %s a été supprimé", EN: "Stage %s deleted"}, nil, RunInfoTypInfo}
	MsgPipelineJobUpdated                   = &Message{"MsgPipelineJobUpdated", trad{FR: "Le job %s du stage %s a été mis à jour", EN: "Job %s in stage %s updated"}, nil, RunInfoTypInfo}
	MsgPipelineJobAdded                     = &Message{"MsgPipelineJobAdded", trad{FR: "Le job %s du stage %s a été ajouté", EN: "Job %s in stage %s added"}, nil, RunInfoTypInfo}
	MsgPipelineJobDeleted                   = &Message{"MsgPipelineJobDeleted", trad{FR: "Le job %s du stage %s a été supprimé", EN: "Job %s in stage %s deleted"}, nil, RunInfoTypInfo}
	MsgSpawnInfoHatcheryStarts              = &Message{"MsgSpawnInfoHatcheryStarts", trad{FR: "La Hatchery %s a démarré le lancement du worker avec le modèle %s", EN: "Hatchery %s starts spawn worker with model %s"}, nil, RunInfoTypInfo}
	MsgSpawnInfoHatcheryErrorSpawn          = &Message{"MsgSpawnInfoHatcheryErrorSpawn", trad{FR: "Une erreur est survenue lorsque la Hatchery %s a démarré un worker avec le modèle %s après %s, err:%s", EN: "Error while Hatchery %s spawn worker with model %s after %s, err:%s"}, nil, RunInfoTypeError}
	MsgSpawnInfoHatcheryStartsSuccessfully  = &Message{"MsgSpawnInfoHatcheryStartsSuccessfully", trad{FR: "La Hatchery %s a démarré le worker %s avec succès en %s", EN: "Hatchery %s spawn worker %s successfully in %s"}, nil, RunInfoTypInfo}
	MsgSpawnInfoHatcheryStartDockerPull     = &Message{"MsgSpawnInfoHatcheryStartDockerPull", trad{FR: "La Hatchery %s a démarré le docker pull de l'image %s...", EN: "Hatchery %s starts docker pull %s..."}, nil, RunInfoTypInfo}
	MsgSpawnInfoHatcheryEndDockerPull       = &Message{"MsgSpawnInfoHatcheryEndDockerPull", trad{FR: "La Hatchery %s a terminé le docker pull de l'image %s", EN: "Hatchery %s docker pull %s done"}, nil, RunInfoTypInfo}
	MsgSpawnInfoHatcheryEndDockerPullErr    = &Message{"MsgSpawnInfoHatcheryEndDockerPullErr", trad{FR: "⚠ La Hatchery %s a terminé le docker pull de l'image %s en erreur: %s", EN: "⚠ Hatchery %s - docker pull %s done with error: %v"}, nil, RunInfoTypeError}
	MsgSpawnInfoDeprecatedModel             = &Message{"MsgSpawnInfoDeprecatedModel", trad{FR: "⚠ Attention vous utilisez un worker model (%s) déprécié", EN: "⚠ Pay attention you are using a deprecated worker model (%s)"}, nil, RunInfoTypeWarning}
	MsgSpawnInfoWorkerEnd                   = &Message{"MsgSpawnInfoWorkerEnd", trad{FR: "✓ Le worker %s a terminé et a passé %s à travailler sur les étapes", EN: "✓ Worker %s finished working on this job and took %s to work on the steps"}, nil, RunInfoTypInfo}
	MsgSpawnInfoJobInQueue                  = &Message{"MsgSpawnInfoJobInQueue", trad{FR: "✓ Le job a été mis en file d'attente", EN: "✓ Job has been queued"}, nil, RunInfoTypInfo}
	MsgSpawnInfoJobTaken                    = &Message{"MsgSpawnInfoJobTaken", trad{FR: "Le job %s a été pris par le worker %s", EN: "Job %s was taken by worker %s"}, nil, RunInfoTypInfo}
	MsgSpawnInfoJobTakenWorkerVersion       = &Message{"MsgSpawnInfoJobTakenWorkerVersion", trad{FR: "Worker %s version:%s os:%s arch:%s", EN: "Worker %s version:%s os:%s arch:%s"}, nil, RunInfoTypInfo}
	MsgSpawnInfoWorkerForJob                = &Message{"MsgSpawnInfoWorkerForJob", trad{FR: "Ce worker %s a été créé pour lancer ce job", EN: "This worker %s was created to take this action"}, nil, RunInfoTypInfo}
	MsgSpawnInfoWorkerForJobError           = &Message{"MsgSpawnInfoWorkerForJobError", trad{FR: "⚠ Ce worker %s a été créé pour lancer ce job, mais ne possède pas tous les pré-requis. Vérifiez que les prérequis suivants:%s", EN: "⚠ This worker %s was created to take this action, but does not have all prerequisites. Please verify the following prerequisites:%s"}, nil, RunInfoTypeError}
	MsgSpawnInfoJobError                    = &Message{"MsgSpawnInfoJobError", trad{FR: "⚠ Impossible de lancer ce job : %s", EN: "⚠ Unable to run this job: %s"}, nil, RunInfoTypInfo}
	MsgWorkflowStarting                     = &Message{"MsgWorkflowStarting", trad{FR: "Le workflow %s#%s a été démarré", EN: "Workflow %s#%s has been started"}, nil, RunInfoTypInfo}
	MsgWorkflowError                        = &Message{"MsgWorkflowError", trad{FR: "⚠ Une erreur est survenue: %v", EN: "⚠ An error has occurred: %v"}, nil, RunInfoTypeError}
	MsgWorkflowConditionError               = &Message{"MsgWorkflowConditionError", trad{FR: "Les conditions de lancement ne sont pas respectées.", EN: "Run conditions aren't ok."}, nil, RunInfoTypInfo}
	MsgWorkflowNodeStop                     = &Message{"MsgWorkflowNodeStop", trad{FR: "Le pipeline a été arrété par %s", EN: "The pipeline has been stopped by %s"}, nil, RunInfoTypInfo}
	MsgWorkflowNodeMutex                    = &Message{"MsgWorkflowNodeMutex", trad{FR: "Le pipeline %s est mis en attente tant qu'il est en cours sur un autre run", EN: "The pipeline %s is waiting while it's running on another run"}, nil, RunInfoTypInfo}
	MsgWorkflowNodeMutexRelease             = &Message{"MsgWorkflowNodeMutexRelease", trad{FR: "Lancement du pipeline %s", EN: "Triggering pipeline %s"}, nil, RunInfoTypInfo}
	MsgWorkflowImportedUpdated              = &Message{"MsgWorkflowImportedUpdated", trad{FR: "Le workflow %s a été mis à jour", EN: "Workflow %s has been updated"}, nil, RunInfoTypInfo}
	MsgWorkflowImportedInserted             = &Message{"MsgWorkflowImportedInserted", trad{FR: "Le workflow %s a été créé", EN: "Workflow %s has been created"}, nil, RunInfoTypInfo}
	MsgSpawnInfoHatcheryCannotStartJob      = &Message{"MsgSpawnInfoHatcheryCannotStart", trad{FR: "Aucune hatchery n'a pu démarrer de worker respectant vos pré-requis de job, merci de les vérifier.", EN: "No hatchery can spawn a worker corresponding your job's requirements. Please check your job's requirements."}, nil, RunInfoTypeWarning}
	MsgWorkflowRunBranchDeleted             = &Message{"MsgWorkflowRunBranchDeleted", trad{FR: "La branche %s  a été supprimée", EN: "Branch %s has been deleted"}, nil, RunInfoTypInfo}
	MsgWorkflowTemplateImportedInserted     = &Message{"MsgWorkflowTemplateImportedInserted", trad{FR: "Le template de workflow %s/%s a été créé", EN: "Workflow template %s/%s has been created"}, nil, RunInfoTypInfo}
	MsgWorkflowTemplateImportedUpdated      = &Message{"MsgWorkflowTemplateImportedUpdated", trad{FR: "Le template de workflow %s/%s a été mis à jour", EN: "Workflow template %s/%s has been updated"}, nil, RunInfoTypInfo}
	MsgWorkflowErrorBadPipelineName         = &Message{"MsgWorkflowErrorBadPipelineName", trad{FR: "Le pipeline %s indiqué dans votre fichier yaml de workflow n'existe pas", EN: "The pipeline %s mentioned in your workflow's yaml file doesn't exist"}, nil, RunInfoTypeError}
	MsgWorkflowErrorBadApplicationName      = &Message{"MsgWorkflowErrorBadApplicationName", trad{FR: "L'application %s indiquée dans votre fichier yaml de workflow n'existe pas ou ne correspond pas aux normes ^[a-zA-Z0-9._-]{1,}$", EN: "The application %s mentioned in your workflow's yaml file doesn't exist or is incorrect with ^[a-zA-Z0-9._-]{1,}$"}, nil, RunInfoTypeError}
	MsgWorkflowErrorBadEnvironmentName      = &Message{"MsgWorkflowErrorBadEnvironmentName", trad{FR: "L'environnement %s indiqué dans votre fichier yaml de workflow n'existe pas", EN: "The environment %s mentioned in your workflow's yaml file doesn't exist"}, nil, RunInfoTypeError}
	MsgWorkflowErrorBadIntegrationName      = &Message{"MsgWorkflowErrorBadIntegrationName", trad{FR: "L'intégration %s indiquée dans votre fichier yaml n'existe pas", EN: "The integration %s mentioned in your yaml file doesn't exist"}, nil, RunInfoTypeError}
	MsgWorkflowErrorBadCdsDir               = &Message{"MsgWorkflowErrorBadCdsDir", trad{FR: "Un problème est survenu avec votre répertoire .cds", EN: "A problem occurred about your .cds directory"}, nil, RunInfoTypeError}
	MsgWorkflowErrorUnknownKey              = &Message{"MsgWorkflowErrorUnknownKey", trad{FR: "La clé '%s' est incorrecte ou n'existe pas", EN: "The key '%s' is incorrect or doesn't exist"}, nil, RunInfoTypeError}
	MsgWorkflowErrorBadVCSStrategy          = &Message{"MsgWorkflowErrorBadVCSStrategy", trad{FR: "Vos informations vcs_* sont incorrectes", EN: "Your vcs_* fields are incorrects"}, nil, RunInfoTypeError}
	MsgWorkflowDeprecatedVersion            = &Message{"MsgWorkflowDeprecatedVersion", trad{FR: "La configuration yaml de votre workflow est dans un format déprécié. Exportez le avec la CLI `cdsctl workflow export %s %s`", EN: "The yaml workflow configuration format is deprecated. Export your workflow with CLI `cdsctl workflow export %s %s`"}, nil, RunInfoTypeWarning}
	MsgWorkflowGeneratedFromTemplateVersion = &Message{"MsgWorkflowGeneratedFromTemplateVersion", trad{FR: "Le workflow a été généré à partir du modèle de workflow: %s.", EN: "The workflow was generated from the template: %s"}, nil, RunInfoTypInfo}
)

// Messages contains all sdk Messages
var Messages = map[string]*Message{
	MsgAppCreated.ID:                           MsgAppCreated,
	MsgAppUpdated.ID:                           MsgAppUpdated,
	MsgPipelineCreated.ID:                      MsgPipelineCreated,
	MsgPipelineCreationAborted.ID:              MsgPipelineCreationAborted,
	MsgPipelineExists.ID:                       MsgPipelineExists,
	MsgAppVariablesCreated.ID:                  MsgAppVariablesCreated,
	MsgAppKeyCreated.ID:                        MsgAppKeyCreated,
	MsgEnvironmentExists.ID:                    MsgEnvironmentExists,
	MsgEnvironmentCreated.ID:                   MsgEnvironmentCreated,
	MsgEnvironmentVariableUpdated.ID:           MsgEnvironmentVariableUpdated,
	MsgEnvironmentVariableCannotBeUpdated.ID:   MsgEnvironmentVariableCannotBeUpdated,
	MsgEnvironmentVariableCreated.ID:           MsgEnvironmentVariableCreated,
	MsgEnvironmentVariableCannotBeCreated.ID:   MsgEnvironmentVariableCannotBeCreated,
	MsgEnvironmentGroupUpdated.ID:              MsgEnvironmentGroupUpdated,
	MsgEnvironmentGroupCannotBeUpdated.ID:      MsgEnvironmentGroupCannotBeUpdated,
	MsgEnvironmentGroupCreated.ID:              MsgEnvironmentGroupCreated,
	MsgEnvironmentGroupCannotBeCreated.ID:      MsgEnvironmentGroupCannotBeCreated,
	MsgEnvironmentGroupDeleted.ID:              MsgEnvironmentGroupDeleted,
	MsgEnvironmentGroupCannotBeDeleted.ID:      MsgEnvironmentGroupCannotBeDeleted,
	MsgEnvironmentKeyCreated.ID:                MsgEnvironmentKeyCreated,
	MsgJobNotValidActionNotFound.ID:            MsgJobNotValidActionNotFound,
	MsgJobNotValidInvalidActionParameter.ID:    MsgJobNotValidInvalidActionParameter,
	MsgPipelineGroupUpdated.ID:                 MsgPipelineGroupUpdated,
	MsgPipelineGroupAdded.ID:                   MsgPipelineGroupAdded,
	MsgPipelineGroupDeleted.ID:                 MsgPipelineGroupDeleted,
	MsgPipelineStageUpdated.ID:                 MsgPipelineStageUpdated,
	MsgPipelineStageUpdating.ID:                MsgPipelineStageUpdating,
	MsgPipelineStageDeletingOldJobs.ID:         MsgPipelineStageDeletingOldJobs,
	MsgPipelineStageInsertingNewJobs.ID:        MsgPipelineStageInsertingNewJobs,
	MsgPipelineStageAdded.ID:                   MsgPipelineStageAdded,
	MsgPipelineStageDeleted.ID:                 MsgPipelineStageDeleted,
	MsgPipelineJobUpdated.ID:                   MsgPipelineJobUpdated,
	MsgPipelineJobAdded.ID:                     MsgPipelineJobAdded,
	MsgPipelineJobDeleted.ID:                   MsgPipelineJobDeleted,
	MsgSpawnInfoHatcheryStarts.ID:              MsgSpawnInfoHatcheryStarts,
	MsgSpawnInfoHatcheryErrorSpawn.ID:          MsgSpawnInfoHatcheryErrorSpawn,
	MsgSpawnInfoHatcheryStartsSuccessfully.ID:  MsgSpawnInfoHatcheryStartsSuccessfully,
	MsgSpawnInfoHatcheryStartDockerPull.ID:     MsgSpawnInfoHatcheryStartDockerPull,
	MsgSpawnInfoHatcheryEndDockerPull.ID:       MsgSpawnInfoHatcheryEndDockerPull,
	MsgSpawnInfoHatcheryEndDockerPullErr.ID:    MsgSpawnInfoHatcheryEndDockerPullErr,
	MsgSpawnInfoDeprecatedModel.ID:             MsgSpawnInfoDeprecatedModel,
	MsgSpawnInfoWorkerEnd.ID:                   MsgSpawnInfoWorkerEnd,
	MsgSpawnInfoJobInQueue.ID:                  MsgSpawnInfoJobInQueue,
	MsgSpawnInfoJobTaken.ID:                    MsgSpawnInfoJobTaken,
	MsgSpawnInfoJobTakenWorkerVersion.ID:       MsgSpawnInfoJobTakenWorkerVersion,
	MsgSpawnInfoWorkerForJob.ID:                MsgSpawnInfoWorkerForJob,
	MsgSpawnInfoWorkerForJobError.ID:           MsgSpawnInfoWorkerForJobError,
	MsgSpawnInfoJobError.ID:                    MsgSpawnInfoJobError,
	MsgWorkflowStarting.ID:                     MsgWorkflowStarting,
	MsgWorkflowError.ID:                        MsgWorkflowError,
	MsgWorkflowConditionError.ID:               MsgWorkflowConditionError,
	MsgWorkflowNodeStop.ID:                     MsgWorkflowNodeStop,
	MsgWorkflowNodeMutex.ID:                    MsgWorkflowNodeMutex,
	MsgWorkflowNodeMutexRelease.ID:             MsgWorkflowNodeMutexRelease,
	MsgWorkflowImportedUpdated.ID:              MsgWorkflowImportedUpdated,
	MsgWorkflowImportedInserted.ID:             MsgWorkflowImportedInserted,
	MsgSpawnInfoHatcheryCannotStartJob.ID:      MsgSpawnInfoHatcheryCannotStartJob,
	MsgWorkflowRunBranchDeleted.ID:             MsgWorkflowRunBranchDeleted,
	MsgWorkflowTemplateImportedInserted.ID:     MsgWorkflowTemplateImportedInserted,
	MsgWorkflowTemplateImportedUpdated.ID:      MsgWorkflowTemplateImportedUpdated,
	MsgWorkflowErrorBadPipelineName.ID:         MsgWorkflowErrorBadPipelineName,
	MsgWorkflowErrorBadApplicationName.ID:      MsgWorkflowErrorBadApplicationName,
	MsgWorkflowErrorBadEnvironmentName.ID:      MsgWorkflowErrorBadEnvironmentName,
	MsgWorkflowErrorBadIntegrationName.ID:      MsgWorkflowErrorBadIntegrationName,
	MsgWorkflowErrorBadCdsDir.ID:               MsgWorkflowErrorBadCdsDir,
	MsgWorkflowErrorUnknownKey.ID:              MsgWorkflowErrorUnknownKey,
	MsgWorkflowErrorBadVCSStrategy.ID:          MsgWorkflowErrorBadVCSStrategy,
	MsgWorkflowDeprecatedVersion.ID:            MsgWorkflowDeprecatedVersion,
	MsgWorkflowGeneratedFromTemplateVersion.ID: MsgWorkflowGeneratedFromTemplateVersion,
}

//Message represent a struc format translated messages
type Message struct {
	ID     string
	Format trad
	Args   []interface{}
	Type   string
}

//NewMessage instanciantes a new message
func NewMessage(m *Message, args ...interface{}) Message {
	return Message{
		Format: m.Format,
		Args:   args,
		ID:     m.ID,
		Type:   m.Type,
	}
}

// SupportedLanguages on API errors
var (
	SupportedLanguages = []language.Tag{
		language.AmericanEnglish,
		language.French,
	}
	matcher = language.NewMatcher(SupportedLanguages)
)

//String returns formated string for the specified language
func (m *Message) String(al string) string {
	acceptedLanguages, _, err := language.ParseAcceptLanguage(al)
	if err != nil {
		return fmt.Sprintf(m.Format[EN], m.Args...)
	}

	t, _, _ := matcher.Match(acceptedLanguages...)
	switch t {
	case language.French, language.AmericanEnglish:
		return fmt.Sprintf(m.Format[lang(t)], m.Args...)
	default:
		return fmt.Sprintf(m.Format[EN], m.Args...)
	}
}

// MessagesToError returns a translated slices of messages as an error
func MessagesToError(messages []Message) error {
	var s string
	for i, err := range messages {
		if i != 0 {
			s += "; "
		}
		s += err.String(language.AmericanEnglish.String())
	}
	return errors.New(s)
}

// ErrorToMessage returns message from an error if possible
func ErrorToMessage(err error) (Message, bool) {
	cdsError := ExtractHTTPError(err, "EN")
	switch cdsError.ID {
	case ErrPipelineNotFound.ID:
		return NewMessage(MsgWorkflowErrorBadPipelineName, cdsError.Data), true
	case ErrEnvironmentNotFound.ID:
		return NewMessage(MsgWorkflowErrorBadEnvironmentName, cdsError.Data), true
	case ErrIntegrationtNotFound.ID:
		return NewMessage(MsgWorkflowErrorBadIntegrationName, cdsError.Data), true
	}

	return Message{}, false
}
