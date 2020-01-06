package main

import (
	"sync"

	"github.com/ovh/cds/tools/smtpmock"
)

var (
	allMessages      = make(map[string][]smtpmock.Message)
	allMessagesMutex sync.Mutex
	messagesCounter  int
	sessions         []string
	sessionsMutex    sync.Mutex
)

func StoreAddMessage(m smtpmock.Message) {
	allMessagesMutex.Lock()
	defer allMessagesMutex.Unlock()

	list := allMessages[m.To]
	list = append([]smtpmock.Message{m}, list...)
	allMessages[m.To] = list
	messagesCounter++
}

func StoreGetMessages() []smtpmock.Message {
	ms := []smtpmock.Message{}
	for k := range allMessages {
		ms = append(ms, allMessages[k]...)
	}
	return ms
}

func StoreGetRecipientMessages(addr string) []smtpmock.Message {
	if ms, ok := allMessages[addr]; ok && len(ms) > 0 {
		return ms
	}
	return []smtpmock.Message{}
}

func StoreCountMessages() int {
	return messagesCounter
}

func StoreCountRecipients() int {
	return len(allMessages)
}

func StoreAddSession(sessionID string) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	sessions = append(sessions, sessionID)
}
