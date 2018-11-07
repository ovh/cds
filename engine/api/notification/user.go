package notification

import (
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	uiURL string
)

// Init initializes notification package
func Init(uiurl string) {
	uiURL = uiurl
}

// GetUserWorkflowEvents return events to send for the given workflow run
func GetUserWorkflowEvents(db gorp.SqlExecutor, w sdk.Workflow, previousWR *sdk.WorkflowNodeRun, nr sdk.WorkflowNodeRun) []sdk.EventNotif {
	events := []sdk.EventNotif{}

	//Compute notification
	params := map[string]string{}
	for _, p := range nr.BuildParameters {
		params[p.Name] = p.Value
	}
	//Set PipelineBuild UI URL
	params["cds.buildURL"] = fmt.Sprintf("%s/project/%s/workflow/%s/run/%d", uiURL, w.ProjectKey, w.Name, nr.Number)
	if p, ok := params["cds.triggered_by.email"]; ok {
		params["cds.author.email"] = p
	} else if p, ok := params["git.author.email"]; ok {
		params["cds.author.email"] = p
	}
	if p, ok := params["cds.triggered_by.username"]; ok {
		params["cds.author"] = p
	} else if p, ok := params["git.author"]; ok {
		params["cds.author"] = p
	}
	params["cds.status"] = nr.Status

	for _, notif := range w.Notifications {
		if ShouldSendUserWorkflowNotification(notif, nr, previousWR) {
			switch notif.Type {
			case sdk.JabberUserNotification:
				jn := &notif.Settings
				//Get recipents from groups
				if jn.SendToGroups {
					u, errPerm := projectPermissionUsers(db, w.ProjectID, permission.PermissionRead)
					if errPerm != nil {
						log.Error("notification[Jabber]. error while loading permission:%s", errPerm.Error())
					}
					for i := range u {
						jn.Recipients = append(jn.Recipients, u[i].Username)
					}
				}
				if jn.SendToAuthor {
					if author, ok := params["cds.author"]; ok {
						jn.Recipients = append(jn.Recipients, author)
					}
				}

				//Finally deduplicate everyone
				removeDuplicates(&jn.Recipients)
				events = append(events, getWorkflowEvent(jn, params))
			case sdk.EmailUserNotification:
				jn := &notif.Settings
				//Get recipents from groups
				if jn.SendToGroups {
					u, errPerm := projectPermissionUsers(db, w.ProjectID, permission.PermissionRead)
					if errPerm != nil {
						log.Error("notification[Email].SendPipelineBuild> error while loading permission:%s", errPerm.Error())
						return nil
					}
					for i := range u {
						jn.Recipients = append(jn.Recipients, u[i].Email)
					}
				}
				if jn.SendToAuthor {
					if email, ok := params["cds.author.email"]; ok {
						jn.Recipients = append(jn.Recipients, email)
					} else if author, okA := params["cds.author"]; okA && author != "" {
						u, err := user.LoadUserWithoutAuth(db, author)
						if err != nil {
							log.Warning("notification[Email].SendPipelineBuild> Cannot load author %s: %s", author, err)
							continue
						}
						jn.Recipients = append(jn.Recipients, u.Email)
					}
				}
				//Finally deduplicate everyone
				removeDuplicates(&jn.Recipients)
				go SendMailNotif(getWorkflowEvent(jn, params))
			}
		}
	}
	return events
}

// ShouldSendUserWorkflowNotification test if the notificationhas to be sent for the given workflow node run
func ShouldSendUserWorkflowNotification(notif sdk.WorkflowNotification, nodeRun sdk.WorkflowNodeRun, previousNodeRun *sdk.WorkflowNodeRun) bool {
	var check = func(s string) bool {
		switch s {
		case sdk.UserNotificationAlways:
			return true
		case sdk.UserNotificationNever:
			return false
		case sdk.UserNotificationChange:
			if previousNodeRun == nil || previousNodeRun.ID == 0 {
				return true
			}
			return previousNodeRun.Status != nodeRun.Status
		}
		return false
	}

	var found bool
	for _, n := range notif.SourceNodeIDs {
		if n == nodeRun.WorkflowNodeID {
			found = true
			break
		}
	}
	if !found {
		return false
	}

	switch nodeRun.Status {
	case sdk.StatusSuccess.String():
		if check(notif.Settings.OnSuccess) {
			return true
		}
	case sdk.StatusFail.String():
		if check(notif.Settings.OnFailure) {
			return true
		}
	case sdk.StatusWaiting.String():
		return notif.Settings.OnStart
	}

	return false
}

func getWorkflowEvent(notif *sdk.UserNotificationSettings, params map[string]string) sdk.EventNotif {
	subject := notif.Template.Subject
	body := notif.Template.Body
	for k, value := range params {
		key := "{{." + k + "}}"
		subject = strings.Replace(subject, key, value, -1)
		body = strings.Replace(body, key, value, -1)
	}

	e := sdk.EventNotif{
		Subject: subject,
		Body:    body,
	}
	for _, r := range notif.Recipients {
		e.Recipients = append(e.Recipients, r)
	}

	return e
}

func removeDuplicates(xs *[]string) {
	found := make(map[string]bool)
	j := 0
	for i, x := range *xs {
		if !found[x] {
			found[x] = true
			(*xs)[j] = (*xs)[i]
			j++
		}
	}
	*xs = (*xs)[:j]
}
