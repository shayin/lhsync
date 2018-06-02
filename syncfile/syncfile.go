package syncfile

import (
	"time"
	"os"
	"github.com/fsnotify/fsnotify"
)

type SyncFile struct {
	FileName string
	FileSize int64
	FileMt   time.Time
	FileMode os.FileMode
	FileMd5  string
	FileType bool
	FileOp   fsnotify.Op

	Content []byte
}

func (sf *SyncFile) Write(p []byte) (int, error) {
	sf.Content = append(sf.Content, p...)
	return len(p), nil
}

func (sf *SyncFile) Read(p []byte) (int, error) {
	n := copy(p, sf.Content)
	return n, nil
}
