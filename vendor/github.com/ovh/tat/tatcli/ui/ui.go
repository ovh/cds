package ui

import (
	"strings"
	"sync"
	"time"

	"github.com/gizak/termui"
	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/config"
	"github.com/ovh/tat/tatcli/internal"
)

var mutex = &sync.Mutex{}

// tatui wrapper designed for dashboard creation
type tatui struct {
	header      *termui.Par
	msg         *termui.Par
	lastRefresh *termui.Par
	homeLeft    *termui.Par
	homeRight   *termui.Par
	send        *termui.Par

	selectedPane         int
	selectedPaneMessages int
	current              int

	onlyFavorites string
	onlyUnread    bool

	currentListTopic          []tat.Topic
	currentListUnreadTopics   map[string]int
	currentTopic              tat.Topic
	currentListMessages       map[int][]tat.Message
	currentMessage            tat.Message
	currentFilterTopics       *tat.TopicCriteria
	currentFilterMessages     map[string]map[int]*tat.MessageCriteria
	currentFilterMessagesText map[string]map[int]string
	currentModeOnTopic        map[string]string
	history                   []string
	currentPosHistory         int
	uiTopicCommands           map[string]string
	firstCallMessages         bool
	hooks                     []config.Hook

	uilists map[int]map[int]*uilist
}

type uilist struct {
	uiType         int
	list           *termui.List
	position, page int
	update         func()
}

const (
	uiTopics int = iota
	uiMessages
	uiMessage
	uiHome
	uiResult
	uiActionBox
)

var (
	uiHeightTop       = 1
	uiHeightSend      = 2
	currentPosHistory = 0
)

const (
	hexColorGreen = "#14892c"
	hexColorRed   = "#d04437"
	hexColorBlue  = "#5484ed"
)

func (ui *tatui) init(args []string) {
	if err := termui.Init(); err != nil {
		panic(err)
	}

	// do not display log in ui mode
	tat.DebugLogFunc = func(string, ...interface{}) {}
	tat.ErrorLogFunc = func(string, ...interface{}) {}

	ui.firstCallMessages = false
	ui.uilists = make(map[int]map[int]*uilist)
	ui.currentFilterMessages = make(map[string]map[int]*tat.MessageCriteria)
	ui.currentFilterMessagesText = make(map[string]map[int]string)
	ui.currentListMessages = make(map[int][]tat.Message)
	ui.currentModeOnTopic = make(map[string]string)
	ui.uiTopicCommands = make(map[string]string)
	ui.history = []string{}

	commands := ui.loadArgs(args)
	ui.loadConfig()
	ui.initHeader()
	ui.initMsg()
	ui.initLastRefresh()
	ui.initHome()
	ui.initSend()
	if ui.currentTopic.Topic != "" {
		ui.showTopics()
		ui.updateTopics()
		ui.showMessages()
		for !ui.firstCallMessages {
			time.Sleep(1 * time.Second)
		}
		ui.execCommands(commands)
	} else {
		ui.showHome()
	}

	ui.initHandles()
}

func (ui *tatui) render() {
	termui.Render(termui.Body)
}

func (ui *tatui) draw(i int) {
	termui.Body.Align()
	termui.Render(termui.Body)
}

func (ui *tatui) initHeader() {
	p := termui.NewPar("TAT ➠ topics: (f)avorites - un(r)ead - (a)ll | (h)ome | (q)uit")
	p.Height = uiHeightTop
	p.TextFgColor = termui.ColorWhite
	p.Border = false
	ui.header = p
}

func (ui *tatui) getNbPerPage() int {
	if _, ok := ui.uilists[uiTopics]; ok {
		return ui.uilists[uiTopics][0].list.Height - 1
	}
	return 1
}

func (ui *tatui) isOnActionBox() bool {
	return ui.selectedPane == uiActionBox
}

func (ui *tatui) initMsg() {
	p := termui.NewPar("")
	p.Height = uiHeightTop
	p.TextFgColor = termui.ColorWhite
	p.BorderTop, p.BorderLeft, p.BorderRight, p.BorderBottom = false, false, false, false
	ui.msg = p
}

func (ui *tatui) initLastRefresh() {
	p := termui.NewPar("")
	p.Height = uiHeightTop
	p.TextFgColor = termui.ColorWhite
	p.BorderTop, p.BorderLeft, p.BorderRight, p.BorderBottom = false, false, false, false
	ui.lastRefresh = p
}

func (ui *tatui) prepareTopMenu() {
	if !strings.Contains(ui.uiTopicCommands[ui.currentTopic.Topic], " /hide-top") {
		termui.Body.AddRows(
			termui.NewRow(
				termui.NewCol(4, 0, ui.header),
				termui.NewCol(6, 0, ui.msg),
				termui.NewCol(2, 0, ui.lastRefresh),
			),
		)
	}
}

