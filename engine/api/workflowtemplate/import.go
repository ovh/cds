package workflowtemplate

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/go-gorp/gorp"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// Push creates or updates a workflow template from a tar.
func Push(db gorp.SqlExecutor, u *sdk.User, tr *tar.Reader) ([]sdk.Message, *sdk.WorkflowTemplate, error) {
	// extract template data from tar
	var apps, pips, envs [][]byte
	var wkf []byte
	var tmpl exportentities.Template

	mError := new(sdk.MultiError)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, sdk.NewError(sdk.ErrWrongRequest, sdk.WrapError(err, "Unable to read tar file"))
		}

		buff := new(bytes.Buffer)
		if _, err := io.Copy(buff, tr); err != nil {
			return nil, nil, sdk.NewError(sdk.ErrWrongRequest, sdk.WrapError(err, "Unable to read tar file"))
		}

		var templateFileName string
		b := buff.Bytes()
		switch {
		case strings.Contains(hdr.Name, ".application."):
			apps = append(apps, b)
		case strings.Contains(hdr.Name, ".pipeline."):
			pips = append(pips, b)
		case strings.Contains(hdr.Name, ".environment."):
			envs = append(envs, b)
		case hdr.Name == "workflow.yml":
			// if a workflow was already found, it's a mistake
			if len(wkf) != 0 {
				mError.Append(fmt.Errorf("Two workflow files found"))
				break
			}
			wkf = b
		default:
			// if a template was already found, it's a mistake
			if templateFileName != "" {
				mError.Append(fmt.Errorf("Two template files found: %s and %s", templateFileName, hdr.Name))
				break
			}
			if err := yaml.Unmarshal(b, &tmpl); err != nil {
				mError.Append(sdk.WrapError(err, "Unable to unmarshal template %s", hdr.Name))
				continue
			}
			templateFileName = hdr.Name
		}
	}

	if !mError.IsEmpty() {
		return nil, nil, sdk.NewError(sdk.ErrWorkflowInvalid, mError)
	}

	// init workflow template struct from data
	wt := tmpl.GetTemplate(wkf, pips, apps, envs)

	// check the workflow template extracted
	if err := wt.ValidateStruct(); err != nil {
		return nil, nil, err
	}

	// check that the user is admin on the given template's group
	var group *sdk.Group
	for _, g := range u.Groups {
		if g.Name == wt.Group.Name {
			for _, a := range g.Admins {
				if a.ID == u.ID {
					group = &g
					break
				}
			}
			break
		}
	}
	if group == nil {
		return nil, nil, sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Invalid given group for template"))
	}
	wt.GroupID = group.ID

	// check if a template already exists for group with same slug
	old, err := Get(db, NewCriteria().Slugs(wt.Slug).GroupIDs(group.ID))
	if err != nil {
		return nil, nil, err
	}
	if old == nil {
		if err := Insert(db, &wt); err != nil {
			return nil, nil, err
		}
		event.PublishWorkflowTemplateAdd(wt, u)

		return []sdk.Message{sdk.NewMessage(sdk.MsgWorkflowTemplateImportedInserted, group.Name, wt.Slug)}, &wt, nil
	}

	new := sdk.WorkflowTemplate(*old)
	new.Update(wt)

	if err := Update(db, &new); err != nil {
		return nil, nil, err
	}

	event.PublishWorkflowTemplateUpdate(*old, new, u)

	return []sdk.Message{sdk.NewMessage(sdk.MsgWorkflowTemplateImportedUpdated, group.Name, new.Slug)}, &new, nil
}
