package main

import (
	"crypto/tls"
	"github.com/mattn/go-gtk/gtk"
	"github.com/mattn/go-xmpp"
	"log"
	"os"
	"strings"
)

func main() {
	gtk.Init(&os.Args)

	window := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	window.SetTitle("GoTalk")
	window.Connect("destroy", func() {
		gtk.MainQuit()
	})
	vbox := gtk.NewVBox(false, 1)
	scrolledwin := gtk.NewScrolledWindow(nil, nil)
	textview := gtk.NewTextView()
	textview.SetEditable(false)
	textview.SetCursorVisible(false)
	scrolledwin.Add(textview)
	vbox.Add(scrolledwin)

	buffer := textview.GetBuffer()

	entry := gtk.NewEntry()
	vbox.PackEnd(entry, false, false, 0)

	window.Add(vbox)
	window.SetSizeRequest(300, 400)
	window.ShowAll()

	dialog := gtk.NewDialog()
	dialog.SetTitle(window.GetTitle())
	sgroup := gtk.NewSizeGroup(gtk.SIZE_GROUP_HORIZONTAL)

	hbox := gtk.NewHBox(false, 1)
	dialog.GetVBox().Add(hbox)
	label := gtk.NewLabel("username:")
	sgroup.AddWidget(label)
	hbox.Add(label)
	username := gtk.NewEntry()
	hbox.Add(username)

	hbox = gtk.NewHBox(false, 1)
	dialog.GetVBox().Add(hbox)
	label = gtk.NewLabel("password:")
	sgroup.AddWidget(label)
	hbox.Add(label)
	password := gtk.NewEntry()
	password.SetVisibility(false)
	hbox.Add(password)

	dialog.AddButton(gtk.STOCK_OK, gtk.RESPONSE_OK)
	dialog.AddButton(gtk.STOCK_CANCEL, gtk.RESPONSE_CANCEL)
	dialog.SetDefaultResponse(gtk.RESPONSE_OK)
	dialog.SetTransientFor(window)
	dialog.ShowAll()
	res := dialog.Run()
	username_ := username.GetText()
	password_ := password.GetText()
	dialog.Destroy()
	if res != gtk.RESPONSE_OK {
		os.Exit(0)
	}

	xmpp.DefaultConfig = tls.Config{
		ServerName:         "talk.google.com",
		InsecureSkipVerify: false,
	}

	talk, err := xmpp.NewClient("talk.google.com:443", username_, password_, false)
	if err != nil {
		log.Fatal(err)
	}

	entry.Connect("activate", func() {
		text := entry.GetText()
		tokens := strings.SplitN(text, " ", 2)
		if len(tokens) == 2 {
			func() {
				defer recover()
				talk.Send(xmpp.Chat{Remote: tokens[0], Type: "chat", Text: tokens[1]})
				entry.SetText("")
			}()
		}
	})

	go func() {
		for {
			func() {
				defer recover()
				chat, err := talk.Recv()
				if err != nil {
					log.Fatal(err)
				}

				var iter gtk.TextIter
				buffer.GetStartIter(&iter)
				if msg, ok := chat.(xmpp.Chat); ok {
					buffer.Insert(&iter, msg.Remote+": "+msg.Text+"\n")
				}
			}()
		}
	}()

	gtk.Main()
}
