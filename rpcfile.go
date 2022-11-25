package taki

import (
	"context"
	"errors"
	"io/fs"
	"net/rpc"
	"os"
)

const RPC_FILE_CLS = "RpcFile"

type RpcFile struct {
	// Next virtual FD
	nextFd uint
	// Map of virtual FDs to files
	files map[uint]*os.File
}

func NewRpcFile() *RpcFile {
	return &RpcFile{
		nextFd: 1,
		files:  make(map[uint]*os.File),
	}
}

type FopenReq struct {
	Path string
	Flag int
	Perm os.FileMode
}

type FreadReq struct {
	Fd uint
	N  int
}
type FreadRes struct {
	B []byte
}

type FwriteReq struct {
	Fd uint
	B  []byte
}
type FwriteRes struct {
	N int
}

type FseekReq struct {
	Fd     uint
	Offset int64
	Whence int
}
type FseekRes struct {
	Pos int64
}

func (s *RpcFile) Fopen(req *FopenReq, res *uint) error {
	fd, err := os.OpenFile(req.Path, req.Flag, req.Perm)
	if err != nil {
		return err
	}
	fdint := s.nextFd
	s.nextFd += 1
	*res = fdint
	s.files[fdint] = fd
	return nil
}

func (s *RpcFile) Fclose(fdint uint, res *bool) error {
	if fd, err := s.getFile(fdint); err != nil {
		return err
	} else {
		err := fd.Close()
		delete(s.files, fdint)
		return err
	}
}

func (s *RpcFile) Fread(req *FreadReq, res *FreadRes) (err error) {
	var fd *os.File
	var n int
	if fd, err = s.getFile(req.Fd); err != nil {
		return
	}
	buf := make([]byte, req.N)
	if n, err = fd.Read(buf); err != nil {
		return
	}
	res.B = buf[:n]
	return
}

func (s *RpcFile) Fwrite(req *FwriteReq, res *FwriteRes) (err error) {
	var fd *os.File
	if fd, err = s.getFile(req.Fd); err != nil {
		return
	}
	if res.N, err = fd.Write(req.B); err != nil {
		return
	}
	return
}

func (s *RpcFile) Fseek(req *FseekReq, res *FseekRes) (err error) {
	var fd *os.File
	if fd, err = s.getFile(req.Fd); err != nil {
		return
	}
	if res.Pos, err = fd.Seek(req.Offset, req.Whence); err != nil {
		return
	}
	return
}

func (s *RpcFile) getFile(fdint uint) (*os.File, error) {
	fd := s.files[fdint]
	if fd == nil {
		return nil, os.ErrInvalid
	}
	return fd, nil
}

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

func NewRemoteFile(parent context.Context, client *rpc.Client) *RemoteFile {
	rf := &RemoteFile{
		client: client,
		fdint:  0,
		callCh: make(chan *rpc.Call, 1),
	}
	rf.ctx, rf.cancelFn = context.WithCancel(parent)
	return rf
}

func (rf *RemoteFile) OpenFile(path string, flag int, perm fs.FileMode) error {
	const METHOD = RPC_FILE_CLS + ".Fopen"
	if rf.fdint != 0 {
		return errors.New("file is already opened")
	}
	req := FopenReq{
		Path: path,
		Flag: flag,
		Perm: perm,
	}
	res := new(uint)
	call := rf.client.Go(METHOD, &req, res, rf.callCh)
	if err := rf.callWait(call); err != nil {
		return err
	}
	rf.fdint = *res
	return nil
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
func (rf *RemoteFile) callWait(call *rpc.Call) error {
	select {
	case <-call.Done:
		if call.Error != nil {
			return call.Error
		}
	case <-rf.ctx.Done():
		return rf.ctx.Err()
	}
	return nil
}

func (rf *RemoteFile) Create(path string) error {
	const flag = os.O_CREATE | os.O_RDWR | os.O_TRUNC
	return rf.OpenFile(path, flag, 0)
}

func (rf *RemoteFile) Open(path string) error {
	const flag = os.O_RDONLY
	return rf.OpenFile(path, flag, 0)
}
