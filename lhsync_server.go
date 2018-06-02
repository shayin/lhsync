package main

import (
	"flag"
	"net"
	"github.com/ngaut/log"
	"os"
	"bufio"
	"hash/crc32"
	"encoding/json"
	"io"
	"fmt"
	"github.com/shayin/lhsync/syncfile"
	"github.com/fsnotify/fsnotify"
)

var (
	listen = flag.String("listen", "127.0.0.1:5623", "server listen address")
	dir    = flag.String("dir", "/tmp/syncdir", "receive dir")
)

func main() {
	flag.Parse()
	l, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatalf("listen error, %s", err.Error())
	}
	err = os.MkdirAll(*dir, 0755)
	if err != nil {
		log.Fatalf("make dir error, %s", err.Error())
	}
	log.Infof("listen %s", *listen)
	serve(l)
}

//处理函数，这是一个状态机
//根据数据包来做解析
//数据包的格式为|0xFF|0xFF|len(高)|len(低)|Data|CRC高16位|0xFF|0xFE
//其中len为data的长度，实际长度为len(高)*256+len(低)
//CRC为32位CRC，取了最高16位共2Bytes
//0xFF|0xFF和0xFF|0xFE类似于前导码
func handle(conn net.Conn) {
	// close connection before exit
	defer conn.Close()
	// 状态机状态
	state := 0x00
	// 数据包长度
	length := uint16(0)
	// crc校验和
	crc16 := uint16(0)
	var recvBuffer []byte
	// 游标
	cursor := uint16(0)
	bufferReader := bufio.NewReader(conn)
	//状态机处理数据
	for {
		recvByte, err := bufferReader.ReadByte()
		//log.Debugf("%+v", recvByte == 0xFF)
		if err != nil {
			if err == io.EOF {
				log.Infof("client %s is close\n", conn.RemoteAddr().String())
			}
			return
		}
		//进入状态机，根据不同的状态来处理
		switch state {
		case 0x00:
			if recvByte == 0xFF {
				state = 0x01
				//初始化状态机
				recvBuffer = nil
				length = 0
				crc16 = 0
			} else {
				state = 0x00
			}
			break
		case 0x01:
			if recvByte == 0xFF {
				state = 0x02
			} else {
				state = 0x00
			}
			break
		case 0x02:
			length += uint16(recvByte) * 256
			state = 0x03
			break
		case 0x03:
			length += uint16(recvByte)
			// 一次申请缓存，初始化游标，准备读数据
			recvBuffer = make([]byte, length)
			cursor = 0
			state = 0x04
			break
		case 0x04:
			//不断地在这个状态下读数据，直到满足长度为止
			recvBuffer[cursor] = recvByte
			cursor++
			if cursor == length {
				state = 0x05
			}
			break
		case 0x05:
			crc16 += uint16(recvByte) * 256
			state = 0x06
			break
		case 0x06:
			crc16 += uint16(recvByte)
			state = 0x07
			break
		case 0x07:
			if recvByte == 0xFF {
				state = 0x08
			} else {
				state = 0x00
			}
		case 0x08:
			if recvByte == 0xFE {
				//执行数据包校验
				if (crc32.ChecksumIEEE(recvBuffer)>>16)&0xFFFF == uint32(crc16) {
					var packet = new(syncfile.SyncFile)
					//把拿到的数据反序列化出来
					json.Unmarshal(recvBuffer, packet)
					//新开协程处理数据
					go processData(packet, conn)
				} else {
					log.Error("verify crc failure !")
					conn.Write([]byte("verify crc failure \n"))
				}
			}
			//状态机归位,接收下一个包
			state = 0x00
		default:
			log.Errorf("unknown packet state: %s", recvByte)
		}
	}
}

func processData(packet *syncfile.SyncFile, conn net.Conn) {
	log.Debug(packet)
	var err error
	path := fmt.Sprintf("%s/%s", *dir, packet.FileName)
	switch packet.FileOp {
	case fsnotify.Create, fsnotify.Write:
		if !packet.FileType {
			err = writeFile(packet.Content, path, packet.FileMode)
		} else {
			err = os.MkdirAll(path, packet.FileMode)
		}
	case fsnotify.Chmod:
		err = chmodFile(path, packet.FileMode)
	case fsnotify.Remove:
		err = removeFile(path)
	case fsnotify.Rename:

	default:
		err = fmt.Errorf("unknow file operation: %+v", packet.FileOp)
	}
	if err != nil {
		conn.Write([]byte(fmt.Sprintf("sync failure,  %s\n", err.Error())))
		return
	}
	conn.Write([]byte("sync success \n"))
}

func renameFile(oldPath string, newPath string) error {
	err := os.Rename(oldPath, newPath)
	if err != nil {
		log.Errorf("rename %s error: ", err.Error())
	}
	return err
}

func removeFile(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		log.Errorf("remove %s error: ", err.Error())
	}
	return err
}

func chmodFile(path string, fMode os.FileMode) error {
	err := os.Chmod(path, fMode)
	if err != nil {
		log.Errorf("chmod %s error: ", err.Error())
	}
	return err
}

func writeFile(data []byte, path string, fMode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_TRUNC|os.O_CREATE, fMode)
	if err != nil {
		log.Errorf("write file error, %s", err.Error())
		return err
	}
	defer f.Close()
	n, err := f.Write(data)
	log.Debugf("%d, %s", n, string(data))
	if err != nil {
		log.Errorf("write file error, ", err.Error())
		return err
	}
	if n != len(data) {
		log.Warnf("sync not complete, expect %d, but %d written", len(data), n)
	}
	log.Infof("sync %s success", path)
	return nil
}

func serve(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Errorf("accept error, %s", err.Error())
			continue
		}
		log.Infof("accept tcp client %s", conn.RemoteAddr().String())
		go handle(conn)
	}
}
