package main

import (
	"crypto/tls"
	"strings"

	"github.com/mattn/go-xmpp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	typeChat      = "chat"
	typeGroupChat = "groupchat"
)

func serverName(host string) string {
	return strings.Split(host, ":")[0]
}

func getNewXMPPClient() (*xmpp.Client, error) {
	xmpp.DefaultConfig = tls.Config{
		ServerName:         serverName(viper.GetString("xmpp_server")),
		InsecureSkipVerify: viper.GetBool("xmpp_insecure_skip_verify"),
	}

	options := xmpp.Options{Host: viper.GetString("xmpp_server"),
		User:          viper.GetString("xmpp_bot_jid"),
		Password:      viper.GetString("xmpp_bot_password"),
		NoTLS:         viper.GetBool("xmpp_notls"),
		StartTLS:      viper.GetBool("xmpp_starttls"),
		Debug:         viper.GetBool("xmpp_debug"),
		Session:       viper.GetBool("xmpp_session"),
		Status:        "",
		StatusMessage: "",
	}

	xmppClient, err := options.NewClient()

	if err != nil {
		log.Panicf("getClient >> NewClient XMPP, err:%s", err)
		return nil, err
	}

	err = sendInitialPresence(xmppClient)
	return xmppClient, err

}

// sendInitialPresence sends initial presence, describes here https://xmpp.org/rfcs/rfc3921.html#presence
// After establishing a session, a client SHOULD send initial presence to the server in order to signal its availability
// for communications. As defined herein, the initial presence stanza (1) MUST possess no 'to' address
// (signalling that it is meant to be broadcasted by the server on behalf of the client) and
// (2) MUST possess no 'type' attribute (signalling the user's availability).
func sendInitialPresence(xmppClient *xmpp.Client) error {
	log.Debugf("Sending initial Presence")
	presence := xmpp.Presence{From: viper.GetString("xmpp_bot_jid")}
	_, err := xmppClient.SendPresence(presence)
	if err != nil {
		log.Errorf("sendInitialPresence >> Error while sending initial presence")
	}

	return err
}
