package ui

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gizak/termui"
	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/config"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/viper"
)

func (ui *tatui) showMessages() {
	ui.current = uiMessages
	ui.selectedPane = uiMessages
	ui.send.BorderLabel = " ✎ Action or New Message "
	termui.Body.Rows = nil

	ui.selectedPaneMessages = 0

	if len(ui.currentListMessages) == 0 {
		ui.currentListMessages[0] = nil
	}

	if _, ok := ui.uilists[uiTopics]; !ok || len(ui.uilists[uiTopics]) == 0 {
		ui.msg.Text = "Please select a topic before doing this action"
		ui.showHome()
		return
	}

	if _, ok := ui.currentFilterMessages[ui.currentTopic.Topic]; !ok {
		ui.clearFilterOnCurrentTopic()
	}

	ui.initMessages()

	go func() {
		for {
			if ui.current != uiMessages {
				break
			}
			mutex.Lock()
			ui.updateMessages()
			ui.firstCallMessages = true
			mutex.Unlock()
			time.Sleep(5 * time.Second)
		}
	}()

	ui.uilists[uiTopics][0].list.BorderRight = true

	ui.prepareTopMenu()

	if len(ui.currentFilterMessages[ui.currentTopic.Topic]) > 1 {
		// preserve order
		for k := 0; k < len(ui.currentFilterMessages[ui.currentTopic.Topic]); k++ {
			termui.Body.AddRows(termui.NewRow(termui.NewCol(12, 0, ui.uilists[uiMessages][k].list)))
		}
	} else {
		termui.Body.AddRows(
			termui.NewRow(
				termui.NewCol(3, 0, ui.uilists[uiTopics][0].list),
				termui.NewCol(9, 0, ui.uilists[uiMessages][0].list),
			),
		)
	}

	ui.prepareSendRow()
	ui.colorizedPanes()
	termui.Clear()
	ui.render()
}

func (ui *tatui) initMessages() {
	for k := range ui.currentFilterMessages[ui.currentTopic.Topic] {
		strs := []string{"[Loading...](fg-black,bg-white)"}
		ls := termui.NewList()
		ls.BorderTop, ls.BorderLeft, ls.BorderRight, ls.BorderBottom = true, false, false, false
		ls.Items = strs
		ls.ItemFgColor = termui.ColorWhite
		ls.BorderLabel = "Messages"
		ls.Width = 25
		ls.Y = 0
		ls.Height = (termui.TermHeight() - uiHeightTop - uiHeightSend) / len(ui.currentFilterMessages[ui.currentTopic.Topic])

		if _, ok := ui.uilists[uiMessages]; !ok {
			ui.uilists[uiMessages] = make(map[int]*uilist)
		}
		ui.uilists[uiMessages][k] = &uilist{uiType: uiMessages, list: ls, position: 0, page: 0, update: ui.updateMessages}
	}
	ui.applyLabelOnMsgPanes()
	ui.render()
}

func (ui *tatui) enterMessage() {
	ui.showMessage()
	ui.render()
}

func (ui *tatui) updateMessages() {
	start := time.Now().UnixNano()
	ui.msg.Text = ""
	for pane := range ui.currentFilterMessages[ui.currentTopic.Topic] {
		if pane > len(ui.uilists[uiMessages]) {
			continue
		}
		nbPerPage := (ui.getNbPerPage() / len(ui.currentFilterMessages[ui.currentTopic.Topic])) - 1
		c := ui.currentFilterMessages[ui.currentTopic.Topic][pane]
		if c == nil {
			c = &tat.MessageCriteria{}
		}
		c.Skip = ui.uilists[uiMessages][pane].page * nbPerPage
		c.Limit = nbPerPage
		c.Topic = ui.currentTopic.Topic
		c.OnlyMsgRoot = "true"
		ui.updateMessagesPane(pane, c)
		delta := int64((time.Now().UnixNano() - start) / 1000000)
		ui.lastRefresh.Text = fmt.Sprintf("%s (%dms)", time.Now().Format(time.Stamp), delta)
		ui.render()
	}
}

