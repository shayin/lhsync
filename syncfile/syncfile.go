package syncfile

import (
	"google.golang.org/grpc"
	"github.com/shayin/lhsync/pb"
	"github.com/ngaut/log"
	"context"
)

type SyncCom struct {
	conn *grpc.ClientConn
}

func NewSyncCom(address string) (*SyncCom, error) {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	syncCom := &SyncCom{
		conn: conn,
	}
	return syncCom, nil
}


func (sc *SyncCom) SyncToServer(fileData *lhsync_pb.FileData) (*lhsync_pb.SyncResp, error) {
	client := lhsync_pb.NewLhSyncClient(sc.conn)
	resp, err := client.SyncFile(context.Background(), fileData)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (sc *SyncCom) Close() {
	log.Infof("close connect")
	sc.conn.Close()
}