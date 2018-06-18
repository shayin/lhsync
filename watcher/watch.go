package watcher

import (
	"github.com/fsnotify/fsnotify"
	"github.com/ngaut/log"
	"path/filepath"
	"os"
	"github.com/shayin/lhsync/syncfile"
	"github.com/shayin/lhsync/config"
	"github.com/shayin/lhsync/pb"
	"fmt"
)

var Watcher *fsnotify.Watcher

func NewWatcher() error {
	var err error
	Watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	return nil
}

func AddWatchRecursive(watchdir map[string]string) error {
	for name, path := range watchdir {
		err := filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
			if f == nil {
				return err
			}
			if f.IsDir() {
				log.Infof("watch: %s", path)
				err = Watcher.Add(path)
				if err != nil {
					log.Fatal(err)
				}
				return nil
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("%s: %s", name, err)
		}
	}
	return nil
}

func WatchAndProcess() error {
	syncCom, err := syncfile.NewSyncCom(config.CConf.Dest)
	if err != nil {
		return err
	}

	defer syncCom.Close()

	for {
		select {
		case event := <-Watcher.Events:
			log.Infof("event: %+v", event)
			fileData, err := getSyncMsg(event)
			if err != nil {
				log.Errorf("process event error: %s", err.Error())
			} else {
				resp, err := syncCom.SyncToServer(fileData)
				if err != nil {
					log.Errorf("send sync info to server error: %s", err.Error())
				} else {
					log.Infof("sync result: %s", resp.Msg)
				}
			}
		case err := <-Watcher.Errors:
			log.Errorf("error:", err)
		}
	}
	return nil
}

func getSyncMsg(event fsnotify.Event) (*lhsync_pb.FileData, error) {
	absPath := event.Name
	pathKey, err := config.GetClientPathKey(event.Name)
	if err != nil {
		return nil, err
	}

	relPath, err := filepath.Rel(config.CConf.WatchDir[pathKey], event.Name)
	if err != nil {
		return nil, err
	}
	fh := syncfile.FileHandler{
		PathKey: pathKey,
		RelPath: relPath,
		FOp:     uint32(event.Op),
	}

	switch event.Op {
	case fsnotify.Create:
		isFile, err := fh.CreateReq(absPath)
		if err != nil {
			return nil, err
		}
		if !isFile {
			if err := Watcher.Add(absPath); err != nil {
				return nil, err
			}
		}
	case fsnotify.Chmod:
		if err := fh.ChmodReq(absPath); err != nil {
			return nil, err
		}
	case fsnotify.Remove:
		if err := fh.RemoveReq(absPath); err != nil {
			return nil, err
		}
	case fsnotify.Rename:
		if err := fh.RenameReq(absPath); err != nil {
			return nil, err
		}
	case fsnotify.Write:
		if err := fh.WriteReq(absPath); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown event handler: %s", event.Op.String())
	}

	pbFh := lhsync_pb.FileData(fh)
	return &pbFh, nil
}
