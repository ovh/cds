package store

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

func AddMessage(m sdk.Message) {
	allMessagesMutex.Lock()
	defer allMessagesMutex.Unlock()

	list := allMessages[m.To]
	list = append([]sdk.Message{m}, list...)
	allMessages[m.To] = list
	messagesCounter++
}

func GetMessages() []sdk.Message {
	ms := []sdk.Message{}
	for k := range allMessages {
		ms = append(ms, allMessages[k]...)
	}
	return ms
}

func GetRecipientMessages(addr string) []sdk.Message {
	if ms, ok := allMessages[addr]; ok && len(ms) > 0 {
		return ms
	}
	return []sdk.Message{}
}

func CountMessages() int {
	return messagesCounter
}

func CountRecipients() int {
	return len(allMessages)
}

func AddSession(sessionID string) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	sessions = append(sessions, sessionID)
}