func (ui *tatui) updateMessagesPane(pane int, criteria *tat.MessageCriteria) {

	// check if ui.uilists[uiMessages] has changed
	if pane >= len(ui.uilists[uiMessages]) {
		return
	}

	messagesJSON, err := internal.Client().MessageList(criteria.Topic, criteria)
	if err != nil {
		ui.msg.Text = err.Error()
		return
	}
	var strs []string
	for _, msg := range messagesJSON.Messages {
		strs = append(strs, ui.formatMessage(msg, true))
	}

	ui.uilists[uiMessages][pane].list.Items = strs
	ui.currentListMessages[pane] = messagesJSON.Messages
	ui.msg.Text = ""
	if ui.selectedPaneMessages == pane {
		ui.addMarker(ui.uilists[uiMessages][pane], pane)
	}
	ui.render()
}

func (ui *tatui) formatMessage(msg tat.Message, withColor bool) string {
	start := ""
	end := ""
	colorUsername := ""
	colorLabel := ""
	if withColor {
		start = "["
		end = "]"
		colorUsername = "(fg-cyan)"
		colorLabel = "(fg-blue)"
	}
	text := fmt.Sprintf("%s ", time.Unix(int64(msg.DateCreation), 0).Format(time.Stamp))
	if !strings.Contains(ui.uiTopicCommands[ui.currentTopic.Topic], " /hide-usernames") {
		text += fmt.Sprintf("%s%s%s%s ", start, msg.Author.Username, end, colorUsername)
	}

	if msg.NbReplies > 0 {
		text += fmt.Sprintf("%s ", ui.getNbMsg(msg.NbReplies))
	}

	if msg.NbVotesUP > 0 {
		text += fmt.Sprintf("%d☝ ", msg.NbVotesUP)
	}

	if msg.NbVotesDown > 0 {
		text += fmt.Sprintf("%d☟ ", msg.NbVotesDown)
	}

	if msg.NbLikes > 0 {
		text += fmt.Sprintf("%d♡ ", msg.NbLikes)
	}

	for _, label := range msg.Labels {
		ccolor := ""
		if withColor {
			ccolor = colorLabel
			if label.Text == "UP" || label.Text == "done" || label.Text == "Success" {
				ccolor = "(fg-green)"
			} else if label.Text == "AL" || label.Text == "open" || label.Text == "Fail" {
				ccolor = "(fg-red)"
			} else if strings.HasPrefix(label.Text, "doing") || label.Text == "Building" {
				ccolor = "(fg-blue)"
			} else if label.Text == "Waiting" {
				ccolor = "(fg-yellow)"
			}
		}
		text += fmt.Sprintf("%s"+label.Text+"〉%s%s", start, end, ccolor)
	}

	return fmt.Sprintf("%s%s", text, msg.Text)
}

func (ui *tatui) getNbMsg(nb int64) string {
	return "☾" + strconv.FormatInt(nb, 10) + "☽"
}

func (ui *tatui) prepareFilterMessages(text, mode, topic string) (*tat.MessageCriteria, string) {
	c := &tat.MessageCriteria{}

	for _, s := range strings.Split(text, " ") {
		tuple := strings.Split(s, ":")
		if len(tuple) < 2 || len(strings.TrimSpace(tuple[1])) == 0 {
			continue
		}

		if strings.EqualFold(tuple[0], "TreeView") {
			c.TreeView = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "IDMessage") {
			c.IDMessage = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "InReplyOfID") {
			c.InReplyOfID = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "InReplyOfIDRoot") {
			c.InReplyOfIDRoot = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "AllIDMessage") {
			c.AllIDMessage = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "Text") {
			c.Text = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "Topic") {
			c.Topic = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "Label") {
			c.Label = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "NotLabel") {
			c.NotLabel = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "AndLabel") {
			c.AndLabel = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "Tag") {
			c.Tag = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "NotTag") {
			c.NotTag = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "AndTag") {
			c.AndTag = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "Username") {
			c.Username = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "DateMinCreation") {
			c.DateMinCreation = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "DateMaxCreation") {
			c.DateMaxCreation = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "DateMinUpdate") {
			c.DateMinUpdate = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "DateMaxUpdate") {
			c.DateMaxUpdate = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "LimitMinNbReplies") {
			c.LimitMinNbReplies = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "LimitMaxNbReplies") {
			c.LimitMaxNbReplies = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "LimitMinNbVotesUP") {
			c.LimitMinNbVotesUP = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "LimitMinNbVotesDown") {
			c.LimitMinNbVotesDown = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "LimitMaxNbVotesUP") {
			c.LimitMaxNbVotesUP = strings.Join(tuple[1:], ":")
		}
		if strings.EqualFold(tuple[0], "LimitMaxNbVotesDown") {
			c.LimitMaxNbVotesDown = strings.Join(tuple[1:], ":")
		}
	}
	if mode == "" {
		mode = "/view"
	}
	ui.currentModeOnTopic[topic] = strings.TrimSpace(mode)
	return c, mode + " " + topic + " " + text + " "
}