func (ui *tatui) prepareSendRow() {
	if strings.Contains(ui.uiTopicCommands[ui.currentTopic.Topic], " /hide-bottom") {
		return
	}
	if !strings.Contains(ui.uiTopicCommands[ui.currentTopic.Topic], " /hide-top") {
		termui.Body.AddRows(
			termui.NewRow(
				termui.NewCol(12, 0, ui.send),
			),
		)
	} else {
		termui.Body.AddRows(
			termui.NewRow(
				termui.NewCol(5, 0, ui.send),
				termui.NewCol(5, 0, ui.msg),
				termui.NewCol(2, 0, ui.lastRefresh),
			),
		)
	}
}

func (ui *tatui) toggleActionBox(forceHide bool) {
	ui.toggleActionOnTopic(" /hide-bottom", forceHide)

	if !strings.Contains(ui.uiTopicCommands[ui.currentTopic.Topic], " /hide-bottom") {
		uiHeightSend = 2
	} else {
		uiHeightSend = 0
	}
	ui.reloadCurrent()
}

func (ui *tatui) toggleTopMenu(forceHide bool) {
	ui.toggleActionOnTopic(" /hide-top", forceHide)

	if !strings.Contains(ui.uiTopicCommands[ui.currentTopic.Topic], " /hide-top") {
		uiHeightTop = 1
		ui.msg.BorderTop = false
		ui.lastRefresh.BorderTop = false
		ui.msg.Height = uiHeightTop
		ui.lastRefresh.Height = uiHeightTop
	} else {
		uiHeightTop = 0
		ui.msg.BorderTop = true
		ui.lastRefresh.BorderTop = true
		ui.msg.Height = uiHeightSend
		ui.lastRefresh.Height = uiHeightSend
	}
	ui.reloadCurrent()
}

func (ui *tatui) toggleActionOnTopic(action string, forceHide bool) {
	if forceHide {
		if !strings.Contains(ui.uiTopicCommands[ui.currentTopic.Topic], action) {
			ui.uiTopicCommands[ui.currentTopic.Topic] += action
		}
	} else {
		if strings.Contains(ui.uiTopicCommands[ui.currentTopic.Topic], action) {
			ui.uiTopicCommands[ui.currentTopic.Topic] = strings.Replace(ui.uiTopicCommands[ui.currentTopic.Topic], action, "", -1)
		} else {
			ui.uiTopicCommands[ui.currentTopic.Topic] += action
		}
	}
}

func (ui *tatui) reloadCurrent() {
	switch ui.current {
	case uiMessage:
		ui.showMessage()
	case uiMessages:
		ui.showMessages()
	case uiHome, uiResult:
		ui.showHome()
	case uiTopics:
		ui.showTopics()
	}
	ui.render()
}

// arrayContains return true if element is in array
func arrayContains(array []string, element string) bool {
	for _, cur := range array {
		if cur == element {
			return true
		}
	}
	return false
}

// loadArgs load args form command line and return topic, filter, command
func (ui *tatui) loadArgs(args []string) []string {
	// example :
	// /YourTopic/SubTopic /split label:open label:doing label:done /save
	// /YourTopic/SubTopic /run CD
	// /YourTopic/SubTopic /run CD /save
	// /YourTopic/SubTopic /monitoring /save
	if len(args) < 1 {
		return []string{}
	}

	topicName := ""
	if strings.HasPrefix(args[0], "/") {
		topicName = args[0]
	}
	c := &tat.TopicCriteria{Topic: strings.TrimSpace(topicName)}

	//topicsB, err := internal.Request("GET", 200, topic.TopicsListURL(c), nil)
	topicsJSON, err := internal.Client().TopicList(c)
	if err != nil {
		internal.Exit("Error while loading topic %s error:%s", args[0], err.Error())
	}

	if len(topicsJSON.Topics) != 1 {
		internal.Exit("Args on tatcli ui should begin with topic name. Please check it on %s", args[0])
	}

	ui.currentTopic = topicsJSON.Topics[0]

	commands := []string{}
	cmd := ""
	for i := 1; i < len(args); i++ {
		if strings.HasPrefix(args[i], "/") {
			if cmd != "" {
				commands = append(commands, cmd)
			}
			cmd = args[i]
		} else {
			cmd += " " + args[i]
		}
	}
	if cmd != "" {
		commands = append(commands, cmd)
	}
	return commands
}

func (ui *tatui) execCommands(commands []string) {
	for i := 0; i < len(commands); i++ {
		ui.send.Text = commands[i]
		ui.processCmd()
	}
}
