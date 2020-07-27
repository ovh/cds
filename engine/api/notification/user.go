package notification

import (
	"context"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/interpolate"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/luascript"
)

var (
	uiURL string
)

// Init initializes notification package
func Init(uiurl string) {
	uiURL = uiurl
}

const (
	paramsAuthorEmail = "cds.author.email"
	paramsAuthorName  = "cds.author"
	paramsStatus      = "cds.status"
	paramsBuildURL    = "cds.buildURL"
)

// GetUserWorkflowEvents return events to send for the given workflow run
func GetUserWorkflowEvents(ctx context.Context, db gorp.SqlExecutor, store cache.Store, projectID int64, projectKey, workflowName string, notifs []sdk.WorkflowNotification, previousWR *sdk.WorkflowNodeRun, nr sdk.WorkflowNodeRun) []sdk.EventNotif {
	events := []sdk.EventNotif{}

	//Compute notification
	params := map[string]string{}
	for _, p := range nr.BuildParameters {
		params[p.Name] = p.Value
	}
	//Set PipelineBuild UI URL
	params[paramsBuildURL] = fmt.Sprintf("%s/project/%s/workflow/%s/run/%d", uiURL, projectKey, workflowName, nr.Number)
	if p, ok := params["cds.triggered_by.email"]; ok {
		params[paramsAuthorEmail] = p
	} else if p, ok := params["git.author.email"]; ok {
		params[paramsAuthorEmail] = p
	}
	if p, ok := params["cds.triggered_by.username"]; ok {
		params[paramsAuthorName] = p
	} else if p, ok := params["git.author"]; ok {
		params[paramsAuthorName] = p
	}
	params[paramsStatus] = nr.Status

	for _, notif := range notifs {
		if ShouldSendUserWorkflowNotification(ctx, notif, nr, previousWR) {
			switch notif.Type {
			case sdk.JabberUserNotification:
				jn := &notif.Settings
				//Get recipents from groups
				if jn.SendToGroups != nil && *jn.SendToGroups {
					u, err := projectPermissionUserIDs(ctx, db, store, projectID, sdk.PermissionRead)
					if err != nil {
						log.Error(ctx, "notification[Jabber]. error while loading permission: %v", err)
						break
					}
					users, err := user.LoadAllByIDs(ctx, db, u)
					if err != nil {
						log.Error(ctx, "notification[Jabber]. error while loading users: %v", err)
						break
					}
					for _, u := range users {
						jn.Recipients = append(jn.Recipients, u.Username)
					}
				}
				if jn.SendToAuthor == nil || *jn.SendToAuthor {
					if author, ok := params[paramsAuthorEmail]; ok {
						jn.Recipients = append(jn.Recipients, author)
					}
				}

				//Finally deduplicate everyone
				removeDuplicates(&jn.Recipients)
				notif, err := getWorkflowEvent(jn, params)
				if err != nil {
					log.Error(ctx, "notification.GetUserWorkflowEvents> unable to handle event %+v: %v", jn, err)
				}
				events = append(events, notif)

			case sdk.EmailUserNotification:
				jn := &notif.Settings
				//Get recipents from groups
				if jn.SendToGroups != nil && *jn.SendToGroups {
					u, err := projectPermissionUserIDs(ctx, db, store, projectID, sdk.PermissionRead)
					if err != nil {
						log.Error(ctx, "notification[Email].GetUserWorkflowEvents> error while loading permission: %v", err)
						return nil
					}
					contacts, err := user.LoadContactsByUserIDs(ctx, db, u)
					if err != nil {
						log.Error(ctx, "notification[Jabber]. error while loading users contacts: %v", err)
						break
					}
					for _, c := range contacts {
						if c.Type == sdk.UserContactTypeEmail {
							jn.Recipients = append(jn.Recipients, c.Value)
						}
					}
				}
				if jn.SendToAuthor == nil || *jn.SendToAuthor {
					if email, ok := params[paramsAuthorEmail]; ok {
						jn.Recipients = append(jn.Recipients, email)
					} else if author, okA := params[paramsAuthorName]; okA &&
						author != "" && author != sdk.SchedulerUsername {
						// Load the user
						au, err := user.LoadByUsername(ctx, db, author)
						if err != nil {
							log.Error(ctx, "notification[Email].GetUserWorkflowEvents> Cannot load author %s: %s", author, err)
							continue
						}
						jn.Recipients = append(jn.Recipients, au.GetEmail())
					}
				}
				//Finally deduplicate everyone
				removeDuplicates(&jn.Recipients)
				notif, err := getWorkflowEvent(jn, params)
				if err != nil {
					log.Error(ctx, "notification.GetUserWorkflowEvents> unable to handle event %+v: %v", jn, err)
				}
				log.Debug("GetUserWorkflowEvents> will send mail notifications: %+v", notif)
				go sendMailNotif(ctx, notif)
			}
		}
	}
	return events
}

// ShouldSendUserWorkflowNotification test if the notificationhas to be sent for the given workflow node run
func ShouldSendUserWorkflowNotification(ctx context.Context, notif sdk.WorkflowNotification, nodeRun sdk.WorkflowNodeRun, previousNodeRun *sdk.WorkflowNodeRun) bool {
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
	for _, n := range notif.SourceNodeRefs {
		if n == nodeRun.WorkflowNodeName {
			found = true
			break
		}
	}
	if !found {
		return false
	}

	switch nodeRun.Status {
	case sdk.StatusSuccess:
		if check(notif.Settings.OnSuccess) && checkConditions(ctx, notif.Settings.Conditions, nodeRun.BuildParameters) {
			return true
		}
	case sdk.StatusFail:
		if check(notif.Settings.OnFailure) && checkConditions(ctx, notif.Settings.Conditions, nodeRun.BuildParameters) {
			return true
		}
	case sdk.StatusWaiting:
		return notif.Settings.OnStart != nil && *notif.Settings.OnStart && checkConditions(ctx, notif.Settings.Conditions, nodeRun.BuildParameters)
	}

	return false
}

func checkConditions(ctx context.Context, conditions sdk.WorkflowNodeConditions, params []sdk.Parameter) bool {
	var conditionsOK bool
	var errc error
	if conditions.LuaScript == "" {
		conditionsOK, errc = sdk.WorkflowCheckConditions(conditions.PlainConditions, params)
	} else {
		luacheck, err := luascript.NewCheck()
		if err != nil {
			log.Error(ctx, "notification check condition error: %s", err)
			return false
		}
		luacheck.SetVariables(sdk.ParametersToMap(params))
		errc = luacheck.Perform(conditions.LuaScript)
		conditionsOK = luacheck.Result
	}
	if errc != nil {
		log.Error(ctx, "notification check condition error on execution: %s", errc)
		return false
	}
	return conditionsOK
}

func getWorkflowEvent(notif *sdk.UserNotificationSettings, params map[string]string) (sdk.EventNotif, error) {
	subject, err := interpolate.Do(notif.Template.Subject, params)
	if err != nil {
		return sdk.EventNotif{}, err
	}

	body, err := interpolate.Do(notif.Template.Body, params)
	if err != nil {
		return sdk.EventNotif{}, err
	}

	e := sdk.EventNotif{
		Subject: subject,
		Body:    body,
	}
	e.Recipients = append(e.Recipients, notif.Recipients...)

	return e, nil
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
