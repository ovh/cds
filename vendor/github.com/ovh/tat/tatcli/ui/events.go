package ui

import (
	"fmt"
	"strings"

	"github.com/gizak/termui"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/viper"
)

func (ui *tatui) initHandles() {
	// Setup handlers
	termui.Handle("/timer/1s", func(e termui.Event) {
		t := e.Data.(termui.EvtTimer)
		ui.draw(int(t.Count))
	})

	termui.Handle("/sys/kbd/C-q", func(termui.Event) {
		termui.StopLoop()
	})

	// Show Home -> C-h, but send <backspace> event
	termui.Handle("/sys/kbd/<backspace>", func(e termui.Event) {
		ui.showHome()
	})

	// C-c is same as /clear
	termui.Handle("/sys/kbd/C-c", func(e termui.Event) {
		ui.clearUI()
	})

	// All topics
	termui.Handle("/sys/kbd/C-b", func(e termui.Event) {
		if ui.current == uiMessage {
			ui.showMessages()
		}
	})

	// All topics
	termui.Handle("/sys/kbd/C-a", func(e termui.Event) {
		ui.current = uiTopics
		ui.onlyFavorites = "false"
		ui.onlyUnread = false
		ui.showTopics()
	})

	// Unread Topics
	termui.Handle("/sys/kbd/C-r", func(e termui.Event) {
		ui.showUnreadTopics()
	})

	// toggle usernames
	termui.Handle("/sys/kbd/C-u", func(e termui.Event) {
		ui.toggleUsernames(false)
	})

	// Favorites Topics
	termui.Handle("/sys/kbd/C-f", func(e termui.Event) {
		ui.showFavoritesTopics()
	})

	termui.Handle("/sys/kbd/<up>", func(e termui.Event) {
		ui.move("up")
	})

	termui.Handle("/sys/kbd/<down>", func(e termui.Event) {
		ui.move("down")
	})

	termui.Handle("/sys/kbd/<enter>", func(e termui.Event) {
		switch ui.selectedPane {
		case uiTopics:
			ui.enterTopic()
			return
		case uiMessages:
			ui.enterMessage()
			return
		case uiActionBox:
			ui.processMsg()
			ui.render()
		}
	})

	termui.Handle("/sys/kbd/<space>", func(e termui.Event) {
		if ui.isOnActionBox() {
			ui.send.Text += " "
		}
		ui.render()
	})

	termui.Handle("/sys/kbd/<tab>", func(e termui.Event) {
		if ui.isOnActionBox() && len(ui.send.Text) > 0 && !strings.HasSuffix(ui.send.Text, " ") {
			ui.autocomplete()
		} else {
			ui.switchBox()
		}
	})

	termui.Handle("/sys/kbd/C-8", func(e termui.Event) {
		if !ui.isOnActionBox() {
			return
		}
		if len(ui.send.Text) > 0 {
			ui.send.Text = ui.send.Text[:len(ui.send.Text)-1]
			ui.render()
		}
	})

	termui.Handle("/sys/kbd/C-k", func(e termui.Event) {
		if ui.currentModeOnTopic[ui.currentTopic.Topic] == "/run" {
			ui.runActionOnMessage(true)
		} else if ui.currentModeOnTopic[ui.currentTopic.Topic] == "/monitoring" {
			ui.monitoringActionOnMessage()
		}
	})

	termui.Handle("/sys/kbd/C-j", func(e termui.Event) {
		if ui.currentModeOnTopic[ui.currentTopic.Topic] == "/run" {
			ui.runActionOnMessage(false)
		} else if ui.currentModeOnTopic[ui.currentTopic.Topic] == "/monitoring" {
			ui.monitoringActionOnMessage()
		}
	})

	termui.Handle("/sys/kbd/C-p", func(e termui.Event) {
		if _, ok := ui.uilists[uiMessages][ui.selectedPaneMessages]; ok {
			ui.openLinksInBrowser()
		}
	})

	termui.Handle("/sys/kbd/C-t", func(e termui.Event) {
		ui.toggleTopMenu(false)
	})

	termui.Handle("/sys/kbd/C-y", func(e termui.Event) {
		ui.toggleActionBox(false)
	})

	termui.Handle("/sys/kbd/C-o", func(e termui.Event) {
		if _, ok := ui.uilists[uiMessages][ui.selectedPaneMessages]; ok {
			ui.openInTatwebui()
		}
	})

	termui.Handle("/sys/kbd", func(e termui.Event) {
		if !ui.isOnActionBox() {
			ui.switchToActionBox()
		}
		ui.send.BorderFg = termui.ColorRed
		if _, ok := ui.uilists[uiActionBox][0]; ok {
			ui.uilists[uiActionBox][0].list.BorderFg = termui.ColorWhite
		}
		char := e.Data.(termui.EvtKbd).KeyStr
		ui.send.Text += char
		ui.render()
	})

	for _, h := range ui.hooks {
		if h.Shortcut == "" {
			continue
		}
		if _, ok := termui.DefaultEvtStream.Handlers["/sys/kbd/"+h.Shortcut]; ok {
			internal.Exit("Shortcut %s is already used in tatcli", h.Shortcut)
		}
		termui.Handle("/sys/kbd/"+h.Shortcut, func(e termui.Event) {
			ui.RunExec(nil, e.Path, "")
		})
	}

}

