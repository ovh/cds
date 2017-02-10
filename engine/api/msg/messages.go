package msg

import "golang.org/x/text/language"
import "fmt"

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
	AppCreated                         = &Message{trad{FR: "L'application %s a été créée avec succès", EN: "Application %s successfully created"}, nil}
	PipelineCreated                    = &Message{trad{FR: "Le pipeline %s a été créé avec succès", EN: "Pipeline %s successfully created"}, nil}
	PipelineExists                     = &Message{trad{FR: "Le pipeline %s existe déjà", EN: "Pipeline %s already exist"}, nil}
	PipelineAttached                   = &Message{trad{FR: "Le pipeline %s a été attaché à l'application %s", EN: "Pipeline %s has been attached to application %s"}, nil}
	PipelineTriggerCreated             = &Message{trad{FR: "Le trigger du pipeline %s de l'application %s vers le pipeline %s l'application %s a été créé avec succès", EN: "Trigger from pipeline %s of application %s to pipeline %s attached to application %s successfully created"}, nil}
	AppGroupInheritPermission          = &Message{trad{FR: "Les permissions du projet sont appliquées sur l'application %s", EN: "Application %s inherits project permissions"}, nil}
	AppGroupSetPermission              = &Message{trad{FR: "Permission accordée au groupe %s sur l'application %s", EN: "Permission applied to group %s to application %s"}, nil}
	AppVariablesCreated                = &Message{trad{FR: "Les variables ont été ajoutées avec succès sur l'application %s", EN: "Application variable for %s are successfully created"}, nil}
	HookCreated                        = &Message{trad{FR: "Hook créé sur le depôt %s vers le pipeline %s", EN: "Hook created on repository %s to pipeline %s"}, nil}
	EnvironmentExists                  = &Message{trad{FR: "L'environnement %s existe déjà", EN: "Environment %s already exist"}, nil}
	EnvironmentCreated                 = &Message{trad{FR: "L'environnement %s a été créé avec succès", EN: "Environment %s successfully created"}, nil}
	EnvironmentVariableUpdated         = &Message{trad{FR: "La variable %s de l'environnement %s a été mise à jour", EN: "Variable %s on environment %s has been updated"}, nil}
	EnvironmentVariableCannotBeUpdated = &Message{trad{FR: "La variable %s de l'environnement %s n'a pu être mise à jour : %s", EN: "Variable %s on environment %s cannot be updated: %s"}, nil}
	EnvironmentVariableCreated         = &Message{trad{FR: "La variable %s de l'environnement %s a été ajoutée", EN: "Variable %s on environment %s has been added"}, nil}
	EnvironmentVariableCannotBeCreated = &Message{trad{FR: "La variable %s de l'environnement %s n'a pu être ajoutée : %s", EN: "Variable %s on environment %s cannot be added: %s"}, nil}
	EnvironmentGroupUpdated            = &Message{trad{FR: "Le groupe %s de l'environnement %s a été mis à jour", EN: "Group %s on environment %s has been updated"}, nil}
	EnvironmentGroupCannotBeUpdated    = &Message{trad{FR: "Le groupe %s de l'environnement %s n'a pu être mis à jour : %s", EN: "Group %s on environment %s cannot be updated: %s"}, nil}
	EnvironmentGroupCreated            = &Message{trad{FR: "Le groupe %s de l'environnement %s a été ajouté", EN: "Group %s on environment %s has been added"}, nil}
	EnvironmentGroupCannotBeCreated    = &Message{trad{FR: "Le groupe %s de l'environnement %s n'a pu être ajouté : %s", EN: "Group %s on environment %s cannot be added: %s"}, nil}
)

//Message represent a struc format translated messages
type Message struct {
	Format trad
	Args   []interface{}
}

//New instanciantes a new message
func New(m *Message, args ...interface{}) Message {
	return Message{
		Format: m.Format,
		Args:   args,
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
