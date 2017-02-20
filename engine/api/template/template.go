package template

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/msg"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/templateextension"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// ApplyTemplate creates an application and configure it with given template
func ApplyTemplate(db *gorp.DbMap, proj *sdk.Project, opts sdk.ApplyTemplatesOptions, user *sdk.User, sessionKey sessionstore.SessionKey) ([]msg.Message, error) {
	var app *sdk.Application
	var err error

	if opts.TemplateName == templateextension.EmptyTemplate.Name {
		app = &sdk.Application{
			Name:              opts.ApplicationName,
			ApplicationGroups: proj.ProjectGroups,
		}

	} else {
		//Get the template
		sdktmpl, err := templateextension.LoadByName(db, opts.TemplateName)
		if err != nil {
			return nil, err
		}

		// Get the go-plugin instance
		templ, deferFunc, err := templateextension.Instance(sdktmpl, user, sessionKey)
		if deferFunc != nil {
			defer deferFunc()
		}
		if err != nil {
			log.Warning("ApplyTemplate> error getting template Extension instance : %s", err)
			return nil, err
		}

		// Apply the template
		app, err = templateextension.Apply(db, templ, proj, opts.TemplateParams, opts.ApplicationName)
		if err != nil {
			log.Warning("ApplyTemplate> error applying template : %s", err)
			return nil, err
		}

		deferFunc()
	}

	//Start a new transaction
	tx, err := db.Begin()
	if err != nil {
		log.Warning("ApplyTemplate> error beginning transaction : %s", err)
		return nil, err
	}

	defer tx.Rollback()

	// Import the application
	done := make(chan bool)
	msgChan := make(chan msg.Message)
	msgList := []msg.Message{}
	go func(array *[]msg.Message) {
		for {
			m, more := <-msgChan
			if !more {
				done <- true
				return
			}
			*array = append(*array, m)
		}
	}(&msgList)

	if err := application.Import(tx, proj, app, app.RepositoriesManager, user, msgChan); err != nil {
		log.Warning("ApplyTemplate> error applying template : %s", err)
		close(msgChan)
		return msgList, err
	}

	lastModified, errProj := project.UpdateProjectDB(tx, proj.Key, proj.Name)
	if errProj != nil {
		log.Warning("ApplyTemplate> cannot update project last modified date : %s", errProj)
		close(msgChan)
		return msgList, errProj
	}
	proj.LastModified = lastModified

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
func ApplyTemplateOnApplication(db *gorp.DbMap, proj *sdk.Project, app *sdk.Application, opts sdk.ApplyTemplatesOptions, user *sdk.User, sessionKey sessionstore.SessionKey) ([]msg.Message, error) {
	//Get the template
	sdktmpl, err := templateextension.LoadByName(db, opts.TemplateName)
	if err != nil {
		return nil, err
	}

	// Get the go-plugin instance
	templ, deferFunc, err := templateextension.Instance(sdktmpl, user, sessionKey)
	if deferFunc != nil {
		defer deferFunc()
	}
	if err != nil {
		log.Warning("ApplyTemplateOnApplication> error getting template Extension instance : %s", err)
		return nil, err
	}

	// Apply the template
	appTempl, err := templateextension.Apply(db, templ, proj, opts.TemplateParams, opts.ApplicationName)
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
	msgChan := make(chan msg.Message)
	msgList := []msg.Message{}
	go func(array *[]msg.Message) {
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
	if err := application.ImportPipelines(tx, proj, app, msgChan); err != nil {
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
