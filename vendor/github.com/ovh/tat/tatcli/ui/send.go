package ui

import (
	"strings"

	"github.com/gizak/termui"
	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/ovh/tat/tatcli/message"
	"github.com/spf13/viper"
)

func (ui *tatui) initSend() {
	p := termui.NewPar("")
	p.Height = uiHeightSend
	p.BorderLeft = false
	p.BorderRight = false
	p.BorderTop = true
	p.BorderBottom = false
	p.TextFgColor = termui.ColorWhite
	p.BorderFg = termui.ColorCyan
	p.BorderLabel = " ✎ Action "
	ui.send = p
}

func (ui *tatui) autocomplete() {
	keys := []string{
		"/clear",
		"/codereview",
		"/favorites",
		"/filter",
		"/filter-topics",
		"/filter-messages",
		"/help",
		"/hide-top",
		"/hide-bottom",
		"/hide-usernames",
		"/label",
		"/label FFFFFF yourLabel",
		"/like",
		"/mode",
		"/monitoring",
		"/open",
		"/open-links",
		"/quit",
		"/run",
		"/set-tatwebui-url",
		"/save",
		"/split",
		"/task",
		"/topics",
		"/toggle-top",
		"/toggle-bottom",
		"/toggle-usernames",
		"/unlabel",
		"/unlabel yourLabel",
		"/unlike",
		"/untask",
		"/unread",
		"/unvoteup",
		"/unvotedown",
		"/version",
		"/voteup",
		"/votedown",
	}
	for _, h := range ui.hooks {
		keys = append(keys, h.Command)
	}
	if strings.HasPrefix(ui.send.Text, "/") {
		for _, key := range keys {
			if key == ui.send.Text {
				continue
			}
			if strings.HasPrefix(key, ui.send.Text) {
				ui.send.Text = key + " "
				ui.render()
				break
			}
		}
	}
}

func (ui *tatui) processMsg() {
	ui.history = append(ui.history, ui.send.Text)
	ui.currentPosHistory = len(ui.history)
	if strings.HasPrefix(ui.send.Text, "/") {
		ui.processCmd()
		return
	}
	if ui.current == uiMessage || ui.current == uiMessages {
		ui.sendMsg()
	}
}

func (ui *tatui) processCmd() {
	switch ui.send.Text {
	case "/clear":
		ui.send.Text = ""
		ui.clearUI()
		return
	case "/favorites":
		ui.send.Text = ""
		ui.showFavoritesTopics()
		return
	case "/help", "/home":
		ui.send.Text = ""
		ui.showHome()
		return
	case "/me":
		ui.send.Text = ""
		ui.showMe()
		return
	case "/quit":
		ui.send.Text = ""
		termui.StopLoop()
	case "/save":
		ui.send.Text = ""
		ui.saveConfig()
		return
	case "/topics":
		ui.send.Text = ""
		ui.showTopics()
		return
	case "/toggle-top":
		ui.send.Text = ""
		ui.toggleTopMenu(false)
		return
	case "/toggle-bottom":
		ui.send.Text = ""
		ui.toggleActionBox(false)
		return
	case "/hide-top":
		ui.send.Text = ""
		ui.toggleTopMenu(true)
		return
	case "/hide-bottom":
		ui.send.Text = ""
		ui.toggleActionBox(true)
		return
	case "/unread":
		ui.send.Text = ""
		ui.showUnreadTopics()
		return
	case "/version":
		ui.send.Text = ""
		ui.showVersion()
		return
	}

	if strings.HasPrefix(ui.send.Text, "/set-tatwebui-url ") {
		ui.setTatWebUIURL(ui.send.Text)
		ui.send.Text = ""
		return
	} else if ui.current == uiMessages && (strings.HasPrefix(ui.send.Text, "/split ")) {
		ui.messagesSplit(ui.send.Text, "/view")
		ui.send.Text = ""
		return
	} else if (ui.current == uiMessages || ui.current == uiTopics) &&
		(ui.send.Text == "/codereview" || ui.send.Text == "/monitoring") {
		ui.messagesSimpleCustomSplit(ui.send.Text)
		ui.send.Text = ""
		return
	} else if ui.current == uiMessages && (strings.HasPrefix(ui.send.Text, "/run ")) {
		ui.messagesRun(ui.send.Text)
		ui.send.Text = ""
		return
	} else if ui.current == uiMessages && (strings.HasPrefix(ui.send.Text, "/mode ")) {
		ui.setMode(ui.send.Text)
		ui.send.Text = ""
		return
	} else if ui.current == uiMessages && (strings.HasPrefix(ui.send.Text, "/filter-messages ") || strings.HasPrefix(ui.send.Text, "/filter ")) {
		ui.setFilterMessages(ui.selectedPaneMessages, ui.send.Text, "/view")
		ui.updateMessages()
		ui.send.Text = ""
		return
	} else if ui.current == uiTopics && (strings.HasPrefix(ui.send.Text, "/filter-topics ") || strings.HasPrefix(ui.send.Text, "/filter ")) {
		ui.setFilterTopics()
		ui.send.Text = ""
		return
	} else if ui.current == uiMessage || ui.current == uiMessages {
		ui.processCmdOnMessage()
		ui.send.Text = ""
		return
	}
}