func (ui *tatui) moveHistory(direction string) {
	if direction == "up" {
		ui.currentPosHistory--
		if ui.currentPosHistory < 0 {
			ui.currentPosHistory = 0
		}
	} else { // down
		ui.currentPosHistory++
		if ui.currentPosHistory >= len(ui.history) {
			ui.currentPosHistory = len(ui.history) - 1
			ui.send.Text = ""
			ui.render()
			return
		}
	}
	if ui.currentPosHistory >= 0 && ui.currentPosHistory < len(ui.history) {
		ui.send.Text = ui.history[ui.currentPosHistory]
	}
	ui.render()
}

func (ui *tatui) move(direction string) {
	if ui.selectedPane == uiActionBox {
		ui.moveHistory(direction)
		return
	}
	if ui.selectedPane != uiMessages &&
		ui.selectedPane != uiMessage &&
		ui.selectedPane != uiTopics {
		return
	}

	//ui.selectedPane = ui.current
	ui.colorizedPanes()
	if direction == "up" {
		if ui.selectedPane == uiMessages {
			ui.moveUP(ui.uilists[ui.selectedPane][ui.selectedPaneMessages])
		} else {
			ui.moveUP(ui.uilists[ui.selectedPane][0])
		}
	} else {
		if ui.selectedPane == uiMessages {
			ui.moveDown(ui.uilists[ui.selectedPane][ui.selectedPaneMessages])
		} else {
			ui.moveDown(ui.uilists[ui.selectedPane][0])
		}
	}
}

func (ui *tatui) moveUP(uil *uilist) {
	p := uil.position
	uil.position--
	if uil.position < 0 {
		uil.page--
		if uil.page < 0 {
			uil.page = 0
			uil.position = 0
		} else {
			uil.update()
			uil.position = len(uil.list.Items) - 1
		}
	}
	if p != uil.position {
		ui.removeMarker(uil, ui.selectedPaneMessages, p)
		ui.addMarker(uil, ui.selectedPaneMessages)
	}
}

