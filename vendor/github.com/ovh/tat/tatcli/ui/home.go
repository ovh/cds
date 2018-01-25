package ui

import (
	"github.com/gizak/termui"
	"github.com/spf13/viper"

	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
)

func (ui *tatui) showHome() {
	ui.current = uiHome
	ui.selectedPane = uiActionBox
	termui.Body.Rows = nil

	ui.prepareTopMenu()

	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(5, 0, ui.homeLeft),
			termui.NewCol(7, 0, ui.homeRight),
		),
	)
	ui.prepareSendRow()
	termui.Clear()
	ui.colorizedPanes()
	ui.render()
}

func (ui *tatui) initHome() {
	ui.initHomeLeft()
	ui.initHomeRight()
}

func (ui *tatui) initHomeLeft() {

	textURL := viper.GetString("url")
	if textURL == "" {
		textURL = "[Invalid URL, please check your config file](fg-red)"
	}

	p := termui.NewPar(`                            TEXT AND TAGS
            ----------------------------------------------
            ----------------------------------------------
                     |||                     |||
                     |||                     |||
                     |||         |||         |||
                     |||         |||         |||
                     |||                     |||
                     |||         |||         |||
                     |||         |||         |||
                     |||                     |||
                     |||                     |||

                       Tatcli Version: ` + tat.Version + `
                    https://github.com/ovh/tat/tatcli
                TAT Engine: https://github.com/ovh/tat
								Current Tat Engine: ` + textURL + `
								Current config file: ` + internal.ConfigFile + `
 Shortcuts:
 - Ctrl + a to view all topics. Cmd /topics in send box
 - Ctrl + b to go back to messsages list, after selected a message
 - Ctrl + c clears filters and UI on current messages list
 - Ctrl + f to view favorites topics. Cmd /favorites
 - Ctrl + h to go back home. Cmd /home or /help
 - Ctrl + t hide or show top menu. Cmd /toggle-top
 - Ctrl + y hide or show actionbox menu. Cmd /toggle-bottom
 - Ctrl + o open current message on tatwebui with a browser. Cmd /open
	          Use option tatwebui-url in config file. See /set-tatwebui-url
 - Ctrl + p open links in current message with a browser. Cmd /open-links
 - Ctrl + j / Ctrl + k (for reverse action):
	    if mode run is enabled, set a msg from open to doing,
	        from doing to done from done to open.
	    if mode monitoring is enabled, set a msg from UP to AL,
	        from AL to UP.
 - Ctrl + q to quit. Cmd /quit
 - Ctrl + r to view unread topics. Cmd /unread
 - Ctrl + u display/hide usernames in messages list. Cmd /toggle-usernames
 - UP / Down to move into topics & messages list
 - UP / Down to navigate through history of action box
 - <tab> to go to next section on screen`)

	p.Height = termui.TermHeight() - uiHeightTop - uiHeightSend
	p.TextFgColor = termui.ColorWhite
	p.BorderTop = true
	p.BorderLeft = false
	p.BorderBottom = false
	ui.homeLeft = p
}

func (ui *tatui) initHomeRight() {
	p := termui.NewPar(`Action Box

  Keywords:
   - /help display this page
   - /me show information about you
   - /version to show tatcli and engine version

  On messages list:
   - /label eeeeee yourLabel to add a label on selected message
   - /unlabel yourLabel to remove label "yourLabel" on selected message
   - /voteup, /votedown, /unvoteup, /unvotedown to vote up or down, or remove vote
   - /task, /untask to add or remove selected message as a personal task
   - /like, /unlike to add or remove like on selected message
   - /filter label:labelA,labelB andtag:tag,tagb
   - /mode (run|monitoring): enable Ctrl + l shortcut, see on left side for help
   - /codereview splits screen into fours panes:
     label:OPENED label:APPROVED label:MERGED label:DECLINED
   - /monitoring splits screen into three panes: label:UP, label:AL, notlabel:AL,UP
     This is the same as two commands:
      - /split label:UP label:AL notlabel:AL,UP
      - /mode monitoring
   - /run <tag> splits screen into three panes: label:open, label:doing, label:done
     /run AA,BB is the same as two commands:
      - /split tag:AA,BB;label:open tag:AA,BB;label:doing tag:AA,BB;label:done
      - /mode run
   - /set-tatwebui-url <urlOfTatWebUI> sets tatwebui-url in tatcli config file. This
      url is used by Ctrl + o shortcut to open message with a tatwebui instance.
   - /split <criteria> splits screen with one section per criteria delimited by space, ex:
      /split label:labelA label:labelB label:labelC
      /split label:labelA,labelB andtag:tag,tagb
      /split tag:myTag;label:labelA,labelB andtag:tag,tagb;label:labelC
   - /save saves current filters in tatcli config file
   - /toggle-usernames displays or hides username in messages list

  For /split and /filter, see all parameters on https://github.com/ovh/tat#parameters

  On topics list, ex:
   - /filter topic:/Private/firstname.lastname
  			see all parameters on https://github.com/ovh/tat#parameters-4

`)

	p.Height = termui.TermHeight() - uiHeightTop - uiHeightSend
	p.TextFgColor = termui.ColorWhite
	p.BorderTop = true
	p.BorderLeft = false
	p.BorderRight = false
	p.BorderBottom = false
	ui.homeRight = p
}