func (ui *tatui) setFilterMessages(pane int, text, mode string) {
	c, criteriaText := ui.prepareFilterMessages(text, mode, ui.currentTopic.Topic)
	ui.currentFilterMessages[ui.currentTopic.Topic][pane] = c
	ui.currentFilterMessagesText[ui.currentTopic.Topic][pane] = criteriaText
}

func (ui *tatui) messagesSplit(text, mode string) {
	ui.clearFilterOnCurrentTopic()
	criteriaText := strings.Replace(text, mode, "", 1)
	criteriaText = strings.Replace(criteriaText, "/split", "", 1)
	i := 0
	for _, v := range strings.Split(criteriaText, " ") {
		if strings.TrimSpace(v) == "" {
			continue
		}
		w := strings.Replace(v, ";", " ", -1)
		ui.setFilterMessages(i, w, mode)
		i++
	}
	ui.showMessages()
	ui.applyLabelOnMsgPanes()
	ui.render()
}

func (ui *tatui) applyLabelOnMsgPanes() {
	mode := ui.currentModeOnTopic[ui.currentTopic.Topic]
	for k := range ui.currentFilterMessages[ui.currentTopic.Topic] {
		label := strings.Split(ui.currentFilterMessagesText[ui.currentTopic.Topic][k], " ")
		if len(label) > 2 {
			ui.uilists[uiMessages][k].list.BorderLabel = mode + " " + strings.Join(label[1:], " ")
		} else {
			ui.uilists[uiMessages][k].list.BorderLabel = mode + " " + ui.currentTopic.Topic
		}
	}
}

func (ui *tatui) messagesRun(text string) {
	tag := strings.Replace(text, "/run ", "", -1)
	ui.messagesSplit(fmt.Sprintf("tag:%s;label:open tag:%s;label:doing tag:%s;label:done", tag, tag, tag), "/run")
}

func (ui *tatui) messagesSimpleCustomSplit(typeSplit string) {
	switch typeSplit {
	case "/monitoring":
		ui.messagesSplit(fmt.Sprintf("label:AL label:UP notLabel:AL,UP"), typeSplit)
	case "/codereview":
		ui.messagesSplit(fmt.Sprintf("label:OPENED label:APPROVED label:MERGED label:DECLINED"), typeSplit)
	}
}

func (ui *tatui) monitoringActionOnMessage() {

	if _, ok := ui.uilists[uiMessages][ui.selectedPaneMessages]; !ok {
		return
	}
	if ui.uilists[uiMessages][ui.selectedPaneMessages].position >= len(ui.currentListMessages[ui.selectedPaneMessages]) {
		return
	}
	msg := ui.currentListMessages[ui.selectedPaneMessages][ui.uilists[uiMessages][ui.selectedPaneMessages].position]
	var erradd, errdel error

	// #d04437 red open
	// #14892c green done
	// #5484ed blue doing

	// TODO use relabel

	if msg.ContainsLabel("AL") {
		// set to doing
		if !msg.ContainsLabel("UP") {
			_, erradd = internal.Client().MessageLabel(msg.Topic, msg.ID, tat.Label{Text: "UP", Color: hexColorGreen})
		}
		_, errdel = internal.Client().MessageUnlabel(msg.Topic, msg.ID, "AL")
	} else {
		// set to AL
		if !msg.ContainsLabel("AL") {
			_, erradd = internal.Client().MessageLabel(msg.Topic, msg.ID, tat.Label{Text: "AL", Color: hexColorRed})
		}
		if msg.ContainsLabel("UP") {
			_, errdel = internal.Client().MessageUnlabel(msg.Topic, msg.ID, "UP")
		}
	}

	ui.msg.Text = ""
	if erradd != nil {
		ui.msg.Text += "add:" + erradd.Error()
	}
	if errdel != nil {
		ui.msg.Text += " del:" + errdel.Error()
		return
	}
	msg.Text = "please wait, updating message..."
	ui.render()
	ui.updateMessages()
}

