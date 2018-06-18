package syncserver

import (
	"context"
	"github.com/shayin/lhsync/pb"
	"github.com/ngaut/log"
	"github.com/fsnotify/fsnotify"
	"fmt"
	"github.com/shayin/lhsync/config"
	"github.com/shayin/lhsync/syncfile"
)

type FileServer struct {}

func (fs *FileServer) SyncFile(ctx context.Context, in *lhsync_pb.FileData) (out *lhsync_pb.SyncResp, err error) {
	log.Infof("receive from client: %+v", in)

	out = new(lhsync_pb.SyncResp)
	if _, ok := config.SConf.SyncPath[in.PathKey]; !ok {
		err = fmt.Errorf("cannot find path config: %s", in.PathKey)
		out.Msg = err.Error()
		return
	}

	fh := syncfile.FileHandler(*in)
	fop := fsnotify.Op(in.FOp)
	absPath := fmt.Sprintf("%s/%s", config.SConf.SyncPath[in.PathKey], in.RelPath)

	switch fop {
	case fsnotify.Create:
		err = fh.CreateHandler(absPath)
	case fsnotify.Write:
		err = fh.WriteHandler(absPath)
	case fsnotify.Chmod:
		err = fh.ChmodHandler(absPath)
	case fsnotify.Remove:
		err = fh.RemoveHandler(absPath)
	case fsnotify.Rename:
		err = fh.RenameHandler(absPath)
	default:
		err = fmt.Errorf("unknown event handler: %s", fop.String())
	}

	if err != nil {
		log.Errorf("%s %s error: %s", fop.String(), absPath, err.Error())
		out.Msg = err.Error()
	} else {
		out.Msg = fmt.Sprintf("%s %s success", fop.String(), absPath)
	}

	return
}