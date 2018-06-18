package syncfile

import (
	"github.com/shayin/lhsync/pb"
	"os"
	"github.com/ngaut/log"
	"crypto/md5"
	"io"
	"fmt"
)

// create file => CREATE + CHMOD
// create dir => CREATE

type FileHandler lhsync_pb.FileData

func (fh *FileHandler) Write(p []byte) (int, error) {
	fh.FContent = append(fh.FContent, p...)
	return len(p), nil
}

func (fh *FileHandler) Read(p []byte) (int, error) {
	n := copy(p, fh.FContent)
	return n, nil
}

func (fh *FileHandler) CreateReq(absPath string) (isFile bool, err error) {
	s, err := os.Stat(absPath)
	if err != nil {
		return
	}
	fh.FType = !s.IsDir()
	isFile = fh.FType
	fh.FSize = s.Size()
	fh.FMt = s.ModTime().UnixNano()
	fh.FMode = uint32(s.Mode().Perm())
	return
}

func (fh *FileHandler) RenameReq(absPath string) (err error) {
	// do nothing
	return
}

func (fh *FileHandler) WriteReq(absPath string) (err error) {
	f, err := os.Open(absPath)
	if err != nil {
		log.Errorf("open error, %s", err.Error())
		return
	}
	defer f.Close()
	h := md5.New()
	_, err = io.Copy(fh, f)
	if err != nil {
		return
	}
	_, err = io.Copy(h, f)
	fh.FMd5 = fmt.Sprintf("%x", h.Sum(nil))
	fh.FType = true

	s, err := os.Stat(absPath)
	if err != nil {
		return
	}
	fh.FType = !s.IsDir()
	fh.FSize = s.Size()
	fh.FMt = s.ModTime().UnixNano()
	fh.FMode = uint32(s.Mode().Perm())
	return
}

func (fh *FileHandler) RemoveReq(absPath string) (err error) {
	// do nothing
	return
}

func (fh *FileHandler) ChmodReq(absPath string) (err error) {
	s, err := os.Stat(absPath)
	if err != nil {
		return
	}
	fh.FType = !s.IsDir()
	fh.FSize = s.Size()
	fh.FMt = s.ModTime().UnixNano()
	fh.FMode = uint32(s.Mode().Perm())
	return
}
