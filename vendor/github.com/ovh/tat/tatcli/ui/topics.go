package ui

import (
	"fmt"
	"time"

	"github.com/gizak/termui"
	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/viper"
)

func (ui *tatui) showUnreadTopics() {
	ui.current = uiTopics
	ui.onlyFavorites = "false"
	ui.onlyUnread = true
	ui.showTopics()
}

func (ui *tatui) showFavoritesTopics() {
	ui.current = uiTopics
	ui.onlyFavorites = "true"
	ui.onlyUnread = false
	ui.showTopics()
}

func (ui *tatui) showTopics() {
	ui.current = uiTopics
	ui.selectedPane = uiTopics
	ui.send.BorderLabel = " ✎ Action "
	ui.msg.Text = "tab or enter to select topic"

	termui.Body.Rows = nil

	ui.initTopics()
	ui.updateTopics()

	go func() {
		for {
			if ui.current != uiTopics {
				break
			}
			time.Sleep(10 * time.Second)
			mutex.Lock()
			ui.updateTopics()
			mutex.Unlock()
		}
	}()

	ui.uilists[uiTopics][0].list.BorderRight = false

	ui.prepareTopMenu()

	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(12, 0, ui.uilists[uiTopics][0].list),
		),
	)
	ui.prepareSendRow()
	termui.Clear()
	ui.colorizedPanes()
	ui.render()
}

func (ui *tatui) initTopics() {
	strs := []string{"[Loading...](fg-black,bg-white)"}

	ls := termui.NewList()
	ls.BorderTop, ls.BorderLeft, ls.BorderRight, ls.BorderBottom = true, false, false, false
	ls.Items = strs
	ls.ItemFgColor = termui.ColorWhite
	if ui.onlyFavorites == "true" {
		ls.BorderLabel = " ★ Favorites Topics ★ "
	} else if ui.onlyUnread {
		ls.BorderLabel = " ✉ Unread Topics ✉ "
	} else {
		ls.BorderLabel = " All Topics "
	}

	ls.Width = 25
	ls.Y = 0
	ls.Height = termui.TermHeight() - uiHeightTop - uiHeightSend

	m := make(map[int]*uilist)
	m[0] = &uilist{uiType: uiTopics, list: ls, position: 0, page: 0, update: ui.updateTopics}
	ui.uilists[uiTopics] = m
}

func (ui *tatui) enterTopic() {
	ui.currentTopic = ui.currentListTopic[ui.uilists[uiTopics][0].position]
	ui.showMessages()
}

func (ui *tatui) updateTopics() {
	start := time.Now().UnixNano()
	nbPerPage := ui.getNbPerPage()

	c := ui.currentFilterTopics
	if c == nil {
		c = &tat.TopicCriteria{}
	}
	if ui.onlyUnread {
		c.Skip = 0
		c.Limit = 1000
	} else {
		c.Skip = ui.uilists[uiTopics][0].page * nbPerPage
		c.Limit = nbPerPage
	}

	c.OnlyFavorites = ui.onlyFavorites
	c.GetNbMsgUnread = "true"

	topicsJSON, err := internal.Client().TopicList(c)
	if err != nil {
		ui.msg.Text = err.Error()
		return
	}
	var strs []string
	topics := []tat.Topic{}
	ui.currentListUnreadTopics = topicsJSON.TopicsMsgUnread
	privateTasksTopic := fmt.Sprintf("/Private/%s/Tasks", viper.GetString("username"))
	for _, topic := range topicsJSON.Topics {
		if topic.Topic == privateTasksTopic {
			continue
		}

		topicLine, isUnread := ui.formatTopic(topic, true)
		if !ui.onlyUnread || isUnread {
			strs = append(strs, topicLine)
			topics = append(topics, topic)
		}
	}
	if ui.onlyUnread {
		max := nbPerPage - 1
		if len(topics) < nbPerPage {
			max = len(topics) - 1
		}
		if ui.uilists[uiTopics][0].page*nbPerPage > len(topics) {
			return
		}
		ui.currentListTopic = topics[ui.uilists[uiTopics][0].page*nbPerPage : max]
		ui.uilists[uiTopics][0].list.Items = strs[ui.uilists[uiTopics][0].page*nbPerPage : max]
	} else {
		ui.currentListTopic = topics
		ui.uilists[uiTopics][0].list.Items = strs
	}

	ui.msg.Text = ""
	ui.addMarker(ui.uilists[uiTopics][0], 0)
	delta := int64((time.Now().UnixNano() - start) / 1000000)
	ui.lastRefresh.Text = fmt.Sprintf("%s (%dms)", time.Now().Format(time.Stamp), delta)
	ui.render()
}

func (ui *tatui) formatTopic(topic tat.Topic, withColor bool) (string, bool) {
	textUnread := ""
	for topicName, unread := range ui.currentListUnreadTopics {
		if topic.Topic == topicName && unread > 0 {
			if withColor {
				textUnread += fmt.Sprintf(" [✉](fg-red)")
			} else {
				textUnread += fmt.Sprintf(" ✉ ")
			}
			break
		}
	}

	return topic.Topic + textUnread, textUnread != ""
}
