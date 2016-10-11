package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func initRequest(req *http.Request) {
	req.Header.Set("CDS-notifs-key", notifsKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "close")
}

const (
	retrySleepSeconds = 5
	nbRetryMax        = 3
)

var notifsKey string
var notifsSystems map[string]string
var notifsSystemStatus map[string]string
var notifON bool
var baseURL string

const (
	cds2stash = "stash"
	cds2tat   = "tat"
	cds2xmpp  = "jabber"
)

func getPath(system string, notifType sdk.NotifType) string {
	if system == cds2xmpp {
		return "/jabber/build"
	}
	if notifType == sdk.ActionBuildNotif {
		return "/action/build"
	}
	if notifType == sdk.BuiltinNotif {
		return "/action/builtin"
	}
	return "/pipeline/build"

}

// Initialize initializes Notifications System
func Initialize(urls, key, base string) {
	notifsKey = key
	tmpNotifsURLs := strings.Split(urls, ",")
	baseURL = base
	notifsSystems = map[string]string{}
	notifsSystemStatus = map[string]string{}

	if key == "" || urls == "" {
		log.Critical("notification.Initialize> Invalid Configuration : invalid URL. See flags --notifs-key and --notifs-urls\n")
	} else {
		for _, url := range tmpNotifsURLs {
			t := strings.SplitN(url, ":", 2)
			if len(t) != 2 {
				log.Critical("notification.Initialize> Invalid format of notifsURLs %s\n", url)
				continue
			}

			switch t[0] {
			case cds2stash, cds2tat, cds2xmpp:
				log.Notice("notification.Initialize> Notifications System %s is enabled on %s\n", t[0], t[1])
				notifsSystems[t[0]] = t[1]
				notifsSystemStatus[t[0]] = "Unknown"
			default:
				log.Critical("notification.Initialize> Invalid Notifications System %s\n", url)
			}
		}

		notifON = true
		go statusChecker()
		go storeCleaner()
	}
}

func statusChecker() {
	for {
		for system, url := range notifsSystems {
			switch system {
			case cds2stash, cds2tat, cds2xmpp:
				notifSystemCheck(url+"/connectivity", system)
			}

		}
		time.Sleep(1 * time.Minute)
	}
}

func notifSystemCheck(path, system string) {
	_innerPost(path, []byte{}, func(resp *http.Response, err error) {
		if err != nil {
			notifsSystemStatus[system] = fmt.Sprintf(" KO (error: %s)", err)
		} else {
			if resp.StatusCode == http.StatusOK {
				notifsSystemStatus[system] = "OK"
			} else {
				notifsSystemStatus[system] = fmt.Sprintf("KO (http code: %d)", resp.StatusCode)
			}
		}
	})
}

//Status is used by Status Handler
func Status() []string {
	ret := []string{}
	for k, v := range notifsSystemStatus {
		ret = append(ret, "Notif "+k+": "+v)
	}
	return ret
}

func post(notif *sdk.Notif) {
	db := database.DB()
	if db == nil {
		return
	}
	var jsonStr []byte

	jsonStr, err := json.Marshal(notif)
	if err != nil {
		log.Critical("notification.post> error while marshalling json for a user notification", err.Error())
		return
	}

	var sent bool
	if !notifON {
		return
	}
	//notifsURLs is set on startup
	for system, url := range notifsSystems {
		//Only actionBuild, pipelineBuild and Builtin may be sent to tat.
		//Only pipeline notif may be sent to stash
		//Only user notif may be sent to jabber
		if (notif.NotifType != sdk.UserNotif && system == cds2tat) ||
			(notif.NotifType == sdk.PipelineBuildNotif && system == cds2stash) ||
			(notif.NotifType == sdk.UserNotif && system == cds2xmpp) {
			if notif.NotifType == sdk.UserNotif {
				if err := Insert(db, notif, system); err != nil {
					log.Critical("notification.post> error while inserting user notification in DB", err.Error())
				}

			}
			path := getPath(system, notif.NotifType)
			_innerPost(url+path, jsonStr, func(resp *http.Response, err error) {
				if notif.NotifType == sdk.UserNotif {
					if err != nil {
						if err := Update(db, notif, "ERROR : "+err.Error()); err != nil {
							log.Warning("notification.post> update failed : %s", err)
						}
					} else {
						if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest {
							if err := Update(db, notif, "SUCCESS"); err != nil {
								log.Warning("notification.post> update failed : %s", err)
							}
						} else {
							if err := Update(db, notif, "ERROR : "+resp.Status); err != nil {
								log.Warning("notification.post> update failed : %s", err)
							}
						}
					}
				}
			})
			sent = true
		}
	}
	if !sent {
		log.Critical("notification.post> %s can't be send", notif.NotifType)
	}
}

