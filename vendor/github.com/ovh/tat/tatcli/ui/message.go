package ui

import (
	"fmt"
	"time"

	"github.com/gizak/termui"
	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
)

func (ui *tatui) showMessage() {
	ui.current = uiMessage
	ui.selectedPane = uiMessage
	ui.send.BorderLabel = " ✎ Action or New Reply "
	termui.Body.Rows = nil

	if ui.uilists[uiMessages][ui.selectedPaneMessages].list == nil || ui.uilists[uiMessages][ui.selectedPaneMessages].position < 0 {
		return
	}

	ui.uilists[uiMessage] = make(map[int]*uilist)

	ui.initMessage()

	go func() {
		for {
			if ui.current != uiMessage {
				break
			}
			mutex.Lock()
			ui.updateMessage()
			mutex.Unlock()
			time.Sleep(5 * time.Second)
		}
	}()
	ui.addMarker(ui.uilists[uiMessage][0], 0)

	ui.prepareTopMenu()
	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(3, 0, ui.uilists[uiTopics][0].list),
			termui.NewCol(9, 0, ui.uilists[uiMessage][0].list),
		),
	)
	ui.prepareSendRow()
	ui.colorizedPanes()
	termui.Clear()
	ui.render()
}

func (ui *tatui) initMessage() {
	strs := []string{"[Loading...](fg-black,bg-white)"}

	ls := termui.NewList()
	ls.BorderTop, ls.BorderLeft, ls.BorderRight, ls.BorderBottom = true, false, false, false
	ls.Items = strs
	ls.ItemFgColor = termui.ColorWhite
	ls.BorderLabel = "Message"
	ls.Overflow = "wrap"
	ls.Width = 25
	ls.Y = 0
	ls.Height = termui.TermHeight() - uiHeightTop - uiHeightSend
	_, ok := ui.uilists[uiMessages][ui.selectedPaneMessages]
	if ok && ui.uilists[uiMessages][ui.selectedPaneMessages].position >= 0 && ui.uilists[uiMessages][ui.selectedPaneMessages].position < len(ui.currentListMessages[ui.selectedPaneMessages]) {
		ui.currentMessage = ui.currentListMessages[ui.selectedPaneMessages][ui.uilists[uiMessages][ui.selectedPaneMessages].position]
		ui.uilists[uiMessage][0] = &uilist{uiType: uiMessage, list: ls, position: 0, page: 0, update: ui.updateMessage}
	}
}

func (ui *tatui) updateMessage() {
	start := time.Now().UnixNano()

	c := &tat.MessageCriteria{
		IDMessage: ui.currentMessage.ID,
		Topic:     ui.currentTopic.Topic,
		TreeView:  "onetree",
	}
	messagesJSON, err := internal.Client().MessageList(ui.currentTopic.Topic, c)
	if err != nil {
		ui.msg.Text = err.Error()
		ui.render()
		return
	}

	thread := []string{}
	for _, msg := range messagesJSON.Messages {
		thread = ui.addMessage(msg, 0)
	}

	ui.uilists[uiMessage][0].list.Items = thread
	ui.uilists[uiMessage][0].list.BorderLabel = " View Message, Ctrl+b to return back "
	delta := int64((time.Now().UnixNano() - start) / 1000000)
	ui.lastRefresh.Text = fmt.Sprintf("%s (%dms)", time.Now().Format(time.Stamp), delta)
	ui.msg.Text = ""
}

func (ui *tatui) addMessage(msg tat.Message, deep int) []string {
	text := ""
	for n := 0; n < deep; n++ {
		text += " "
	}
	if deep > 0 {
		text += " ➥ "
	}

	text += ui.formatMessage(msg, true)

	thread := []string{}
	thread = append(thread, text)
	deep++
	for _, replies := range msg.Replies {
		thread = append(thread, ui.addMessage(replies, deep)...)
	}
	return thread
}
