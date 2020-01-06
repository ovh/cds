package main

import (
	"sync"

	"github.com/ovh/cds/tools/smtpmock/sdk"
)

var (
	allMessages      = make(map[string][]sdk.Message)
	allMessagesMutex sync.Mutex
	messagesCounter  int
	sessions         []string
	sessionsMutex    sync.Mutex
)

func StoreAddMessage(m sdk.Message) {
	allMessagesMutex.Lock()
	defer allMessagesMutex.Unlock()

	list := allMessages[m.To]
	list = append([]sdk.Message{m}, list...)
	allMessages[m.To] = list
	messagesCounter++
}

func StoreGetMessages() []sdk.Message {
	ms := []sdk.Message{}
	for k := range allMessages {
		ms = append(ms, allMessages[k]...)
	}
	return ms
}

func StoreGetRecipientMessages(addr string) []sdk.Message {
	if ms, ok := allMessages[addr]; ok && len(ms) > 0 {
		return ms
	}
	return []sdk.Message{}
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
