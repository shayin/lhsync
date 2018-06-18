package main

import (
	"flag"
	"net"
	"github.com/ngaut/log"
	"github.com/shayin/lhsync/config"
	"google.golang.org/grpc"
	"github.com/shayin/lhsync/pb"
	"github.com/shayin/lhsync/syncserver"
	"google.golang.org/grpc/reflection"
)

var (
	sConfFile = flag.String("c", "server.yaml", "conf file")
	dir    = flag.String("dir", "/tmp/syncdir", "receive dir")
)

func main() {
	flag.Parse()
	if err := config.InitServerConfig(*sConfFile); err != nil {
		log.Fatal(err)
	}
	l, err := net.Listen("tcp", config.SConf.Listen)
	if err != nil {
		log.Fatalf("listen %s error: %s", config.SConf.Listen, err.Error())
	}
	log.Infof("listen at %s", config.SConf.Listen)
	rpcServer := grpc.NewServer()
	lhsync_pb.RegisterLhSyncServer(rpcServer, &syncserver.FileServer{})
	reflection.Register(rpcServer)
	if err = rpcServer.Serve(l); err != nil {
		log.Fatalf("serve at %s error: ", config.SConf.Listen, err.Error())
	}
}