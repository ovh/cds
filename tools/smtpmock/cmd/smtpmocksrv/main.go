package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/fsamin/smtp"
	"github.com/ovh/cds/tools/smtpmock"

	"github.com/labstack/echo"
)

type Message struct {
	FromAgent     string
	RemoteAddress string
	User          string
	From          string
	To            string
	Content       string
}

var (
	flagSMTPAddress  string
	flagHTTPAddress  string
	allMessages      = make(map[string][]Message)
	allMessagesMutex sync.Mutex
	messagesCounter  int
)

func init() {
	flag.StringVar(&flagSMTPAddress, "smtp-address", ":2023", "SMTP Server address")
	flag.StringVar(&flagHTTPAddress, "http-address", ":2024", "HTTP Server address")
}

func main() {
	flag.Parse()

	go func() {
		if err := smtpmock.StartServer(
			context.Background(),
			flagSMTPAddress,
			smtpmock.Handle("*@*", smtpHandler),
		); err != nil {
			log.Fatal(err)
		}
	}()

	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		var s = fmt.Sprintf("SMTP Server listenning on %s\n", flagSMTPAddress)
		s += fmt.Sprintf("%d mails received to %d recipents\n", messagesCounter, len(allMessages))

		return c.String(http.StatusOK, s)
	})
	e.GET("/messages", func(c echo.Context) error {
		return c.JSON(http.StatusOK, allMessages)
	})
	e.GET("/messages/:recipent", func(c echo.Context) error {
		return c.JSON(http.StatusOK, allMessages[c.Param("recipent")])
	})
	e.GET("/messages/:recipent/latest", func(c echo.Context) error {
		messages := allMessages[c.Param("recipent")]
		if len(messages) == 0 {
			return c.JSON(http.StatusNotFound, "not found")
		}

		return c.JSON(http.StatusOK, messages[0])
	})

	e.Logger.Fatal(e.Start(flagHTTPAddress))

}

func smtpHandler(envelope *smtp.Envelope) error {
	allMessagesMutex.Lock()
	defer allMessagesMutex.Unlock()

	list := allMessages[envelope.MessageTo]

	m := Message{
		RemoteAddress: envelope.RemoteAddr,
		FromAgent:     envelope.FromAgent,
		To:            envelope.MessageTo,
		From:          envelope.MessageFrom,
		User:          envelope.User,
	}

	btes, err := ioutil.ReadAll(envelope.MessageData)
	if err != nil {
		return err
	}

	m.Content = string(btes)

	list = append([]Message{m}, list...)
	allMessages[envelope.MessageTo] = list
	messagesCounter++
	return nil
}
