package workflowtemplate

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/sdk"
)

// Push creates or updates a workflow template from a tar.
func Push(db gorp.SqlExecutor, wt *sdk.WorkflowTemplate, u sdk.Identifiable) ([]sdk.Message, error) {
	// check if a template already exists for group with same slug
	old, err := LoadBySlugAndGroupID(db, wt.Slug, wt.GroupID, LoadOptions.Default)
	if err != nil {
		return nil, err
	}
	if old == nil {
		if err := Insert(db, wt); err != nil {
			return nil, err
		}

		newTemplate, err := LoadByID(db, wt.ID, LoadOptions.Default)
		if err != nil {
			return nil, err
		}

		event.PublishWorkflowTemplateAdd(*newTemplate, u)

		return []sdk.Message{sdk.NewMessage(sdk.MsgWorkflowTemplateImportedInserted, newTemplate.Group.Name, newTemplate.Slug)}, nil
	}

	clone := sdk.WorkflowTemplate(*old)
	clone.Update(*wt)

	// execute template with no instance only to check if parsing is ok
	if _, err := Execute(&clone, nil); err != nil {
		return nil, err
	}

	if err := Update(db, &clone); err != nil {
		return nil, err
	}

	newTemplate, err := LoadByID(db, clone.ID, LoadOptions.Default)
	if err != nil {
		return nil, err
	}

	event.PublishWorkflowTemplateUpdate(*old, *newTemplate, "", u)

	return []sdk.Message{sdk.NewMessage(sdk.MsgWorkflowTemplateImportedUpdated, newTemplate.Group.Name, newTemplate.Slug)}, nil
}
