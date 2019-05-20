package workflowtemplate

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/sdk"
)

// Push creates or updates a workflow template from a tar.
func Push(db gorp.SqlExecutor, wt *sdk.WorkflowTemplate, u sdk.Identifiable) ([]sdk.Message, error) {
	// check if a template already exists for group with same slug
	old, err := GetBySlugAndGroupIDs(db, wt.Slug, []int64{wt.Group.ID})
	if err != nil {
		return nil, err
	}
	if old == nil {
		if err := Insert(db, wt); err != nil {
			return nil, err
		}
		event.PublishWorkflowTemplateAdd(*wt, u)

		return []sdk.Message{sdk.NewMessage(sdk.MsgWorkflowTemplateImportedInserted, wt.Group.Name, wt.Slug)}, nil
	}

	new := sdk.WorkflowTemplate(*old)
	new.Update(*wt)

	// execute template with no instance only to check if parsing is ok
	if _, err := Execute(&new, nil); err != nil {
		return nil, err
	}

	if err := Update(db, &new); err != nil {
		return nil, err
	}

	event.PublishWorkflowTemplateUpdate(*old, new, "", u)

	return []sdk.Message{sdk.NewMessage(sdk.MsgWorkflowTemplateImportedUpdated, wt.Group.Name, new.Slug)}, nil
}
