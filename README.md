# senju

file watch event handle framework using fsnotify

## usage

``` go
package main

import (
	"context"
	"fmt"
	"github.com/wano/senju"
	"gopkg.in/fsnotify/fsnotify.v1"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	sut := senju.New()
	eventHandler := senju.NewEventHandler()
	eventHandler.SetHandler(func(event fsnotify.Event) func(context.Context) error {
		return func(ctx context.Context) error {
			fmt.Println(event)
			return nil
		}
	}, senju.Rename) //watch Rename event
	sut.Add("/tmp/senju", eventHandler)

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL, syscall.SIGQUIT)

	wg := &sync.WaitGroup{}
	go func(){
		defer wg.Done()
		if err := sut.Run(); err != nil {
			log.Println(err)
		}
	}()
	wg.Add(1)
	<-sigCh
	sut.Close()
	wg.Wait()
}

```