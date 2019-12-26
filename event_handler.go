package senju

import (
	"context"
	"gopkg.in/fsnotify/fsnotify.v1"
	"sync"
)

type EventName = string

const (
	Create EventName = "Create"
	Write EventName = "Write"
	Rename EventName = "Rename"
	Remove EventName = "Remove"
	Chmod EventName = "Chmod"
)

type Handler func(fsnotify.Event) func(context.Context) error

type EventHandler struct {
	handlerMap map[EventName]Handler
	l *sync.RWMutex
}

func NewEventHandler() *EventHandler {
	m := map[EventName]Handler{}
	return &EventHandler{m, &sync.RWMutex{}}
}

func (e *EventHandler) SetHandler(handler Handler, name EventName) {
	e.l.Lock()
	e.handlerMap[name] = handler
	e.l.Unlock()
}

func (e *EventHandler) GetHandler(name EventName) (Handler, bool) {
	e.l.RLock()
	defer e.l.RUnlock()
	h, ok := e.handlerMap[name]
	return h, ok
}