package application

import (
	"regexp"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// TriggerPipeline linked to received hook
func TriggerPipeline(tx gorp.SqlExecutor, store cache.Store, h sdk.Hook, branch string, hash string, author string, p *sdk.Pipeline, projectData *sdk.Project) (*sdk.PipelineBuild, error) {

	// Create pipeline args
	var args []sdk.Parameter
	args = append(args, sdk.Parameter{
		Name:  "git.branch",
		Value: branch,
	})
	args = append(args, sdk.Parameter{
		Name:  "git.hash",
		Value: hash,
	})
	args = append(args, sdk.Parameter{
		Name:  "git.author",
		Value: author,
	})
	args = append(args, sdk.Parameter{
		Name:  "git.repository",
		Value: h.Repository,
	})
	args = append(args, sdk.Parameter{
		Name:  "git.project",
		Value: h.Project,
	})

	// Load pipeline Argument
	parameters, err := pipeline.GetAllParametersInPipeline(tx, p.ID)
	if err != nil {
		return nil, err
	}
	p.Parameter = parameters

	// get application
	a, err := LoadByID(tx, store, h.ApplicationID, nil, LoadOptions.WithRepositoryManager, LoadOptions.WithVariablesWithClearPassword)
	if err != nil {
		return nil, err
	}
	applicationPipelineArgs, err := GetAllPipelineParam(tx, h.ApplicationID, p.ID)
	if err != nil {
		return nil, err
	}

	trigger := sdk.PipelineBuildTrigger{
		ManualTrigger:    false,
		VCSChangesBranch: branch,
		VCSChangesHash:   hash,
		VCSChangesAuthor: author,
	}

	// Get commit message to check if we have to skip the build
	if a.RepositoriesManager != nil {
		if b, _ := repositoriesmanager.CheckApplicationIsAttached(tx, a.RepositoriesManager.Name, projectData.Key, a.Name); b && a.RepositoryFullname != "" {
			//Get the RepositoriesManager Client (the last args are useless to get commit)
			client, _ := repositoriesmanager.AuthorizedClient(tx, projectData.Key, a.RepositoriesManager.Name, store)
			if client != nil {
				commit, err := client.Commit(a.RepositoryFullname, hash)
				if err != nil {
					log.Warning("hook> can't get commit %s from %s on %s : %s", hash, a.RepositoryFullname, a.RepositoriesManager.Name, err)
				}
				match, err := regexp.Match(".*\\[ci skip\\].*|.*\\[cd skip\\].*", []byte(commit.Message))
				if err != nil {
					log.Warning("hook> Cannot check %s/%s for commit %s by %s : %s (%s)", projectData.Key, a.Name, hash, author, commit.Message, err)
				}
				if match {
					log.Info("hook> Skipping build of %s/%s for commit %s by %s", projectData.Key, a.Name, hash, author)
					return nil, nil
				}
			}
		} else {
			log.Debug("Application is not attached (%s %s %s)", a.RepositoriesManager.Name, projectData.Key, a.Name)
		}
	}

	// FIXME add possibility to trigger a pipeline on a specific env
	pb, errpb := pipeline.InsertPipelineBuild(tx, projectData, p, a, applicationPipelineArgs, args, &sdk.DefaultEnv, 0, trigger)
	if errpb != nil {
		return nil, sdk.WrapError(errpb, "hook> Unable to insert pipeline build")
	}

	return pb, nil
}
