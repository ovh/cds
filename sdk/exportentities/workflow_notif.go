package exportentities

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Booleans
var (
	True  = true
	False = false
)

// NotificationEntry represents a notification set on a nodeEntry
type NotificationEntry struct {
	Type     string                        `json:"type" yaml:"type"`
	Settings *sdk.UserNotificationSettings `json:"settings,omitempty" yaml:"settings,omitempty"`
}

// craftNotificationEntry returns the NotificationEntry and the name of the nodeEntries concerned
func craftNotificationEntry(w sdk.Workflow, notif sdk.WorkflowNotification) ([]string, NotificationEntry, error) {
	entry := NotificationEntry{}
	nodeNames := make([]string, len(notif.SourceNodeRefs))
	for i, ref := range notif.SourceNodeRefs {
		node := w.WorkflowData.NodeByName(ref)
		if node == nil {
			log.Error("unable to find workflow node %s", ref)
			return nil, entry, sdk.ErrWorkflowNodeNotFound
		}
		nodeNames[i] = node.Name
	}
	sort.Strings(nodeNames)
	entry.Type = notif.Type
	entry.Settings = &notif.Settings

	// Replace the default values by nil
	if entry.Settings.OnStart != nil && !*entry.Settings.OnStart {
		entry.Settings.OnStart = nil
	}
	if entry.Settings.SendToGroups != nil && !*entry.Settings.SendToGroups {
		entry.Settings.SendToGroups = nil
	}
	if entry.Settings.SendToAuthor != nil && *entry.Settings.SendToAuthor {
		entry.Settings.SendToAuthor = nil
	}
	// Replace the default values by empty strings
	if entry.Settings.OnSuccess == sdk.UserNotificationChange {
		entry.Settings.OnSuccess = ""
	}
	if entry.Settings.OnFailure == sdk.UserNotificationAlways {
		entry.Settings.OnFailure = ""
	}
	// Replace default templates by nil if they are default values
	if entry.Settings.Template != nil {
		defaultTemplate, has := sdk.UserNotificationTemplateMap[entry.Type]
		if !has {
			return nil, entry, fmt.Errorf("workflow notification %s not found", entry.Type)
		}
		if defaultTemplate.Subject == entry.Settings.Template.Subject {
			entry.Settings.Template.Subject = ""
		}
		if defaultTemplate.Body == entry.Settings.Template.Body {
			entry.Settings.Template.Body = ""
		}
		if entry.Settings.Template.Body == "" && entry.Settings.Template.Subject == "" {
			entry.Settings.Template = nil
		}
	}

	// Finally if settings are all default, lets skip it
	if entry.Settings.OnFailure == "" &&
		entry.Settings.OnStart == nil &&
		entry.Settings.OnSuccess == "" &&
		len(entry.Settings.Recipients) == 0 &&
		entry.Settings.SendToAuthor == nil &&
		entry.Settings.SendToGroups == nil &&
		entry.Settings.Template == nil {
		entry.Settings = nil
	}

	return nodeNames, entry, nil
}

func craftNotifications(w sdk.Workflow, exportedWorkflow *Workflow) error {
	for _, notif := range w.Notifications {
		nodeNames, notifEntry, err := craftNotificationEntry(w, notif)
		if err != nil {
			return sdk.WrapError(err, "unable to craft notification")
		}
		// If it's a single pipeline workflow, the pipelineName is set
		if exportedWorkflow.PipelineName != "" {
			exportedWorkflow.Notifications = append(exportedWorkflow.Notifications, notifEntry)
		} else {
			if exportedWorkflow.MapNotifications == nil {
				exportedWorkflow.MapNotifications = make(map[string][]NotificationEntry)
			}
			s := strings.Join(nodeNames, ",")
			exportedWorkflow.MapNotifications[s] = append(exportedWorkflow.MapNotifications[s], notifEntry)
		}
	}
	return nil
}

func checkWorkflowNotificationsValidity(w Workflow) error {
	mError := new(sdk.MultiError)
	if len(w.Workflow) != 0 {
		if len(w.Notifications) != 0 {
			mError.Append(fmt.Errorf("Error: wrong usage: notify not allowed here"))
		}
	} else {
		if len(w.MapNotifications) > 0 {
			mError.Append(fmt.Errorf("Error: wrong usage: notifications not allowed here"))
		}
	}

	for nodeNames := range w.MapNotifications {
		names := strings.Split(nodeNames, ",")
		for _, s := range names {
			name := strings.TrimSpace(s)
			if _, ok := w.Workflow[name]; !ok {
				mError.Append(fmt.Errorf("Error: wrong usage: invalid notification on %s (%s is missing)", nodeNames, name))
			}
		}
	}
	if len(*mError) == 0 {
		return nil
	}
	return mError
}

func processNotificationValues(notif NotificationEntry) (sdk.WorkflowNotification, error) {
	n := sdk.WorkflowNotification{
		Type: notif.Type,
	}
	defaultTemplate, has := sdk.UserNotificationTemplateMap[n.Type]
	//Check the type
	if !has {
		return n, fmt.Errorf("Error: wrong usage: invalid notification type %s", n.Type)
	}
	//Default settings
	if notif.Settings == nil {
		n.Settings = sdk.UserNotificationSettings{
			OnFailure:    sdk.UserNotificationAlways,
			OnSuccess:    sdk.UserNotificationChange,
			OnStart:      &False,
			SendToAuthor: &True,
			SendToGroups: &False,
			Template:     &defaultTemplate,
		}
	} else {
		n.Settings = *notif.Settings
	}
	//Default values
	if n.Settings.OnFailure == "" {
		n.Settings.OnFailure = sdk.UserNotificationAlways
	}
	if n.Settings.OnSuccess == "" {
		n.Settings.OnSuccess = sdk.UserNotificationChange
	}
	if n.Settings.OnStart == nil {
		n.Settings.OnStart = &False
	}
	if n.Settings.SendToAuthor == nil {
		n.Settings.SendToAuthor = &True
	}
	if n.Settings.SendToGroups == nil {
		n.Settings.SendToGroups = &False
	}
	if n.Settings.Template == nil {
		n.Settings.Template = &defaultTemplate
	} else {
		if n.Settings.Template.Subject == "" {
			n.Settings.Template.Subject = defaultTemplate.Subject
		}
		if n.Settings.Template.Body == "" {
			n.Settings.Template.Subject = defaultTemplate.Body
		}
	}
	return n, nil
}

func (w *Workflow) processNotifications(wrkflw *sdk.Workflow) error {
	// Multiple pipelines in the workflow
	if len(w.MapNotifications) > 0 {
		for nodeNames, notifs := range w.MapNotifications {
			nodes := strings.Split(nodeNames, ",")
			// nodes are considered as references of nodes
			for _, notif := range notifs {
				n, err := processNotificationValues(notif)
				if err != nil {
					return sdk.WrapError(err, "unable to process notification")
				}
				n.SourceNodeRefs = nodes
				wrkflw.Notifications = append(wrkflw.Notifications, n)
			}
		}
	} else {
		//single pipeline
		for _, notif := range w.Notifications {
			n, err := processNotificationValues(notif)
			if err != nil {
				return sdk.WrapError(err, "unable to process notification")
			}
			n.SourceNodeRefs = []string{wrkflw.WorkflowData.Node.Name}
			wrkflw.Notifications = append(wrkflw.Notifications, n)
		}
	}
	return nil
}
