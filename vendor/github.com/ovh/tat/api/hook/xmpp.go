package hook

import (
	log "github.com/Sirupsen/logrus"
	"github.com/ovh/tat"
	"github.com/spf13/viper"
)

var hookXMPPEnabled bool

func initXMPPHook() {
	hookXMPPEnabled = viper.GetString("tat2xmpp_url") != ""
}

func sendXMPP(hook *tat.HookJSON, path string, topic tat.Topic) error {
	if hook.HookMessage.MessageJSONOut.Message.Author.Username == viper.GetString("tat2xmpp_username") {
		log.Debugf("sendXMPP: Skip msg from %s on topic %s", viper.GetString("tat2xmpp_username"), topic.Topic)
		return nil
	}
	log.Debugf("sendXMPP: enter for post XMPP via tat2XMPP setted on topic %s", topic.Topic)

	return sendWebHook(hook, viper.GetString("tat2xmpp_url")+"/hook", topic, tat.HookTat2XMPPHeaderKey, viper.GetString("tat2xmpp_key"))
}