func (ui *tatui) postHookRunActionOnMessage(action string, msg tat.Message) {
	hook := strings.TrimSpace(viper.GetString("post-hook-run-action"))
	if hook == "" {
		return
	}

	_, err := exec.LookPath(hook)
	if err != nil {
		internal.Exit("Invalid hook path for post-hook-run-action, err: %s", err.Error())
		return
	}

	jsonStr, err := json.Marshal(msg)
	if err != nil {
		internal.Exit("Error while marshalling msg for post-hook-run-action, err: %s", err.Error())
		return
	}

	cmd := exec.Command(hook, action, msg.ID, string(jsonStr))
	if e := cmd.Start(); e != nil {
		internal.Exit("Error with post-hook-run-action, err: %s", err.Error())
		return
	}

	ui.msg.Text = "Waiting " + hook + "for command to finish..."
	err = cmd.Wait()
	ui.msg.Text = fmt.Sprintf("Command finished: %v", err)
}

func (ui *tatui) runActionOnMessage(reverse bool) {
	if _, ok := ui.uilists[uiMessages][ui.selectedPaneMessages]; !ok {
		return
	}
	if ui.uilists[uiMessages][ui.selectedPaneMessages].position >= len(ui.currentListMessages[ui.selectedPaneMessages]) {
		return
	}
	msg := ui.currentListMessages[ui.selectedPaneMessages][ui.uilists[uiMessages][ui.selectedPaneMessages].position]
	var erradd, erradd2, errdel, errdel2 error

	// #d04437 red open
	// #14892c green done
	// #5484ed blue doing

	action := "notdefined"

	if (msg.ContainsLabel("open") && !reverse) ||
		(msg.ContainsLabel("done") && reverse) { // set to doing
		errdel = ui.removeLabel(msg, "open")
		errdel2 = ui.removeLabel(msg, "done")
		if !msg.ContainsLabel("doing") {
			//_, erradd = message.Action("task", "/", msg.ID, "", "")
			_, erradd = internal.Client().MessageTask(msg.Topic, msg.ID)
		}
		action = "doing"
	} else if (msg.ContainsLabel("doing") && !reverse) ||
		(msg.ContainsLabel("open") && reverse) { // set to done
		errdel = ui.removeLabel(msg, "doing")
		errdel2 = ui.removeLabel(msg, "open")
		if !msg.ContainsLabel("done") {
			//_, erradd = message.Action("label", "/", msg.ID, "done", hexColorGreen)
			_, erradd = internal.Client().MessageLabel(msg.Topic, msg.ID, tat.Label{Text: "done", Color: hexColorGreen})
		}
		action = "done"
	} else if (msg.ContainsLabel("done") && !reverse) ||
		(msg.ContainsLabel("doing") && reverse) { // set to open
		errdel = ui.removeLabel(msg, "done")
		errdel2 = ui.removeLabel(msg, "doing")
		if !msg.ContainsLabel("open") {
			//_, erradd = message.Action("label", "/", msg.ID, "open", hexColorRed)
			_, erradd = internal.Client().MessageLabel(msg.Topic, msg.ID, tat.Label{Text: "open", Color: hexColorRed})
		}
		action = "open"
	} else {
		_, erradd = internal.Client().MessageTask(msg.Topic, msg.ID)
		action = "doing"
	}

	ui.msg.Text = ""
	if erradd != nil {
		ui.msg.Text += "add:" + erradd.Error()
	}
	if erradd2 != nil {
		ui.msg.Text += "add:" + erradd2.Error()
	}
	if errdel != nil {
		ui.msg.Text += " del:" + errdel.Error()
		return
	}
	if errdel2 != nil {
		ui.msg.Text += " del:" + errdel2.Error()
		return
	}
	ui.postHookRunActionOnMessage(action, msg)
	msg.Text = "please wait, updating message..."
	ui.render()
	ui.updateMessages()
}