func (ui *tatui) processCmdOnMessage() {
	if ui.send.Text == "/open" {
		ui.openInTatwebui()
	} else if ui.send.Text == "/open-links" {
		ui.openLinksInBrowser()
	} else if ui.send.Text == "/toggle-usernames" {
		ui.toggleUsernames(false)
	} else if ui.send.Text == "/hide-usernames" {
		ui.toggleUsernames(true)
	} else if strings.HasPrefix(ui.send.Text, "/label ") {
		ui.sendLabel()
	} else if strings.HasPrefix(ui.send.Text, "/unlabel ") {
		ui.sendUnlabel()
	} else if strings.HasPrefix(ui.send.Text, "/vote") ||
		strings.HasPrefix(ui.send.Text, "/like") ||
		strings.HasPrefix(ui.send.Text, "/unlike") ||
		strings.HasPrefix(ui.send.Text, "/task") ||
		strings.HasPrefix(ui.send.Text, "/untask") ||
		strings.HasPrefix(ui.send.Text, "/unvote") {
		// /voteup, /votedown, /unvoteup, /unvotedown, /like, /unlike, /task, /untask
		ui.sendSimpleActionMsg()
	}

	for _, h := range ui.hooks {
		if strings.HasPrefix(ui.send.Text, h.Command) {
			ui.RunExec(&h, "", ui.send.Text)
		}
	}

	ui.send.Text = ""
	ui.render()
}

func (ui *tatui) sendSimpleActionMsg() {
	t := strings.TrimSpace(ui.send.Text[1:])

	var err error
	var msg *tat.MessageJSONOut
	switch t {
	case "voteup":
		msg, err = internal.Client().MessageVoteUP(ui.currentTopic.Topic, ui.currentListMessages[ui.selectedPaneMessages][ui.uilists[uiMessages][ui.selectedPaneMessages].position].ID)
	case "votedown":
		msg, err = internal.Client().MessageVoteDown(ui.currentTopic.Topic, ui.currentListMessages[ui.selectedPaneMessages][ui.uilists[uiMessages][ui.selectedPaneMessages].position].ID)
	case "unvoteup":
		msg, err = internal.Client().MessageUnVoteUP(ui.currentTopic.Topic, ui.currentListMessages[ui.selectedPaneMessages][ui.uilists[uiMessages][ui.selectedPaneMessages].position].ID)
	case "unvotedown":
		msg, err = internal.Client().MessageUnVoteDown(ui.currentTopic.Topic, ui.currentListMessages[ui.selectedPaneMessages][ui.uilists[uiMessages][ui.selectedPaneMessages].position].ID)
	case "like":
		msg, err = internal.Client().MessageLike(ui.currentTopic.Topic, ui.currentListMessages[ui.selectedPaneMessages][ui.uilists[uiMessages][ui.selectedPaneMessages].position].ID)
	case "unlike":
		msg, err = internal.Client().MessageUnlike(ui.currentTopic.Topic, ui.currentListMessages[ui.selectedPaneMessages][ui.uilists[uiMessages][ui.selectedPaneMessages].position].ID)
	case "task":
		msg, err = internal.Client().MessageTask(ui.currentTopic.Topic, ui.currentListMessages[ui.selectedPaneMessages][ui.uilists[uiMessages][ui.selectedPaneMessages].position].ID)
	case "untask":
		msg, err = internal.Client().MessageUntask(ui.currentTopic.Topic, ui.currentListMessages[ui.selectedPaneMessages][ui.uilists[uiMessages][ui.selectedPaneMessages].position].ID)
	}

	if err != nil {
		ui.msg.Text = err.Error()
	} else {
		ui.uilists[uiMessages][ui.selectedPaneMessages].list.Items[ui.uilists[uiMessages][ui.selectedPaneMessages].position] = ui.formatMessage(msg.Message, true)
		ui.addMarker(ui.uilists[uiMessages][ui.selectedPaneMessages], ui.selectedPaneMessages)
	}
	if msg.Info != "" {
		ui.msg.Text = msg.Info
	}
}

func (ui *tatui) sendLabel() {
	if len(strings.TrimSpace(ui.send.Text)) == 0 {
		return
	}

	// "/label " : 6 char
	t := ui.send.Text[6:]
	// /label ###### text
	color := t[0:strings.Index(t, " ")]
	text := t[strings.Index(t, " ")+1:]

	msg, err := internal.Client().MessageLabel(ui.currentTopic.Topic, ui.currentListMessages[ui.selectedPaneMessages][ui.uilists[uiMessages][ui.selectedPaneMessages].position].ID, tat.Label{Text: text, Color: color})
	if err != nil {
		ui.msg.Text = err.Error()
	}
	if msg.Info != "" {
		ui.msg.Text = msg.Info
	}
}

