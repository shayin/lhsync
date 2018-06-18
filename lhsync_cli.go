package main

import (
	"flag"
	"github.com/ngaut/log"
	"github.com/shayin/lhsync/config"
	"os/signal"
	"syscall"
	"os"
	"github.com/shayin/lhsync/watcher"
)

var (
	cConfFile = flag.String("c", "server.yaml", "conf file")
)

func main() {
	flag.Parse()

	if err := config.InitClientConfig(*cConfFile); err != nil {
		log.Fatalf("init config fail: %s", err.Error())
	}

	if err := watcher.NewWatcher(); err != nil {
		log.Fatalf("new watcher fail: ", err.Error())
	}

	if err := watcher.AddWatchRecursive(config.CConf.WatchDir); err != nil {
		log.Fatalf("add watch path fail: %s", err.Error())
	}

	if err := watcher.WatchAndProcess(); err != nil {
		log.Fatalf("watch fail: %s", err.Error())
	}

	signChan := make(chan os.Signal, 1)
	doneChan := make(chan bool)
	signal.Notify(signChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for s := range signChan {
			log.Infof("catch signal %+v", s)
			doneChan <- true
		}
	}()
	<-doneChan
}
