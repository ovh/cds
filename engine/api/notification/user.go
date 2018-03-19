package notification

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	apiURL string
	uiURL  string
)

// Init initializes notification package
func Init(apiurl, uiurl string) {
	apiURL = apiurl
	uiURL = uiurl
}

// GetUserEvents returns event from user notification
func GetUserEvents(db gorp.SqlExecutor, pb *sdk.PipelineBuild, previous *sdk.PipelineBuild) []sdk.EventNotif {
	//Load notif
	userNotifs, errLoad := LoadUserNotificationSettings(db, pb.Application.ID, pb.Pipeline.ID, pb.Environment.ID)
	if errLoad != nil {
		log.Error("notification.GetUserEvents> error while loading user notification settings: %s", errLoad)
		return nil
	}
	if userNotifs == nil {
		log.Debug("notification.GetUserEvents> no user notification on pipeline %d, app %d, env %d", pb.Application.ID, pb.Pipeline.ID, pb.Environment.ID)
		return nil
	}

	//Compute notification
	params := map[string]string{}
	for _, p := range pb.Parameters {
		params[p.Name] = p.Value
	}
	params["cds.status"] = pb.Status.String()
	//Set PipelineBuild UI URL
	params["cds.buildURL"] = fmt.Sprintf("%s/project/%s/application/%s/pipeline/%s/build/%d?envName=%s", uiURL, pb.Pipeline.ProjectKey, pb.Application.Name, pb.Pipeline.Name, pb.BuildNumber, url.QueryEscape(pb.Environment.Name))
	//find author (triggeredBy user or changes author)
	if pb.Trigger.TriggeredBy != nil {
		params["cds.author"] = pb.Trigger.TriggeredBy.Username
	} else if pb.Trigger.VCSChangesAuthor != "" {
		params["cds.author"] = pb.Trigger.VCSChangesAuthor
	}

	events := []sdk.EventNotif{}

	for t, notif := range userNotifs.Notifications {
		if ShouldSendUserNotification(notif, pb, previous) {
			switch t {
			case sdk.JabberUserNotification:
				jn, ok := notif.(*sdk.JabberEmailUserNotificationSettings)
				if !ok {
					log.Error("notification.GetUserEvents> cannot deal with %s", notif)
					continue
				}
				//Get recipents from groups
				if jn.SendToGroups {
					u, errPerm := applicationPipelineEnvironmentUsers(db, pb.Application.ID, pb.Pipeline.ID, pb.Environment.ID, permission.PermissionRead)
					if errPerm != nil {
						log.Error("notification[Jabber].SendPipelineBuild> error while loading permission:%s", errPerm.Error())
					}
					for i := range u {
						jn.Recipients = append(jn.Recipients, u[i].Username)
					}
				}
				//Get recipents from groups
				if jn.SendToGroups {
					u, errEnv := applicationPipelineEnvironmentUsers(db, pb.Application.ID, pb.Pipeline.ID, pb.Environment.ID, permission.PermissionRead)
					if errEnv != nil {
						log.Error("notification[Jabber].SendPipelineBuild> error while loading permission:%s", errEnv.Error())
					}
					for i := range u {
						jn.Recipients = append(jn.Recipients, u[i].Username)
					}
				}
				if jn.SendToAuthor {
					//find author (triggeredBy user or changes author)
					if pb.Trigger.TriggeredBy != nil {
						jn.Recipients = append(jn.Recipients, pb.Trigger.TriggeredBy.Username)
					} else if pb.Trigger.VCSChangesAuthor != "" {
						jn.Recipients = append(jn.Recipients, pb.Trigger.VCSChangesAuthor)
					}
				}
				//Finally deduplicate everyone
				removeDuplicates(&jn.Recipients)
				events = append(events, getEvent(pb, jn, params))
			case sdk.EmailUserNotification:
				jn, ok := notif.(*sdk.JabberEmailUserNotificationSettings)
				if !ok {
					log.Error("notification.GetUserEvents> cannot deal with %s", notif)
					continue
				}
				//Get recipents from groups
				if jn.SendToGroups {
					u, errEnv := applicationPipelineEnvironmentUsers(db, pb.Application.ID, pb.Pipeline.ID, pb.Environment.ID, permission.PermissionRead)
					if errEnv != nil {
						log.Error("notification[Email].SendPipelineBuild> error while loading permission:%s", errEnv.Error())
						return nil
					}
					for i := range u {
						jn.Recipients = append(jn.Recipients, u[i].Email)
					}
				}
				if jn.SendToAuthor {
					var username string
					if pb.Trigger.TriggeredBy != nil {
						username = pb.Trigger.TriggeredBy.Username
					} else if pb.Trigger.VCSChangesAuthor != "" {
						username = pb.Trigger.VCSChangesAuthor
					}
					if username != "" {
						u, err := user.LoadUserWithoutAuth(db, username)
						if err != nil {
							log.Warning("notification[Email].SendPipelineBuild> Cannot load author %s: %s", username, err)
							continue
						}
						jn.Recipients = append(jn.Recipients, u.Email)
					}
				}
				//Finally deduplicate everyone
				removeDuplicates(&jn.Recipients)
				go SendMailNotif(getEvent(pb, jn, params))
			}
		}
	}
	return events
}