func (ui *tatui) moveDown(uil *uilist) {
	p := uil.position
	uil.position++
	nbPerPage := 1
	if uil.uiType == uiMessages {
		nbPerPage = (ui.getNbPerPage() / len(ui.currentFilterMessages[ui.currentTopic.Topic])) - 1
	} else {
		nbPerPage = ui.getNbPerPage()
	}

	if uil.position >= len(uil.list.Items) && nbPerPage == len(uil.list.Items) {
		uil.page++
		uil.update()
		uil.position = 0
	} else if nbPerPage > len(uil.list.Items) && uil.position >= len(uil.list.Items) {
		uil.position--
	}
	if p != uil.position {
		ui.removeMarker(uil, ui.selectedPaneMessages, p)
		ui.addMarker(uil, ui.selectedPaneMessages)
	}
}

func (ui *tatui) addMarker(uil *uilist, selectedPane int) {
	if uil == nil || uil.list.Items == nil || uil.position < 0 || uil.position >= len(uil.list.Items) {
		return
	}
	if uil.uiType == uiMessages && ui.currentListMessages != nil {
		_, ok := ui.currentListMessages[selectedPane]
		if ok && len(ui.currentListMessages[selectedPane]) > uil.position {
			uil.list.Items[uil.position] = fmt.Sprintf("[%s](bg-green)", ui.formatMessage(ui.currentListMessages[selectedPane][uil.position], false))
		}
	} else if uil.uiType == uiTopics && ui.currentListTopic != nil {
		if len(ui.currentListTopic) > uil.position {
			topic, _ := ui.formatTopic(ui.currentListTopic[uil.position], false)
			uil.list.Items[uil.position] = fmt.Sprintf("[%s](bg-green)", topic)
		}
	} else {
		uil.list.Items[uil.position] = fmt.Sprintf("[➨](fg-green) %s", uil.list.Items[uil.position])
	}
	ui.render()
}

func (ui *tatui) removeMarker(uil *uilist, selectedPane, pos int) {
	if pos < 0 || pos >= len(uil.list.Items) {
		return
	}
	if uil.uiType == uiMessages && ui.currentListMessages != nil {
		uil.list.Items[pos] = ui.formatMessage(ui.currentListMessages[selectedPane][pos], true)
	} else if uil.uiType == uiTopics && ui.currentListTopic != nil {
		topic, _ := ui.formatTopic(ui.currentListTopic[pos], true)
		uil.list.Items[pos] = topic
	} else {
		uil.list.Items[pos] = strings.Replace(uil.list.Items[pos], "[➨](fg-green) ", "", 1)
	}
	ui.render()
}

func (ui *tatui) switchToActionBox() {
	ui.selectedPane = uiActionBox
	ui.colorizedPanes()
}

func (ui *tatui) switchBox() {
	switch ui.current {
	case uiHome, uiResult:
		ui.switchBoxFromHome()
	case uiTopics:
		ui.switchBoxFromTopics()
	case uiMessages:
		ui.switchBoxFromMessages()
	case uiMessage:
		ui.switchBoxFromMessage()
	}
	ui.colorizedPanes()
	termui.Clear()
	ui.render()
}

func (ui *tatui) switchBoxFromHome() {
	ui.selectedPane = uiActionBox
	ui.send.BorderFg = termui.ColorRed
}

func (ui *tatui) switchBoxFromTopics() {
	switch ui.selectedPane {
	case uiActionBox:
		ui.selectedPane = uiTopics
	case uiTopics:
		if _, ok := ui.uilists[uiMessage][0]; ok {
			ui.selectedPane = uiMessage
		} else if _, ok := ui.uilists[uiMessages][0]; ok {
			ui.selectedPane = uiMessages
		} else {
			ui.selectedPane = uiActionBox
		}
	case uiMessages, uiMessage:
		ui.selectedPane = uiActionBox
	}
}

