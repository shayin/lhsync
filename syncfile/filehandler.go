package syncfile

import (
	"os"
	"github.com/ngaut/log"
)

func (fh *FileHandler) CreateHandler(absPath string) (err error) {
	fMode := os.FileMode(fh.FMode)
	if fh.FType {
		err = fh.WriteHandler(absPath)
	} else {
		err = os.MkdirAll(absPath, fMode)
	}
	return
}

func (fh *FileHandler) WriteHandler(absPath string) (err error) {
	fMode := os.FileMode(fh.FMode)
	f, err := os.OpenFile(absPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, fMode)
	if err != nil {
		return
	}
	defer f.Close()
	data := fh.FContent
	n, err := f.Write(data)
	log.Debugf("%d, %s", n, string(data))
	if err != nil {
		return
	}
	if n != len(data) {
		log.Warnf("sync not complete, expect %d, but %d written", len(data), n)
	}

	return
}

func (fh *FileHandler) ChmodHandler(absPath string) (err error) {
	fMode := os.FileMode(fh.FMode)
	err = os.Chmod(absPath, fMode)
	return
}

func (fh *FileHandler) RemoveHandler(absPath string) (err error) {
	err = os.RemoveAll(absPath)
	return
}

func (fh *FileHandler) RenameHandler(absPath string) (err error) {
	//err = os.Rename(absPath, absPath)
	err = os.RemoveAll(absPath)
	return
}
