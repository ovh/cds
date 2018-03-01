package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/mattn/go-xmpp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	cdsbot    *botClient
	xmlRegexp = regexp.MustCompile(`<\/[\w\-][^><]*>|<[\w\-][^><]*\/>`)
)

const resource = "cds"

type botClient struct {
	creation               time.Time
	XMPPClient             *xmpp.Client
	admins                 []string
	nbXMPPErrors           int
	nbXMPPErrorsAfterRetry int
	nbXMPPSent             int
	nbXMPPAnswers          int
	nbRenew                int
	chats                  chan xmpp.Chat
}

func getBotClient() (*botClient, error) {
	xClient, err := getNewXMPPClient()
	if err != nil {
		log.Errorf("getClient >> error with getNewXMPPClient err:%s", err)
		return nil, err
	}

	instance := &botClient{
		XMPPClient: xClient,
		admins:     strings.Split(viper.GetString("admin_cds2xmpp"), ","),
	}

	log.Infof("admin configured:%+v", viper.GetString("admin_cds2xmpp"))

	return instance, nil
}

func (bot *botClient) born() {
	bot.creation = time.Now().UTC()
	rand.Seed(time.Now().Unix())

	if viper.GetString("admin_conference") != "" {
		conferences = append(conferences, viper.GetString("admin_conference"))
	}

	bot.chats = make(chan xmpp.Chat)
	go bot.sendToXMPP()

	bot.helloWorld()

	go bot.receive()
	go do()

	for {
		sendInitialPresence(bot.XMPPClient)
		time.Sleep(10 * time.Second)
		bot.sendPresencesOnConfs()
		time.Sleep(20 * time.Second)
	}
}

func (bot *botClient) helloWorld() {
	for _, a := range bot.admins {
		log.Infof("helloWorld >> sending hello world to %s", a)

		bot.chats <- xmpp.Chat{
			Remote: a,
			Type:   "chat",
			Text:   fmt.Sprintf("Hi, I'm CDS2XMPP, what a good day to be alive. /cds cds2xmpp status for more information"),
		}
	}
}

const status = `
CDS2XMPP Status

Started:{{.creation}} since {{.since}}
Admin: {{.admin}}

XMPP:
- sent: {{.sent}}, errors: {{.nbXMPPErrors}}, errors after retry: {{.nbXMPPErrorsAfterRetry}}
- renew: {{.nbRenew}}

----
Bot:
- answers: {{.nbXMPPAnswers}}

`

func (bot *botClient) getStatus() string {

	data := map[string]string{
		"creation":               fmt.Sprintf("%s", cdsbot.creation),
		"since":                  fmt.Sprintf("%s", time.Now().Sub(cdsbot.creation)),
		"admin":                  viper.GetString("admin_cds2xmpp"),
		"sent":                   fmt.Sprintf("%d", bot.nbXMPPSent),
		"nbXMPPErrors":           fmt.Sprintf("%d", bot.nbXMPPErrors),
		"nbXMPPErrorsAfterRetry": fmt.Sprintf("%d", bot.nbXMPPErrorsAfterRetry),
		"nbRenew":                fmt.Sprintf("%d", bot.nbRenew),
		"nbXMPPAnswers":          fmt.Sprintf("%d", bot.nbXMPPAnswers),
	}

	t, errp := template.New("status").Parse(status)
	if errp != nil {
		log.Errorf("getStatus> Error:%s", errp.Error())
		return "Error while prepare status:" + errp.Error()
	}

	var buffer bytes.Buffer
	if err := t.Execute(&buffer, data); err != nil {
		log.Errorf("getStatus> Error:%s", errp.Error())
		return "Error while prepare status (execute):" + err.Error()
	}

	return buffer.String()
}

func (bot *botClient) sendPresencesOnConfs() error {
	bot.nbRenew++
	for _, c := range conferences {
		bot.XMPPClient.JoinMUCNoHistory(c, resource)
	}
	return nil
}

func (bot *botClient) sendToXMPP() {
	for {
		chat := <-bot.chats
		if isXML(chat.Text) {
			cdsbot.XMPPClient.SendHtml(chat)
		} else {
			cdsbot.XMPPClient.Send(chat)
		}
		time.Sleep(time.Duration(viper.GetInt("xmpp_delay")) * time.Second)
	}
}

// XML is detected if presence of tags like </foo> or <foo/>
// This means that <br> is not detected as XML, but <br/> is
func isXML(text string) bool {
	return len(xmlRegexp.FindAllString(text, -1)) > 0
}

func (bot *botClient) receive() {
	for {
		chat, err := bot.XMPPClient.Recv()
		if err != nil {
			if !strings.Contains(err.Error(), "EOF") {
				log.Errorf("receive >> err: %s", err)
			}
		}
		isError := false
		switch v := chat.(type) {
		case xmpp.Chat:
			if v.Remote != "" {
				if v.Type == "error" {

					isError = true
					log.Errorf("receive> msg error from xmpp :%+v\n", v)

					if !strings.HasSuffix(v.Text, " [cds2xmppRetry]") {
						bot.nbXMPPErrors++
						go cdsbot.sendRetry(v)
					} else {
						bot.nbXMPPErrorsAfterRetry++
					}
				} else {
					log.Debugf("receive> msg from xmpp :%+v\n", v)
				}
			}

			if !isError {
				bot.receiveMsg(v)
			}
		}
	}
}

func (bot *botClient) sendRetry(v xmpp.Chat) {
	time.Sleep(60 * time.Second)
	bot.chats <- xmpp.Chat{
		Remote: v.Remote,
		Type:   getTypeChat(v.Remote),
		Text:   v.Text + " [cds2xmppRetry]",
	}
}

func getTypeChat(s string) string {
	if strings.Contains(s, "@conference.") {
		return typeGroupChat
	}
	return typeChat
}

func (bot *botClient) receiveMsg(chat xmpp.Chat) {
	log.Debugf("receiveMsg >> enter remote:%s text:%s", chat.Remote, chat.Text)
	if time.Now().Add(-10*time.Second).Unix() < bot.creation.Unix() {
		log.Debugf("receiveMsg >> exit, bot is starting... ")
		return
	}

	if strings.HasPrefix(chat.Text, "cds, ") || strings.HasPrefix(chat.Text, "/cds ") {
		log.Infof("receiveMsg for cdsbot >> %s from remote:%s stamp:%s", chat.Text, chat.Remote, chat.Stamp)
		bot.answer(chat)
	}

}
