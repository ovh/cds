package main

import (
	"crypto/tls"
	"fmt"
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
	hosts := strings.Split(viper.GetString("xmpp_server"), ",")

	var xmppClient *xmpp.Client
	for _, host := range hosts {
		log.Infof("Trying to connect on host %s", host)

		xmpp.DefaultConfig = tls.Config{
			ServerName:         serverName(host),
			InsecureSkipVerify: viper.GetBool("xmpp_insecure_skip_verify"),
		}

		options := xmpp.Options{
			Host:     host,
			User:     viper.GetString("xmpp_bot_jid"),
			Password: viper.GetString("xmpp_bot_password"),
			NoTLS:    viper.GetBool("xmpp_notls"),
			StartTLS: viper.GetBool("xmpp_starttls"),
			Debug:    viper.GetBool("xmpp_debug"),
			Session:  viper.GetBool("xmpp_session"),
		}

		var errNewClient error
		xmppClient, errNewClient = options.NewClient()

		if errNewClient != nil {
			log.Errorf("getClient >> NewClient XMPP, err with host %s:%s", host, errNewClient)
			continue
		}

		// If we are here, that means that the connection has been successful and we can stop iterating over hosts and use the current host
		break
	}

	if xmppClient == nil {
		return nil, fmt.Errorf("connection failed with all hosts (%v)", hosts)
	}

	return xmppClient, sendInitialPresence(xmppClient)
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
