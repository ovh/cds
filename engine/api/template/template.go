package template

import (
	"database/sql"
	"fmt"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/hook"
	"github.com/ovh/cds/engine/api/poller"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// Generic template variable
const (
	RepoURL string = "repo"
)

// ApplyTemplate creates an application and configure it with given templates
func ApplyTemplate(tx *sql.Tx, p *sdk.Project, app *sdk.Application) error {

	// Create application, add its variables and add project's group access
	err := application.CreateFromWizard(tx, app, p)
	if err != nil {
		return fmt.Errorf("ApplyTemplate> %s", err)
	}

	// Apply build template
	buildPipeline, err := applyBuildTemplate(tx, p, app)
	if err != nil {
		log.Warning("ApplyTemplate> %s", err)
		return err
	}

	if app.BuildTemplate.ID != UglyID && app.RepositoryFullname != "" && app.RepositoriesManager != nil {
		if app.RepositoriesManager.HooksSupported {
			_, err := hook.CreateHook(tx, p.Key, app.RepositoriesManager, app.RepositoryFullname, app, buildPipeline)
			if err != nil {
				log.Warning("ApplyTemplate> %s", err)
				return err
			}
		} else if app.RepositoriesManager.PollingSupported {
			err := poller.InsertPoller(tx, &sdk.RepositoryPoller{
				Application: *app,
				Pipeline:    *buildPipeline,
				Enabled:     true,
				Name:        app.RepositoriesManager.Name,
			})
			if err != nil {
				log.Warning("ApplyTemplate> %s", err)
				return err
			}
		}
	}

	// Apply deployment template
	err = applyDeployTemplate(tx, p, buildPipeline, app)
	if err != nil {
		log.Warning("ApplyTemplate> %s", err)
		return err
	}

	return nil
}
