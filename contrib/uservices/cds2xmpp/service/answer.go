package main

import (
	"strings"

	"github.com/mattn/go-xmpp"
	"github.com/spf13/viper"

	"github.com/ovh/cds/sdk/bot"
)

func (xmppBot *botClient) answer(chat xmpp.Chat) {

	typeXMPP := getTypeChat(chat.Remote)
	remote := chat.Remote
	to := strings.Split(chat.Remote, "@")[0]
	if typeXMPP == "groupchat" {
		if strings.Contains(chat.Remote, "/") {
			t := strings.Split(chat.Remote, "/")
			remote = t[0]
			to = t[1]
		}
	}

	xmppBot.chats <- xmpp.Chat{
		Remote: remote,
		Type:   typeXMPP,
		Text:   to + ": " + xmppBot.prepareAnswer(chat.Text, chat.Remote),
	}
	xmppBot.nbXMPPAnswers++
}

func (xmppBot *botClient) prepareAnswer(text, remote string) string {
	question := strings.TrimSpace(text[5:]) // remove '/cds ' or 'cds, '

	switch question {
	case "help":
		return xmppBot.help()
	case "cds2xmpp status":
		if xmppBot.isAdmin(remote) {
			return xmppBot.getStatus()
		}
		return "forbidden for you " + remote
	default:
		return bot.Answer(question)
	}
}

func (xmppBot *botClient) help() string {
	out := `
Begin conversation with "cds," or "/cds"

Simple request: "cds, ping"

/cds cds2xmpp status (for admin only)

`

	return out + viper.GetString("more_help")
}

func (xmppBot *botClient) isAdmin(r string) bool {
	for _, a := range xmppBot.admins {
		if strings.HasPrefix(r, a) {
			return true
		}
	}
	return false
}
