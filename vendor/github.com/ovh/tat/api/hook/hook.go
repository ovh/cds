package hook

import (
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/ovh/tat"
	topicDB "github.com/ovh/tat/api/topic"
	"github.com/spf13/viper"
)

// InitHooks initializes hooks
func InitHooks() {
	initKafka()
	initWebhook()
	initXMPPHook()
}

// CloseHooks closes hooks
func CloseHooks() {
	closeKafka()
}

// SendHook sends a hook if topic contains hook parameter
func SendHook(hook *tat.HookJSON, topic tat.Topic) {
	go innerSendHook(hook, topic)
}

// GetCapabilities returns tat capabilities about hooks
func GetCapabilities() []tat.CapabilitieHook {
	hooks := []tat.CapabilitieHook{
		{HookType: tat.HookTypeKafka, HookEnabled: hookKafkaEnabled},
		{HookType: tat.HookTypeWebHook, HookEnabled: hookWebhookEnabled},
		{HookType: tat.HookTypeXMPP, HookEnabled: hookXMPPEnabled},
	}
	return hooks
}

func innerSendHook(hook *tat.HookJSON, topic tat.Topic) {
	innerSendHookTopicParameters(hook, topic)
	innerSendHookTopicFilters(hook, topic)
}

func innerSendHookTopicParameters(hook *tat.HookJSON, topic tat.Topic) {
	for _, p := range topic.Parameters {
		if !tat.ArrayContains(tat.HooksType, p.Key) {
			continue
		}
		h := &tat.HookJSON{
			HookMessage: hook.HookMessage,
			Hook: tat.Hook{
				Action:      "all",
				Type:        p.Key,
				Destination: p.Value,
				Enabled:     true,
			},
		}
		runHook(h, nil, topic)
	}
}

func runHook(h *tat.HookJSON, f *tat.Filter, topic tat.Topic) {
	log.Debugf("runHook enter f:%+v", f)
	if !h.Hook.Enabled {
		log.Debugf("Hook not enabled on topic %s h:%+v", topic.Topic, h)
		return
	}

	if f != nil {
		if h.Hook.Errors > viper.GetInt("hooks_max_errors") {
			log.Warnf("Max errors reached on hook %s for topic %s", h.Hook.ID, topic.Topic)
			for _, fh := range f.Hooks {
				fh.Enabled = false
			}
			topicDB.UpdateFilter(&topic, f)
			return
		}
		h.Username = f.Username
	}

	if h.Hook.Action != "all" && h.Hook.Action != h.HookMessage.Action {
		log.Debugf("Skip action:%s", h.Hook.Action, h)
		return
	}

	if strings.HasPrefix(h.Hook.Type, tat.HookTypeWebHook) {
		if err := sendWebHook(h, h.Hook.Destination, topic, "", ""); err != nil {
			log.Errorf("sendHook webhook err:%s", err)
		}
	} else if strings.HasPrefix(h.Hook.Type, tat.HookTypeKafka) {
		log.Infof("sendOnKafkaTopic")
		if err := sendOnKafkaTopic(h, h.Hook.Destination, topic); err != nil {
			log.Errorf("sendHook kafka err:%s", err)
		}
	} else if h.Hook.Type == tat.HookTypeXMPP || h.Hook.Type == tat.HookTypeXMPPOut {
		if err := sendXMPP(h, h.Hook.Destination, topic); err != nil {
			log.Errorf("sendHook XMPP err:%s", err)
		}
	}
}

func innerSendHookTopicFilters(h *tat.HookJSON, topic tat.Topic) {
	log.Debugf("topic.Filters:%+v", topic.Filters)
	for _, f := range topic.Filters {
		if matchCriteria(h.HookMessage.MessageJSONOut.Message, f.Criteria) {
			for _, hh := range f.Hooks {
				hbis := &tat.HookJSON{
					HookMessage: h.HookMessage,
					Hook: tat.Hook{
						Action:      hh.Action,
						Type:        hh.Type,
						Destination: hh.Destination,
						Enabled:     hh.Enabled,
					},
				}
				runHook(hbis, &f, topic)
			}
		}
	}
}

func matchCriteria(m tat.Message, c tat.FilterCriteria) bool {
	/*
		bson:"label" json:"label,omitempty
		bson:"notLabel" json:"notLabel,omitempty
		bson:"andLabel" json:"andLabel,omitempty
		bson:"tag" json:"tag,omitempty
		bson:"notTag" json:"notTag,omitempty
		bson:"andTag" json:"andTag,omitempty
		bson:"username" json:"username,omitempty
		bson:"onlyMsgRoot" json:"onlyMsgRoot,omitempty
	*/

	if c.OnlyMsgRoot && m.InReplyOfID != "" {
		return false
	}

	if c.Label != "" {
		labels := strings.Split(c.Label, ",")
		ok := false
		for _, l := range labels {
			if m.ContainsLabel(l) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	if c.NotLabel != "" {
		notLabels := strings.Split(c.NotLabel, ",")
		for _, l := range notLabels {
			if m.ContainsLabel(l) {
				return false
			}
		}
	}

	if c.AndLabel != "" {
		andLabels := strings.Split(c.AndLabel, ",")
		ok := 0
		for _, l := range andLabels {
			if m.ContainsLabel(l) {
				ok++
			}
		}
		if ok != len(andLabels) {
			return false
		}
	}

	if c.Tag != "" {
		tags := strings.Split(c.Tag, ",")
		ok := false
		for _, l := range tags {
			if m.ContainsTag(l) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	if c.NotTag != "" {
		notTags := strings.Split(c.NotTag, ",")
		for _, l := range notTags {
			if m.ContainsTag(l) {
				return false
			}
		}
	}

	if c.AndTag != "" {
		andTags := strings.Split(c.AndTag, ",")
		ok := 0
		for _, l := range andTags {
			if m.ContainsTag(l) {
				ok++
			}
		}
		if ok != len(andTags) {
			return false
		}
	}

	if c.Username != "" && c.Username != m.Author.Username {
		return false
	}

	return true
}