func (ui *tatui) switchBoxFromMessages() {
	switch ui.selectedPane {
	case uiActionBox:
		if len(ui.currentFilterMessages[ui.currentTopic.Topic]) == 1 {
			ui.selectedPane = uiTopics
		} else {
			ui.selectedPane = uiMessages
			ui.selectedPaneMessages = 0
			ui.addMarker(ui.uilists[uiMessages][ui.selectedPaneMessages], ui.selectedPaneMessages)
		}
	case uiTopics:
		ui.selectedPane = uiMessages
		ui.selectedPaneMessages = 0
	case uiMessages:
		if len(ui.currentFilterMessages[ui.currentTopic.Topic]) == 1 {
			ui.selectedPane = uiActionBox
		} else {
			ui.removeMarker(ui.uilists[uiMessages][ui.selectedPaneMessages], ui.selectedPaneMessages, ui.uilists[uiMessages][ui.selectedPaneMessages].position)
			ui.selectedPaneMessages++
			if ui.selectedPaneMessages >= len(ui.currentFilterMessages[ui.currentTopic.Topic]) {
				ui.selectedPaneMessages = 0
				ui.selectedPane = uiActionBox
			}
			if ui.selectedPane == uiMessages {
				ui.addMarker(ui.uilists[uiMessages][ui.selectedPaneMessages], ui.selectedPaneMessages)
			}
		}
	}
}

func (ui *tatui) switchBoxFromMessage() {
	switch ui.selectedPane {
	case uiActionBox:
		ui.selectedPane = uiTopics
	case uiTopics:
		ui.selectedPane = uiMessage
	case uiMessage:
		ui.selectedPane = uiActionBox
	}
}

func (ui *tatui) openLinksInBrowser() {
	msg := ui.currentListMessages[ui.selectedPaneMessages][ui.uilists[uiMessages][ui.selectedPaneMessages].position]
	for _, url := range msg.Urls {
		open.Run(url)
	}
}

func (ui *tatui) toggleUsernames(forceHide bool) {
	ui.toggleActionOnTopic(" /hide-usernames", forceHide)
	ui.updateMessages()
}

func (ui *tatui) openInTatwebui() {
	msg := ui.currentListMessages[ui.selectedPaneMessages][ui.uilists[uiMessages][ui.selectedPaneMessages].position]
	base := strings.TrimSpace(viper.GetString("tatwebui-url"))

	if base == "" {
		ui.msg.Text = "You need to set TatWebui URL in your config file, see /set-tatwebui-url"
		ui.render()
		return
	}

	view := "standardview"

	// search view setted on topic parameters
	for _, p := range ui.currentTopic.Parameters {
		if p.Key == "tatwebui.view.restricted" && p.Value != "" {
			view = strings.TrimSpace(strings.Replace(p.Value, "-list", "", 1))
			break
		} else if p.Key == "tatwebui.view.default" && p.Value != "" {
			view = strings.TrimSpace(strings.Replace(p.Value, "-list", "", 1))
		}
	}
	if !strings.HasSuffix(base, "/") {
		view = "/" + view
	}
	url := fmt.Sprintf("%s%s/list%s?idMessage=%s", base, view, ui.currentTopic.Topic, msg.ID)
	open.Run(url)
}

func (ui *tatui) colorizedPanes() {
	if _, ok := ui.uilists[uiTopics]; ok {
		ui.uilists[uiTopics][0].list.BorderFg = termui.ColorWhite
	}
	if _, ok := ui.uilists[uiMessages]; ok {
		for k := range ui.currentFilterMessages[ui.currentTopic.Topic] {
			ui.uilists[uiMessages][k].list.BorderFg = termui.ColorWhite
		}
	}
	if _, ok := ui.uilists[uiMessage]; ok {
		ui.uilists[uiMessage][0].list.BorderFg = termui.ColorWhite
	}

	ui.send.BorderFg = termui.ColorWhite

	switch ui.selectedPane {
	case uiActionBox:
		ui.send.BorderFg = termui.ColorRed
	case uiTopics, uiMessage:
		ui.uilists[ui.selectedPane][0].list.BorderFg = termui.ColorRed
	case uiMessages:
		ui.uilists[ui.selectedPane][ui.selectedPaneMessages].list.BorderFg = termui.ColorRed
	}
}