func (ui *tatui) sendUnlabel() {
	if len(strings.TrimSpace(ui.send.Text)) == 0 {
		return
	}

	// "/unlabel " : 8 char
	labelToRemove := strings.TrimSpace(ui.send.Text[8:])
	msg, err := internal.Client().MessageUnlabel(ui.currentTopic.Topic, ui.currentListMessages[ui.selectedPaneMessages][ui.uilists[uiMessages][ui.selectedPaneMessages].position].ID, labelToRemove)
	if err != nil {
		ui.msg.Text = err.Error()
	}
	if msg.Info != "" {
		ui.msg.Text = msg.Info
	}
}

func (ui *tatui) clearUI() {
	ui.uiTopicCommands[ui.currentTopic.Topic] = ""
	if ui.current == uiTopics {
		ui.currentFilterTopics = &tat.TopicCriteria{}
		ui.msg.Text = "filters on topics cleared"
		ui.updateTopics()
	} else if ui.current == uiMessages {
		ui.clearFilterOnCurrentTopic()
		ui.msg.Text = "filters on messages cleared"
		ui.showMessages()
	}
}

func (ui *tatui) clearFilterOnCurrentTopic() {
	ui.uilists[uiMessages] = make(map[int]*uilist)
	ui.currentFilterMessages[ui.currentTopic.Topic] = make(map[int]*tat.MessageCriteria)
	ui.currentFilterMessages[ui.currentTopic.Topic][0] = nil
	ui.currentFilterMessagesText[ui.currentTopic.Topic] = make(map[int]string)
	ui.currentFilterMessagesText[ui.currentTopic.Topic][0] = ""
	ui.currentModeOnTopic[ui.currentTopic.Topic] = "/view"
}

func (ui *tatui) setFilterTopics() {
	c := &tat.TopicCriteria{}

	for _, s := range strings.Split(ui.send.Text, " ") {
		tuple := strings.Split(s, ":")
		if len(tuple) != 2 || len(strings.TrimSpace(tuple[1])) == 0 {
			continue
		}

		if strings.EqualFold(tuple[0], "IDTopic") {
			c.IDTopic = tuple[1]
		}
		if strings.EqualFold(tuple[0], "Topic") {
			c.Topic = tuple[1]
		}
		if strings.EqualFold(tuple[0], "Description") {
			c.Description = tuple[1]
		}
		if strings.EqualFold(tuple[0], "DateMinCreation") {
			c.DateMinCreation = tuple[1]
		}
		if strings.EqualFold(tuple[0], "DateMaxCreation") {
			c.DateMaxCreation = tuple[1]
		}
		if strings.EqualFold(tuple[0], "GetForTatAdmin") {
			c.GetForTatAdmin = tuple[1]
		}
		if strings.EqualFold(tuple[0], "Group") {
			c.Group = tuple[1]
		}
	}

	ui.uilists[uiTopics][0].list.BorderLabel = ui.uilists[uiTopics][0].list.BorderLabel + " filter on " + ui.send.Text + " "
	ui.currentFilterTopics = c
	ui.switchBox()
	ui.updateTopics()
}

func (ui *tatui) showMe() {
	ui.msg.Text = ""
	body, err := internal.Client().UserMe()
	if err != nil {
		ui.msg.Text = err.Error()
		return
	}

	out, err := tat.Sprint(body)
	if err != nil {
		ui.msg.Text = err.Error()
		return
	}

	ui.showResult("tatcli user me -> GET on /user/me ", string(out))
}

func (ui *tatui) showVersion() {
	ui.msg.Text = ""
	body, err := internal.Client().Version()
	if err != nil {
		ui.msg.Text = err.Error()
		return
	}
	ui.showResult("tatcli version -> GET on "+viper.GetString("url")+"/version ", string(body))
	ui.msg.Text = "Tatcli:" + tat.Version
}

func (ui *tatui) sendMsg() {
	if ui.current != uiMessage && ui.current != uiMessages {
		return
	}
	if len(strings.TrimSpace(ui.send.Text)) == 0 {
		return
	}
	var err error
	var msg *tat.MessageJSONOut
	switch ui.current {
	case uiMessages:
		msg, err = message.Create(ui.currentTopic.Topic, ui.send.Text)
	case uiMessage:
		msg, err = internal.Client().MessageReply(ui.currentTopic.Topic, ui.currentMessage.ID, ui.send.Text)
	}
	if err != nil {
		ui.msg.Text = err.Error()
		return
	}
	if msg.Info != "" {
		ui.msg.Text = msg.Info
	}
	ui.send.Text = ""
	ui.render()
	ui.updateMessages()
}

func (ui *tatui) setMode(mode string) {
	modeValue := strings.TrimSpace(strings.Replace(mode, "/mode ", "", 1))
	if !strings.HasPrefix(modeValue, "/") {
		modeValue = "/" + modeValue
	}
	ui.currentModeOnTopic[ui.currentTopic.Topic] = modeValue
	ui.msg.Text = "Mode " + ui.currentModeOnTopic[ui.currentTopic.Topic] + " activated on topic " + ui.currentTopic.Topic
	ui.applyLabelOnMsgPanes()
}
