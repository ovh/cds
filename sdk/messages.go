package sdk

import (
	"fmt"

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
	SpawnInfoHatcheryStarts             = &Message{1, trad{FR: "La Hatchery (%s) a démarré le lancement du worker avec le model %s", EN: "Hatchery (%s) starts spawn worker with model %s"}, nil}
	SpawnInfoHatcheryErrorSpawn         = &Message{2, trad{FR: "Une erreur est survenue lorsque la Hatchery (%s) a démarré un worker avec le model %s après %s, err:%s", EN: "Error while Hatchery (%s) spawn worker with model %s after %s, err:%s"}, nil}
	SpawnInfoHatcheryStartsSuccessfully = &Message{3, trad{FR: "La Hatchery (%s) a démarré le worker avec succès en %s", EN: "Hatchery (%s) spawn worker successfully in %s"}, nil}
	SpawnInfoWorkerEnd                  = &Message{4, trad{FR: "Le worker %s a terminé et a passé %s à travaillé sur les étapes", EN: "Worker %s finished working on this job and took %s to work on the steps"}, nil}
	SpawnInfoJobTaken                   = &Message{5, trad{FR: "Le job a été prise par le worker %s", EN: "Job was taen by worker %s"}, nil}
	SpawnInfoWorkerForJob               = &Message{6, trad{FR: "Ce worker a été créé pour lancer ce job", EN: "This worker was created to take this action"}, nil}
)

// Messages contains all sdk Messages
var Messages = map[int]*Message{
	SpawnInfoHatcheryStarts.ID:             SpawnInfoHatcheryStarts,
	SpawnInfoHatcheryErrorSpawn.ID:         SpawnInfoHatcheryErrorSpawn,
	SpawnInfoHatcheryStartsSuccessfully.ID: SpawnInfoHatcheryStartsSuccessfully,
	SpawnInfoWorkerEnd.ID:                  SpawnInfoWorkerEnd,
	SpawnInfoJobTaken.ID:                   SpawnInfoJobTaken,
	SpawnInfoWorkerForJob.ID:               SpawnInfoWorkerForJob,
}

//Message represent a struc format translated messages
type Message struct {
	ID     int
	Format trad
	Args   []interface{}
}

//NewMessage instanciantes a new message
func NewMessage(m *Message, args ...interface{}) Message {
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
