package workflow

import (
	"sync"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

// ParseAndImport parse an exportentities.Application and insert or update the application in database
func ParseAndImport(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, ew *exportentities.Workflow, force bool, u *sdk.User) ([]sdk.Message, error) {
	log.Info("ParseAndImport>> Import workflow %s in project %s (force=%v)", ew.Name, proj.Key, force)
	log.Debug("ParseAndImport>> Workflow: %+v", ew)

	//Check valid application name
	rx := sdk.NamePatternRegex
	if !rx.MatchString(ew.Name) {
		return nil, sdk.WrapError(sdk.ErrInvalidApplicationPattern, "ParseAndImport>> Workflow name %s do not respect pattern %s", ew.Name, sdk.NamePattern)
	}

	w := new(sdk.Workflow)
	w.Name = ew.Name
	w.ProjectID = proj.ID
	w.ProjectKey = proj.Key

	//Inherit permissions from project
	if len(ew.Permissions) == 0 {
		ew.Permissions = make(map[string]int)
		for _, p := range proj.ProjectGroups {
			ew.Permissions[p.Group.Name] = p.Permission
		}
	}

	//Compute permissions
	for g, p := range ew.Permissions {
		perm := sdk.GroupPermission{Group: sdk.Group{Name: g}, Permission: p}
		w.Groups = append(w.Groups, perm)
	}

	//Parse workflow
	w, errW := ew.GetWorkflow()
	if errW != nil {
		return nil, sdk.WrapError(errW, "ParseAndImport> Workflow parsing error")
	}

	//Import
	done := new(sync.WaitGroup)
	done.Add(1)
	msgChan := make(chan sdk.Message)
	msgList := []sdk.Message{}
	go func(array *[]sdk.Message) {
		defer done.Done()
		for {
			m, more := <-msgChan
			if !more {
				return
			}
			*array = append(*array, m)
		}
	}(&msgList)

	globalError := Import(db, store, proj, w, u, force, msgChan)
	close(msgChan)
	done.Wait()

	return msgList, globalError
}