func getHTTPClient() *http.Client {
	tr := &http.Transport{}

	timeout := time.Duration(120 * time.Second)
	return &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}
}

func _innerPost(requestPath string, jsonStr []byte, callback func(*http.Response, error)) {
	var ntry, codeStatus int
	var lastErr error

	for ntry < nbRetryMax && (codeStatus < http.StatusOK || codeStatus >= http.StatusBadRequest) {
		log.Debug(string(jsonStr))
		req, err := http.NewRequest("POST", requestPath, bytes.NewReader(jsonStr))
		if err != nil {
			log.Warning("notification._innerPost> Error with http.NewRequest %s", err.Error())
			if callback != nil {
				callback(nil, err)
			}
			return
		}

		initRequest(req)
		ntry++

		resp, err := getHTTPClient().Do(req)
		if err != nil {
			log.Warning("notification._innerPost> Error http.Client.Do %s, it's try %d, new try", err.Error(), ntry)
			time.Sleep(time.Duration(retrySleepSeconds) * time.Second)
			lastErr = err
			continue
		}
		codeStatus = resp.StatusCode

		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest {
			body, _ := ioutil.ReadAll(resp.Body)
			log.Info("notification._innerPost> Response From notification system :%s", string(body))
			if callback != nil {
				callback(resp, nil)
			}
			return
		}
		body, _ := ioutil.ReadAll(resp.Body)
		logtxt := fmt.Sprintf("notification> Response Status:%s", resp.Status)
		logtxt += fmt.Sprintf(" Request path:%s", requestPath)
		logtxt += fmt.Sprintf(" Request:%s", string(jsonStr))
		logtxt += fmt.Sprintf(" Response Headers:%s", resp.Header)
		logtxt += fmt.Sprintf(" Response Body:%s", string(body))

		if nbRetryMax == ntry {
			log.Critical(logtxt)
			if callback != nil {
				callback(resp, nil)
			}
			return
		}
		log.Notice(logtxt)
		log.Warning("notification._innerPost> Error resp.StatusCode %d, it's try %d", resp.StatusCode, ntry)
		time.Sleep(time.Duration(retrySleepSeconds) * time.Second)
	}
	log.Critical("Notification> %s", lastErr)
	if callback != nil {
		callback(nil, lastErr)
	}
}

// SendMailNotif Send user notification by mail
func SendMailNotif(notif *sdk.Notif) {
	db := database.DB()
	if db == nil {
		return
	}
	Insert(db, notif, "email")
	log.Notice("notification.SendMailNotif> Send notif '%s'", notif.Title)
	errors := []string{}
	for _, recipient := range notif.Recipients {
		if err := mail.SendEmail(notif.Title, bytes.NewBufferString(notif.Message), recipient); err != nil {
			errors = append(errors, err.Error())
		}
	}
	if len(errors) > 0 {
		Update(db, notif, "ERROR : "+strings.Join(errors, ", "))
	} else {
		Update(db, notif, "SUCCESS")
	}

}
