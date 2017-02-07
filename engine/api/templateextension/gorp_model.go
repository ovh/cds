package templateextension

import (
	"encoding/json"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

//TemplateExtension is a gorp wrapper around sdk.TemplateExtension
type TemplateExtension sdk.TemplateExtension

//PostInsert is a DB Hook on TemplateExtension to store params as JSON in DB
func (t *TemplateExtension) PostInsert(s gorp.SqlExecutor) error {
	btes, err := json.Marshal(t.Params)
	if err != nil {
		return err
	}

	query := "insert into template_params (template_id, params) values ($1, $2)"
	if _, err := s.Exec(query, t.ID, btes); err != nil {
		return err
	}

	for _, a := range t.Actions {
		query := `insert into template_action (template_id, action_id) values ($1, 
					(
						select id from action where name = $2 and public = true limit 1
					))`
		if _, err := s.Exec(query, t.ID, a); err != nil {

			return err
		}
	}

	return nil
}

//PostUpdate is a DB Hook on TemplateExtension to store params as JSON in DB
func (t *TemplateExtension) PostUpdate(s gorp.SqlExecutor) error {
	btes, err := json.Marshal(t.Params)
	if err != nil {
		return err
	}

	query := "update template_params set params = $2 where template_id = $1"
	if _, err := s.Exec(query, t.ID, btes); err != nil {
		return err
	}
	query = "delete from template_action where template_id = $1"
	if _, err := s.Exec(query, t.ID); err != nil {
		return err
	}
	for _, a := range t.Actions {
		query := `insert into template_action (template_id, action_id) values ($1, 
					(
						select id from action where name = $2 and public = true limit 1
					))`
		if _, err := s.Exec(query, t.ID, a); err != nil {
			return err
		}
	}
	return nil
}

//PreDelete is a DB Hook on TemplateExtension to store params as JSON in DB
func (t *TemplateExtension) PreDelete(s gorp.SqlExecutor) error {
	query := "delete from template_params where template_id = $1"
	if _, err := s.Exec(query, t.ID); err != nil {
		return err
	}
	query = "delete from template_action where template_id = $1"
	if _, err := s.Exec(query, t.ID); err != nil {
		return err
	}
	return nil
}

func init() {
	gorpmapping.Register(gorpmapping.New(TemplateExtension{}, "template", true, "id"))
}
