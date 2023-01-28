package rpcfs

import (
	"context"
	"io/fs"
	"net/rpc"
)

type RemoteFile struct {
	client *rpc.Client
	// Remote file descriptor
	fdint uint
	// Cancel context
	ctx context.Context
	// Cancel handle
	cancelFn context.CancelFunc
	// Call channel
	callCh chan *rpc.Call
}

func newRemoteFile(parent context.Context, client *rpc.Client, fd uint) *RemoteFile {
	rf := &RemoteFile{
		client: client,
		fdint:  fd,
		callCh: make(chan *rpc.Call, 1),
	}
	rf.ctx, rf.cancelFn = context.WithCancel(parent)
	return rf
}

func (rf *RemoteFile) Close() error {
	const METHOD = RPC_FILE_CLS + ".Fclose"
	if rf.fdint == 0 {
		return nil
	}
	res := new(bool)
	// Use separate channel for close in case main one is in use
	call := rf.client.Go(METHOD, rf.fdint, res, nil)
	// Clear fdint so we know we're closed
	rf.fdint = 0
	if err := rf.callWait(call); err != nil {
		return err
	}
	return nil
}

func (rf *RemoteFile) Stat() (fs.FileInfo, error) {
	const METHOD = RPC_FILE_CLS + ".Fstat"
	var res fs.FileInfo
	err := rf.callWait(rf.client.Go(METHOD, rf.fdint, &res, rf.callCh))
	if err != nil {
		return nil, err
	} else {
		return res, nil
	}
}

func (rf *RemoteFile) Read(b []byte) (n int, err error) {
	const METHOD = RPC_FILE_CLS + ".Fread"
	req := FreadReq{Fd: rf.fdint, N: len(b)}
	res := FreadRes{}
	if err = rf.callWait(rf.client.Go(METHOD, &req, &res, rf.callCh)); err != nil {
		return
	}
	n = copy(b[:], res.B[:])
	return
}

func (rf *RemoteFile) Write(b []byte) (n int, err error) {
	const METHOD = RPC_FILE_CLS + ".Fwrite"
	req := FwriteReq{Fd: rf.fdint, B: b}
	res := FwriteRes{}
	if err = rf.callWait(rf.client.Go(METHOD, &req, &res, rf.callCh)); err != nil {
		return
	}
	n = res.N
	return
}

func (rf *RemoteFile) Seek(offset int64, whence int) (pos int64, err error) {
	const METHOD = RPC_FILE_CLS + ".Fseek"
	req := FseekReq{Fd: rf.fdint, Offset: offset, Whence: whence}
	res := FseekRes{}
	if err = rf.callWait(rf.client.Go(METHOD, &req, &res, rf.callCh)); err != nil {
		return
	}
	pos = res.Pos
	return
}

// Helper function that waits for either the call to finish or the context to be cancelled.
func (rf *RemoteFile) callWait(call *rpc.Call) (err error) {
	select {
	case <-call.Done:
		err = call.Error
	case <-rf.ctx.Done():
		err = rf.ctx.Err()
	}
	return
}
