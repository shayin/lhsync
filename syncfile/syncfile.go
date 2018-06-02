package syncfile

import (
	"time"
	"os"
)

type Op uint32

const (
	Create Op = 1 << iota
	Write
	Remove
	Rename
	Chmod
)

type SyncFile struct {
	FileName string
	FileSize int64
	FileMt   time.Time
	FileMode os.FileMode
	FileMd5  string
	FileType bool
	FileOp   int
}

type SyncPacket struct {
	Header  *SyncFile
	Content []byte
}

func (sp *SyncPacket) Write(p []byte) (int, error) {
	sp.Content = append(sp.Content, p...)
	return len(p), nil
}

func (sp *SyncPacket) Read(p []byte) (int, error) {
	n := copy(p, sp.Content)
	return n, nil
}
