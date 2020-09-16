package v2

import (
	"context"
	"sort"

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
	Type        string                        `json:"type" yaml:"type"`
	Pipelines   []string                      `json:"pipelines" yaml:"pipelines,omitempty"`
	Settings    *sdk.UserNotificationSettings `json:"settings,omitempty" yaml:"settings,omitempty"`
	Integration string                        `json:"integration,omitempty" yaml:"integration,omitempty"`
}

// craftNotificationEntry returns the NotificationEntry and the name of the nodeEntries concerned
func craftNotificationEntry(ctx context.Context, w sdk.Workflow, notif sdk.WorkflowNotification) (NotificationEntry, error) {
	entry := NotificationEntry{
		Pipelines: make([]string, len(notif.SourceNodeRefs)),
	}
	for i, ref := range notif.SourceNodeRefs {
		node := w.WorkflowData.NodeByName(ref)
		if node == nil {
			log.Error(ctx, "unable to find workflow node %s", ref)
			return entry, sdk.ErrWorkflowNodeNotFound
		}
		entry.Pipelines[i] = node.Name
	}
	sort.Strings(entry.Pipelines)
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
			return entry, sdk.NewErrorFrom(sdk.ErrWrongRequest, "workflow notification %s not found", entry.Type)
		}
		if defaultTemplate.Subject == entry.Settings.Template.Subject {
			entry.Settings.Template.Subject = ""
		}
		if defaultTemplate.Body == entry.Settings.Template.Body {
			entry.Settings.Template.Body = ""
		}
		if entry.Settings.Template.Body == "" && entry.Settings.Template.Subject == "" {
			if entry.Settings.Template.DisableComment == nil || !*entry.Settings.Template.DisableComment {
				entry.Settings.Template = nil
			}
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

	return entry, nil
}

func craftNotifications(ctx context.Context, w sdk.Workflow, exportedWorkflow *Workflow) error {
	for i, notif := range w.Notifications {
		notifEntry, err := craftNotificationEntry(ctx, w, notif)
		if err != nil {
			return sdk.WrapError(err, "unable to craft notification")
		}
		if exportedWorkflow.Notifications == nil {
			exportedWorkflow.Notifications = make([]NotificationEntry, len(w.Notifications))
		}
		exportedWorkflow.Notifications[i] = notifEntry
	}
	for _, e := range w.EventIntegrations {
		entry := NotificationEntry{
			Integration: e.Name,
			Type:        sdk.EventsNotification,
		}
		if exportedWorkflow.Notifications == nil {
			exportedWorkflow.Notifications = make([]NotificationEntry, len(w.Notifications))
		}
		exportedWorkflow.Notifications = append(exportedWorkflow.Notifications, entry)

	}
	return nil
}

func CheckWorkflowNotificationsValidity(w Workflow) error {
	mError := new(sdk.MultiError)
	for _, notifEntry := range w.Notifications {
		for _, s := range notifEntry.Pipelines {
			if _, ok := w.Workflow[s]; !ok {
				mError.Append(sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid notification on %s (%s is missing)", notifEntry.Pipelines, s))
			}
		}
	}
	if len(*mError) == 0 {
		return nil
	}
	return mError
}

func ProcessNotificationValues(notif NotificationEntry) (sdk.WorkflowNotification, error) {
	n := sdk.WorkflowNotification{
		Type: notif.Type,
	}
	defaultTemplate, has := sdk.UserNotificationTemplateMap[n.Type]
	//Check the type
	if !has {
		return n, sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid notification type %s", n.Type)
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
			n.Settings.Template.Body = defaultTemplate.Body
		}
		if n.Settings.Template.DisableComment == nil || !*n.Settings.Template.DisableComment {
			n.Settings.Template.DisableComment = nil
		}
	}
	return n, nil
}

func (w *Workflow) processNotifications(wrkflw *sdk.Workflow) error {
	for _, notif := range w.Notifications {
		if notif.Type == sdk.EventsNotification {
			if notif.Integration == "" {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "notification of type event must be linked to an integration")
			}
			wrkflw.EventIntegrations = append(wrkflw.EventIntegrations, sdk.ProjectIntegration{Name: notif.Integration})
			continue
		}
		n, err := ProcessNotificationValues(notif)
		if err != nil {
			return sdk.WrapError(err, "unable to process notification")
		}
		n.SourceNodeRefs = notif.Pipelines
		wrkflw.Notifications = append(wrkflw.Notifications, n)
	}
	return nil
}