func (ui *tatui) removeLabel(msg tat.Message, label string) error {
	var err error
	if label == "doing" && msg.ContainsLabel("doing") {
		internal.Client().MessageUntask(msg.Topic, msg.ID)
	} else {
		internal.Client().MessageUnlabel(msg.Topic, msg.ID, label)
		if label == "done" {
			for _, l := range msg.Labels {
				if strings.HasPrefix(l.Text, "done:") {
					internal.Client().MessageUnlabel(msg.Topic, msg.ID, l.Text)
				}
			}
		}
	}
	return err
}

func (ui *tatui) RunExec(hookIn *config.Hook, pathHook string, sendText string) {

	if ui.current != uiMessage && ui.current != uiMessages {
		return
	}

	if sendText == "" && pathHook == "" {
		return
	}
	if sendText != "" && hookIn == nil {
		return
	}
	var hook *config.Hook
	if pathHook != "" {
		for _, h := range ui.hooks {
			if "/sys/kbd/"+h.Shortcut == pathHook {
				hook = &h
				sendText = ""
				break
			}
		}
	} else {
		hook = hookIn
	}

	userExec := strings.Split(hook.Exec, " ")
	execCMD := strings.TrimSpace(userExec[0])
	if execCMD == "" {
		return
	}

	_, err := exec.LookPath(execCMD)
	if err != nil {
		internal.Exit("Invalid exec path for post-exec-run-action, err: %s", err.Error())
		return
	}

	msg := ui.currentListMessages[ui.selectedPaneMessages][ui.uilists[uiMessages][ui.selectedPaneMessages].position]

	if len(hook.Topics) > 0 && !tat.ArrayContains(hook.Topics, msg.Topic) {
		return
	}

	toExec := strings.Replace(hook.Exec, "$UI_SELECTED_MSG_ID", msg.ID, 1)
	toExec = strings.Replace(toExec, "$UI_SELECTED_MSG_TEXT", msg.Text, 1)
	toExec = strings.Replace(toExec, "$UI_SELECTED_MSG_TOPIC", msg.Topic, 1)
	toExec = strings.Replace(toExec, "$UI_SELECTED_MSG_AUTHOR_USERNAME", msg.Author.Username, 1)
	toExec = strings.Replace(toExec, "$UI_SELECTED_MSG_DATE_CREATION", fmt.Sprintf("%f", msg.DateCreation), 1)
	toExec = strings.Replace(toExec, "$UI_SELECTED_MSG_DATE_UPDATE", fmt.Sprintf("%f", msg.DateUpdate), 1)
	toExec = strings.Replace(toExec, "$UI_CURRENT_USERNAME", viper.GetString("username"), 1)

	if sendText != "" {
		sendTextWithoutCMD := strings.Replace(sendText, hook.Command+" ", "", 1)
		toExec = strings.Replace(toExec, "$UI_ACTION_TEXT", sendTextWithoutCMD, 1)
	}

	args := []string{}
	if len(toExec) > 1 {
		args = strings.Split(toExec, " ")
	}

	cmd := exec.Command(execCMD, args[1:]...)
	if e := cmd.Start(); e != nil {
		internal.Exit("Error with exec hook, err: %s", e.Error())
		return
	}

	ui.msg.Text = "Waiting " + hook.Exec + " for command to finish..."
	err = cmd.Wait()
	if err != nil {
		ui.msg.Text = fmt.Sprintf("Error:%s cmd:%s", err.Error(), strings.Join(cmd.Args, " "))
	} else {
		ui.msg.Text = fmt.Sprintf("Success: %s", strings.Join(cmd.Args, " "))
	}

}
