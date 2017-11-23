package template

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/templateextension"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// ApplyTemplate creates an application and configure it with given template
func ApplyTemplate(db *gorp.DbMap, store cache.Store, proj *sdk.Project, opts sdk.ApplyTemplatesOptions, user *sdk.User, sessionKey sessionstore.SessionKey, apiURL string) ([]sdk.Message, error) {
	var app *sdk.Application
	if opts.TemplateName == templateextension.EmptyTemplate.Name {
		app = &sdk.Application{
			Name:              opts.ApplicationName,
			ApplicationGroups: proj.ProjectGroups,
		}
	} else {
		//Get the template
		sdktmpl, errl := templateextension.LoadByName(db, opts.TemplateName)
		if errl != nil {
			return nil, errl
		}

		// Get the go-plugin instance
		templ, deferFunc, erri := templateextension.Instance(sdktmpl, user, sessionKey, apiURL)
		if deferFunc != nil {
			defer deferFunc()
		}
		if erri != nil {
			log.Warning("ApplyTemplate> error getting template Extension instance : %s", erri)
			return nil, erri
		}

		// Apply the template
		var erra error
		app, erra = templateextension.Apply(db, store, templ, proj, opts.TemplateParams, opts.ApplicationName)
		if erra != nil {
			log.Warning("ApplyTemplate> error applying template : %s", erra)
			return nil, erra
		}

		deferFunc()
	}

	//Start a new transaction
	tx, errb := db.Begin()
	if errb != nil {
		log.Warning("ApplyTemplate> error beginning transaction : %s", errb)
		return nil, errb
	}

	defer tx.Rollback()

	// Import the application
	done := make(chan bool)
	msgChan := make(chan sdk.Message)
	msgList := []sdk.Message{}
	go func(array *[]sdk.Message) {
		for {
			m, more := <-msgChan
			if !more {
				done <- true
				return
			}
			*array = append(*array, m)
		}
	}(&msgList)

	if err := application.Import(tx, store, proj, app, app.VCSServer, user, msgChan); err != nil {
		log.Warning("ApplyTemplate> error applying template : %s", err)
		close(msgChan)
		return msgList, err
	}

	if errProj := project.UpdateLastModified(tx, store, user, proj, sdk.ProjectApplicationLastModificationType); errProj != nil {
		log.Warning("ApplyTemplate> cannot update project last modified date : %s", errProj)
		close(msgChan)
		return msgList, errProj
	}

	close(msgChan)
	<-done

	log.Debug("ApplyTemplate> Commit the transaction")
	if err := tx.Commit(); err != nil {
		log.Warning("ApplyTemplate> error commiting transaction : %s", err)
		return msgList, err
	}

	log.Debug("ApplyTemplate> Done")

	return msgList, nil
}

// ApplyTemplateOnApplication configure an application it with given template
func ApplyTemplateOnApplication(db *gorp.DbMap, store cache.Store, proj *sdk.Project, app *sdk.Application, opts sdk.ApplyTemplatesOptions, user *sdk.User, sessionKey sessionstore.SessionKey, apiURL string) ([]sdk.Message, error) {
	//Get the template
	sdktmpl, err := templateextension.LoadByName(db, opts.TemplateName)
	if err != nil {
		return nil, err
	}

	// Get the go-plugin instance
	templ, deferFunc, err := templateextension.Instance(sdktmpl, user, sessionKey, apiURL)
	if deferFunc != nil {
		defer deferFunc()
	}
	if err != nil {
		log.Warning("ApplyTemplateOnApplication> error getting template Extension instance : %s", err)
		return nil, err
	}

	// Apply the template
	appTempl, err := templateextension.Apply(db, store, templ, proj, opts.TemplateParams, opts.ApplicationName)
	if err != nil {
		log.Warning("ApplyTemplateOnApplication> error applying template : %s", err)
		return nil, err
	}

	//Add the templated pipelines on the application
	app.Pipelines = append(app.Pipelines, appTempl.Pipelines...)

	//Start a new transaction
	tx, err := db.Begin()
	if err != nil {
		log.Warning("ApplyTemplateOnApplication> error beginning transaction : %s", err)
		return nil, err
	}

	defer tx.Rollback()

	done := make(chan bool)
	msgChan := make(chan sdk.Message)
	msgList := []sdk.Message{}
	go func(array *[]sdk.Message) {
		for {
			m, more := <-msgChan
			if !more {
				done <- true
				return
			}
			*array = append(*array, m)
		}
	}(&msgList)

	//Import the pipelines
	if err := application.ImportPipelines(tx, store, proj, app, user, msgChan); err != nil {
		log.Warning("ApplyTemplateOnApplication> error applying template : %s", err)
		close(msgChan)
		return msgList, err
	}

	close(msgChan)
	<-done

	log.Debug("ApplyTemplateOnApplication> Commit the transaction")
	if err := tx.Commit(); err != nil {
		log.Warning("ApplyTemplateOnApplication> error commiting transaction : %s", err)
		return msgList, err
	}

	deferFunc()
	log.Debug("ApplyTemplateOnApplication> Done")

	return msgList, nil
}
