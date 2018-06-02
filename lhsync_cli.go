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
)

var (
	dest     = flag.String("dest", "127.0.0.1:5623", "sync to server")
	watchdir = flag.String("dir", "", "watch dir")
	file = flag.String("file", "", "sync file")
)

var is_little_endian bool

func main() {
	flag.Parse()
	if *file == "" && *watchdir == "" {
		log.Fatalf("-file | -dir")
	}

	is_little_endian = systemEndian()

	watcher, err := fsnotify.NewWatcher()
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
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Infof("modified file:", event.Name)
					syncToServer(event.Name, *dest)
				}
			case err := <-watcher.Errors:
				log.Errorf("error:", err)
			}
		}
	}()
	err = watcher.Add(*watchdir)
	if err != nil {
		log.Fatal(err)
	}
	wg.Wait()
}

func syncToServer(fName string, dest string) error {
	fileInfo, err := getFileInfo(fName)
	if err != nil {
		return err
	}
	conn, err := net.Dial("tcp", dest)
	if err != nil {
		log.Errorf("dial error, %s", err.Error())
		return err
	}
	defer conn.Close()
	fHandler, err := os.Open(fName)
	if err != nil {
		log.Errorf("open file error, %s", err.Error())
		return err
	}
	defer fHandler.Close()
	var packet = new(syncfile.SyncPacket)
	packet.Header = fileInfo
	io.Copy(packet, fHandler)

	tcpPacket, err := doPacket(packet)
	if err != nil {
		log.Errorf("dopacket error, %s", err.Error())
		return err
	}

	// 开始发送
	log.Infof("begin sync %s to %s", fName, dest)
	conn.Write(tcpPacket)
	log.Infof("complete sync %s to %s", fName, dest)

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
func doPacket(packet *syncfile.SyncPacket) (tcpPacket []byte, err error) {
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

func getFileInfo(fName string) (*syncfile.SyncFile, error) {
	sf := new(syncfile.SyncFile)
	fi, err := os.Lstat(fName)
	if err != nil {
		log.Errorf("get file info error, %s", err.Error())
		return sf, err
	}
	f, err := os.Open(fName)
	if err != nil {
		log.Errorf("open error, %s", err.Error())
		return sf, err
	}

	h := md5.New()
	_, err = io.Copy(h, f)
	sf.FileName = fi.Name()
	sf.FileSize = fi.Size()
	sf.FileMt = fi.ModTime()
	sf.FileMode = fi.Mode().Perm()
	sf.FileMd5 = fmt.Sprintf("%x", h.Sum(nil))
	sf.FileType = fi.IsDir()
	return sf, nil
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
