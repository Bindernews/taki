package rpcfs

import (
	"context"
	"io/fs"
	"net/rpc"
	"os"
)

// Client side of RpcFs.
type RpcFs struct {
	// Client connected to server
	client *rpc.Client
	// Context with cancel
	ctx context.Context
	// Cancel function
	cancel context.CancelFunc
}

func NewRpcFs(parent context.Context, client *rpc.Client) *RpcFs {
	ctx, cancel := context.WithCancel(parent)
	return &RpcFs{
		client: client,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (f *RpcFs) OpenFile(path string, flag int, perm fs.FileMode) (*RemoteFile, error) {
	req := FopenReq{
		Path: path,
		Flag: flag,
		Perm: perm,
	}
	res := new(uint)
	if err := f.rpcCall(".Fopen", req, res); err != nil {
		return nil, err
	}
	rf := newRemoteFile(f.ctx, f.client, *res)
	return rf, nil
}

// Implement FS.Open
func (f *RpcFs) Open(path string) (fs.File, error) {
	return f.OpenRead(path)
}

func (f *RpcFs) OpenRead(path string) (*RemoteFile, error) {
	const flag = os.O_RDONLY
	return f.OpenFile(path, flag, 0)
}

func (f *RpcFs) Create(path string) (*RemoteFile, error) {
	const flag = os.O_CREATE | os.O_RDWR | os.O_TRUNC
	return f.OpenFile(path, flag, 0)
}

// Implement GlobFS.Glob
func (f *RpcFs) Glob(pattern string) ([]string, error) {
	matches := make([]string, 0)
	err := f.rpcCall(".Glob", pattern, &matches)
	return matches, err
}

// Implement ReadDirFS.ReadDir
func (f *RpcFs) ReadDir(path string) ([]fs.DirEntry, error) {
	req := ReaddirReq{Path: path}
	res := make([]fs.DirEntry, 0)
	err := f.rpcCall(".ReadDir", &req, &res)
	return res, err
}

func (f *RpcFs) rpcCall(method string, args, reply any) error {
	methodReal := RPC_FILE_CLS + method
	call := f.client.Go(methodReal, args, reply, nil)
	select {
	case <-call.Done:
		return call.Error
	case <-f.ctx.Done():
		return f.ctx.Err()
	}
}
