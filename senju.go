package senju

import (
	"context"
	"gopkg.in/fsnotify/fsnotify.v1"
	"log"
	"os"
	"sync"
	"time"
)

type Senju struct {
	eventHandlerMap map[string]*EventHandler
	l *sync.RWMutex
	stopCh chan struct{}
	duration time.Duration
	errHandler func(error)
}

func New() *Senju {
	m := map[string]*EventHandler{}
	return &Senju{
		eventHandlerMap: m,
		l:               &sync.RWMutex{},
		stopCh:          make(chan struct{}, 1),
		duration: 500 * time.Millisecond,
		errHandler: func(err error) {
			log.Println(err)
		},
	}
}

func (s *Senju) SetErrorHandler(f func(error)) {
	s.errHandler = f
}

func (s *Senju) Add(name string, handler *EventHandler) {
	s.l.Lock()
	s.eventHandlerMap[name] = handler
	s.l.Unlock()
}

func (s *Senju) SetDuration(d time.Duration) {
	s.duration = d
}

func (s *Senju) searchFile(watcher *fsnotify.Watcher) error {
	for fileName, _ := range s.eventHandlerMap {
		_, err := os.Stat(fileName)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if err := watcher.Add(fileName); err != nil {
			return err
		}
	}
	return nil
}

func (s *Senju) Close() error {
	s.stopCh<- struct{}{}
	return nil
}

func (s *Senju) Run() error {
	ctx := context.Background()
	return s.RunWithContext(ctx)
}

func (s *Senju) RunWithContext(ctx context.Context) error {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	defer watcher.Close()

	f := func(ctx context.Context, eventHandler *EventHandler, event fsnotify.Event, name EventName) {
		handler, ok := eventHandler.GetHandler(name)
		if ok {
			go func(){
				if err := handler(event)(ctx); err != nil {
					s.errHandler(err)
				}
			}()
		}

	}

	ticker := time.NewTicker(s.duration)
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func(ctx2 context.Context) {
		defer wg.Done()
		for {
			select {
			case <-ticker.C:
				if err := s.searchFile(watcher); err != nil {
					s.errHandler(err)
				}

			case <-ctx2.Done():
				return
			}
		}
	}(ctx)

	wg.Add(1)

	go func(ctx2 context.Context){
		defer wg.Done()
		for {
			select {
			case event := <-watcher.Events:
				target, ok := s.eventHandlerMap[event.Name]
				if !ok {
					continue
				}

				var eventName EventName
				if event.Op&fsnotify.Create == fsnotify.Create {
					eventName = Create
				} else if event.Op&fsnotify.Write == fsnotify.Write {
					eventName = Write
				} else if event.Op&fsnotify.Rename == fsnotify.Rename {
					eventName = Rename
				} else if event.Op&fsnotify.Remove == fsnotify.Remove {
					eventName = Remove
				} else if event.Op&fsnotify.Chmod == fsnotify.Chmod {
					eventName = Chmod
				}

				f(ctx2, target, event, eventName)

			case err := <-watcher.Errors:
				s.errHandler(err)

			case <-ctx2.Done():
				return
			}
		}
	}(ctx)

	wg.Add(1)
	<-s.stopCh
	cancel()
	wg.Wait()
	return nil
}
