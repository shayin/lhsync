package main

import (
	"flag"
	"fmt"
	"crypto/md5"
	"os"
	"io"
	"net"
	"github.com/ngaut/log"
	"github.com/shayin/lhsync/syncfile"
	"encoding/json"
	"encoding/binary"
	"unsafe"
	"hash/crc32"
	"github.com/fsnotify/fsnotify"
	"sync"
	"strings"
	"path/filepath"
)

var (
	dest     = flag.String("dest", "127.0.0.1:5623", "sync to server")
	watchdir = flag.String("dir", "", "watch dir")
	file     = flag.String("file", "", "sync file")
)

var is_little_endian bool
var watcher *fsnotify.Watcher

func main() {
	flag.Parse()
	if *file == "" && *watchdir == "" {
		log.Fatalf("-file | -dir")
	}

	is_little_endian = systemEndian()
	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Infof("event: %+v", event)
				syncToServer(event, *dest)
			case err := <-watcher.Errors:
				log.Errorf("error:", err)
			}
		}
	}()

	err = filepath.Walk(*watchdir, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			log.Infof("watch: %s", path)
			err = watcher.Add(path)
			if err != nil {
				log.Fatal(err)
			}
			return nil
		}
		return nil
	})
	if err != nil {
		log.Fatalf("watch dir %s error: %s", *watchdir, err.Error())
	}

	wg.Wait()
}

func syncToServer(event fsnotify.Event, dest string) error {
	fileInfo, err := getSyncFileInfo(event)
	if err != nil {
		return err
	}
	conn, err := net.Dial("tcp", dest)
	if err != nil {
		log.Errorf("dial error, %s", err.Error())
		return err
	}
	defer conn.Close()

	tcpPacket, err := doPacket(fileInfo)
	if err != nil {
		log.Errorf("dopacket error, %s", err.Error())
		return err
	}

	// 开始发送
	log.Infof("begin sync %s to %s", event.Name, dest)
	conn.Write(tcpPacket)
	log.Infof("complete sync %s to %s", event.Name, dest)

	for {
		buffer := make([]byte, 1024)
		num, err := conn.Read(buffer)
		if err == nil && num > 0 {
			log.Info(string(buffer[:num]))
			break
		}
	}
	return nil
}

// 打包
// |0xFF|0xFF|len(高)|len(低)|Data|CRC高16位|0xFF|0xFE
func doPacket(packet *syncfile.SyncFile) (tcpPacket []byte, err error) {
	jsonData, err := json.Marshal(packet)
	if err != nil {
		return
	}
	jsonLen := len(jsonData)
	// 包切片
	tcpPacket = append(tcpPacket, 0xFF, 0xFF)

	// 数据长度字节
	var lenByte = make([]byte, 2)
	binary.BigEndian.PutUint16(lenByte, uint16(jsonLen))
	// 小字节序 256 => [0 1]
	// 大字节序 256 => [1 0]
	if is_little_endian {
		tcpPacket = append(tcpPacket, lenByte[1], lenByte[0])
	} else {
		tcpPacket = append(tcpPacket, lenByte[0], lenByte[1])
	}
	tcpPacket = append(tcpPacket, jsonData...)

	binary.BigEndian.PutUint16(lenByte, uint16((crc32.ChecksumIEEE(jsonData)>>16)&0xFFFF))
	if is_little_endian {
		tcpPacket = append(tcpPacket, lenByte[1], lenByte[0])
	} else {
		tcpPacket = append(tcpPacket, lenByte[0], lenByte[1])
	}
	tcpPacket = append(tcpPacket, 0xFF, 0xFE)
	return
}

func getSyncFileInfo(event fsnotify.Event) (*syncfile.SyncFile, error) {
	sf := new(syncfile.SyncFile)
	path := event.Name
	sf.FileOp = event.Op
	sf.FileName = getRelativeFileName(path)
	if sf.FileName == "" {
		return sf, fmt.Errorf("cannot get relative file name, path: %s, watchpath: %s", path, *watchdir)
	}

	switch event.Op {
	case fsnotify.Create:
		watcher.Add(path)
	case fsnotify.Write:
		f, err := os.Open(path)
		if err != nil {
			log.Errorf("open error, %s", err.Error())
			return sf, err
		}
		defer f.Close()
		h := md5.New()
		_, err = io.Copy(sf, f)
		if err != nil {
			log.Errorf("iocopy error, %s", err.Error())
			return sf, err
		}
		_, err = io.Copy(h, f)
		sf.FileMd5 = fmt.Sprintf("%x", h.Sum(nil))
	case fsnotify.Chmod:
	case fsnotify.Remove:
		return sf, nil
	case fsnotify.Rename:
	default:
		err := fmt.Errorf("unknown event: %+v", event.Op)
		log.Error(err)
		return sf, err
	}
	fi, err := os.Lstat(path)
	if err != nil {
		log.Errorf("get file info error, %s", err.Error())
		return sf, err
	}

	sf.FileSize = fi.Size()
	sf.FileMt = fi.ModTime()
	sf.FileMode = fi.Mode().Perm()
	sf.FileType = fi.IsDir()
	log.Debugf("file: %+v", sf)
	return sf, nil
}

func getRelativeFileName(allPath string) string {
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()
	pathInfo := strings.SplitAfter(allPath, *watchdir)
	if len(pathInfo) <= 1 {
		return ""
	}
	p := pathInfo[1]
	return p
}

// 是否小字节序
func systemEndian() bool {
	const INT_SIZE = int(unsafe.Sizeof(0))
	var i = 0x1
	bs := (*[INT_SIZE]byte)(unsafe.Pointer(&i))
	if bs[0] == 0 {
		log.Debugf("system endian is little endian")
		return true
	} else {
		log.Debugf("system endian is big endian")
		return false
	}
}