// GetUserWorkflowEvents return events to send for the given workflow run
func GetUserWorkflowEvents(db gorp.SqlExecutor, wr sdk.WorkflowRun, previousWR sdk.WorkflowNodeRun, nr sdk.WorkflowNodeRun) []sdk.EventNotif {
	events := []sdk.EventNotif{}

	//Compute notification
	params := map[string]string{}
	for _, p := range nr.BuildParameters {
		params[p.Name] = p.Value
	}
	//Set PipelineBuild UI URL
	params["cds.buildURL"] = fmt.Sprintf("%s/project/%s/workflow/%s/run/%d", uiURL, wr.Workflow.ProjectKey, wr.Workflow.Name, wr.Number)
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

	for _, notif := range wr.Workflow.Notifications {
		if ShouldSendUserWorkflowNotification(notif, nr, previousWR) {
			switch notif.Type {
			case sdk.JabberUserNotification:
				jn, ok := notif.Settings.(*sdk.JabberEmailUserNotificationSettings)
				if !ok {
					log.Error("notification.GetUserWorkflowEvents[Jabber]> cannot deal with %s", notif)
					continue
				}
				//Get recipents from groups
				if jn.SendToGroups {
					u, errPerm := projectPermissionUsers(db, wr.Workflow.ProjectID, permission.PermissionRead)
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
				jn, ok := notif.Settings.(*sdk.JabberEmailUserNotificationSettings)
				if !ok {
					log.Error("notification.GetUserEvents[Email]> cannot deal with %s", notif)
					continue
				}
				//Get recipents from groups
				if jn.SendToGroups {
					u, errPerm := projectPermissionUsers(db, wr.Workflow.ProjectID, permission.PermissionRead)
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
func ShouldSendUserWorkflowNotification(notif sdk.WorkflowNotification, nodeRun sdk.WorkflowNodeRun, previousNodeRun sdk.WorkflowNodeRun) bool {
	var check = func(s sdk.UserNotificationEventType) bool {
		switch s {
		case sdk.UserNotificationAlways:
			return true
		case sdk.UserNotificationNever:
			return false
		case sdk.UserNotificationChange:
			if previousNodeRun.ID == 0 {
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
		if check(notif.Settings.Success()) {
			return true
		}
	case sdk.StatusFail.String():
		if check(notif.Settings.Failure()) {
			return true
		}
	case sdk.StatusWaiting.String():
		return notif.Settings.Start()
	}

	return false
}

func getWorkflowEvent(notif *sdk.JabberEmailUserNotificationSettings, params map[string]string) sdk.EventNotif {
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

//ShouldSendUserNotification check if user notification has to be sent
func ShouldSendUserNotification(notif sdk.UserNotificationSettings, current *sdk.PipelineBuild, previous *sdk.PipelineBuild) bool {
	var check = func(s sdk.UserNotificationEventType) bool {
		switch s {
		case sdk.UserNotificationAlways:
			return true
		case sdk.UserNotificationNever:
			return false
		case sdk.UserNotificationChange:
			if previous == nil {
				return true
			}
			return current.Status.String() != previous.Status.String()
		}
		return false
	}
	switch current.Status {
	case sdk.StatusSuccess:
		if check(notif.Success()) {
			return true
		}
	case sdk.StatusFail:
		if check(notif.Failure()) {
			return true
		}
	case sdk.StatusBuilding:
		return notif.Start()
	}
	return false
}

func getEvent(pb *sdk.PipelineBuild, notif *sdk.JabberEmailUserNotificationSettings, params map[string]string) sdk.EventNotif {
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

//UserNotificationInput is a way to parse notification
type UserNotificationInput struct {
	Notifications         map[string]interface{} `json:"notifications"`
	ApplicationPipelineID int64                  `json:"application_pipeline_id"`
	Environment           sdk.Environment        `json:"environment"`
	Pipeline              sdk.Pipeline           `json:"pipeline"`
}

//LoadAllUserNotificationSettingsByProject load data for a project
func LoadAllUserNotificationSettingsByProject(db gorp.SqlExecutor, projectKey string, u *sdk.User) ([]sdk.UserNotification, error) {
	n := []sdk.UserNotification{}

	var query string
	var args []interface{}
	//Handler admin
	if u == nil || u.Admin {
		query = `SELECT 	application_pipeline_id, environment_id, project.projectkey, settings, pipeline.id, pipeline.name, environment.name
		FROM  	application_pipeline_notif
		JOIN 	application_pipeline ON application_pipeline.id = application_pipeline_notif.application_pipeline_id
		JOIN 	pipeline ON pipeline.id = application_pipeline.pipeline_id
		JOIN 	environment ON environment.id = environment_id
		JOIN 	project ON project.id = pipeline.project_id
		WHERE 	project.projectkey = $1
		ORDER BY pipeline.name`
		args = []interface{}{projectKey}
	} else {
		query = `
		SELECT 	application_pipeline_id, environment_id, project.projectkey, settings, pipeline.id, pipeline.name, environment.name
		FROM  	application_pipeline_notif
		JOIN 	application_pipeline ON application_pipeline.id = application_pipeline_notif.application_pipeline_id
		JOIN 	pipeline ON pipeline.id = application_pipeline.pipeline_id
		JOIN 	environment ON environment.id = environment_id
		JOIN 	project ON project.id = pipeline.project_id
		JOIN	application_group ON application_pipeline.application_id = application_group.application_id
		JOIN 	pipeline_group ON pipeline.id = pipeline_group.pipeline_id
		JOIN 	group_user ON group_user.group_id = pipeline_group.group_id AND group_user.group_id = application_group.group_id
		WHERE 	project.projectkey = $1
		AND 	group_user.user_id = $2
		ORDER BY pipeline.name`
		args = []interface{}{projectKey, u.ID}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var un sdk.UserNotification
		var settings string
		if err := rows.Scan(&un.ApplicationPipelineID, &un.Environment.ID, &un.Environment.ProjectKey, &settings, &un.Pipeline.ID, &un.Pipeline.Name, &un.Environment.Name); err != nil {
			return nil, err
		}
		var err error
		un.Notifications, err = sdk.ParseUserNotificationSettings([]byte(settings))
		if err != nil {
			return nil, err
		}

		if u != nil {
			if !permission.AccessToEnvironment(un.Environment.ProjectKey, un.Environment.Name, u, permission.PermissionRead) {
				continue
			}
		}

		n = append(n, un)
	}

	return n, nil
}

//LoadAllUserNotificationSettings load data from application_pipeline_notif
func LoadAllUserNotificationSettings(db gorp.SqlExecutor, appID int64) ([]sdk.UserNotification, error) {
	n := []sdk.UserNotification{}
	query := `
		SELECT 	application_pipeline_id, environment_id, settings, pipeline.id, pipeline.name, environment.name
		FROM  	application_pipeline_notif
		JOIN 	application_pipeline ON application_pipeline.id = application_pipeline_notif.application_pipeline_id
		JOIN 	pipeline ON pipeline.id = application_pipeline.pipeline_id
		JOIN 	environment ON environment.id = environment_id
		WHERE 	application_pipeline.application_id = $1
		ORDER BY pipeline.name
	`

	rows, err := db.Query(query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var un sdk.UserNotification
		var settings string
		err := rows.Scan(&un.ApplicationPipelineID, &un.Environment.ID, &settings, &un.Pipeline.ID, &un.Pipeline.Name, &un.Environment.Name)
		if err != nil {
			return nil, err
		}
		un.Notifications, err = sdk.ParseUserNotificationSettings([]byte(settings))
		if err != nil {
			return nil, err
		}
		n = append(n, un)
	}

	return n, nil
}

//LoadUserNotificationSettings load data from application_pipeline_notif
func LoadUserNotificationSettings(db gorp.SqlExecutor, appID, pipID, envID int64) (*sdk.UserNotification, error) {
	var n = &sdk.UserNotification{}
	var settings string
	query := `
		SELECT 	application_pipeline_id, environment_id, settings, pipeline.id, pipeline.name, environment.name
		FROM  	application_pipeline_notif
		JOIN 	application_pipeline ON application_pipeline.id = application_pipeline_notif.application_pipeline_id
		JOIN 	pipeline ON pipeline.id = application_pipeline.pipeline_id
		JOIN 	environment ON environment.id = environment_id
		WHERE 	application_pipeline.application_id = $1
		AND		application_pipeline.pipeline_id = $2
		AND 	application_pipeline_notif.environment_id = $3
		ORDER BY pipeline.name
	`

	if err := db.QueryRow(query, appID, pipID, envID).Scan(&n.ApplicationPipelineID, &n.Environment.ID, &settings,
		&n.Pipeline.ID, &n.Pipeline.Name, &n.Environment.Name); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Warning("notification.LoadUserNotificationSettings>1> %s", err)
		return nil, err
	}

	var err error
	n.Notifications, err = sdk.ParseUserNotificationSettings([]byte(settings))
	if err != nil {
		log.Warning("notification.LoadUserNotificationSettings>2> %s", err)
		return nil, err
	}

	return n, nil
}

// DeleteNotification Delete a notifications for the given application/pipeline/environment
func DeleteNotification(db gorp.SqlExecutor, appID, pipID, envID int64) error {
	query := `
		DELETE FROM application_pipeline_notif
		USING 	application_pipeline, application, pipeline
		WHERE application_pipeline.id = application_pipeline_notif.application_pipeline_id
		AND application_pipeline.application_id = $1
		AND application_pipeline.pipeline_id = $2
		AND application_pipeline_notif.environment_id = $3
	`
	_, err := db.Exec(query, appID, pipID, envID)
	return err
}

//InsertOrUpdateUserNotificationSettings insert or update value in application_pipeline_notif
func InsertOrUpdateUserNotificationSettings(db gorp.SqlExecutor, appID, pipID, envID int64, notif *sdk.UserNotification) error {
	query := `
		SELECT 	count(1)
		FROM  	application_pipeline_notif
		JOIN 	application_pipeline ON application_pipeline.id = application_pipeline_notif.application_pipeline_id
		WHERE 	application_pipeline.application_id = $1
		AND		application_pipeline.pipeline_id = $2
		AND 	application_pipeline_notif.environment_id = $3
	`

	var nb int
	if err := db.QueryRow(query, appID, pipID, envID).Scan(&nb); err != nil {
		log.Error("notification.InsertOrUpdateUserNotificationSettings> Error counting application_pipeline_notif %d %d %d: %s", appID, pipID, envID, err)
		return err
	}
	var appPipelineID int64
	if nb == 0 {
		query = `
			INSERT INTO application_pipeline_notif (application_pipeline_id, environment_id)
			VALUES (
				(
				SELECT 	application_pipeline.id
				FROM 	application_pipeline
				WHERE 	application_pipeline.application_id = $1
				AND		application_pipeline.pipeline_id = $2
				),$3
			)
			RETURNING application_pipeline_id
		`
		if err := db.QueryRow(query, appID, pipID, envID).Scan(&appPipelineID); err != nil {
			log.Error("notification.InsertOrUpdateUserNotificationSettings> Error inserting application_pipeline_notif %d %d %d: %s", appID, pipID, envID, err)
			return err
		}
	}

	if appPipelineID != 0 {
		notif.ApplicationPipelineID = appPipelineID
		notif.Environment.ID = envID
	}

	bytes, err := json.Marshal(notif.Notifications)
	if err != nil {
		log.Error("notification.InsertOrUpdateUserNotificationSettings> Error marshalling notifications settings: %s", err)
		return err
	}

	//Update settings
	query = `
		UPDATE 	application_pipeline_notif SET settings = $4
		FROM 	application_pipeline
		WHERE 	application_pipeline.application_id = $1
		AND		application_pipeline.pipeline_id = $2
		AND 	application_pipeline_notif.environment_id = $3
		AND 	application_pipeline.id = application_pipeline_notif.application_pipeline_id
	`
	res, err := db.Exec(query, appID, pipID, envID, string(bytes))
	if err != nil {
		log.Error("notification.InsertOrUpdateUserNotificationSettings> Error updating notifications settings %d %d %d: %s", appID, pipID, envID, err)
		return err
	}
	if i, _ := res.RowsAffected(); i != 1 {
		log.Error("notification.InsertOrUpdateUserNotificationSettings> Error updating notifications settings %d %d %d : %d rows updated", appID, pipID, envID, i)
		return fmt.Errorf("Error updating notifications settings %d %d %d : %d rows updated", appID, pipID, envID, i)
	}

	return nil
}
