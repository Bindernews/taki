package rpcfs

import (
	"io/fs"
	"os"
)

// rpc class name for requests
const RPC_FILE_CLASS = "RpcFsServer"

type RpcFsServer struct {
	// Next virtual FD
	nextFd uint
	// Map of virtual FDs to files
	files map[uint]*os.File
	// Root local FS
	rootFs fs.FS
}

func NewRpcFsServer(root string) *RpcFsServer {
	return &RpcFsServer{
		nextFd: 1,
		files:  make(map[uint]*os.File),
		rootFs: os.DirFS(root),
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

type ReaddirReq struct {
	Path string
}
type ReaddirRes struct {
	Entries []fs.DirEntry
}

func (s *RpcFsServer) Fopen(req *FopenReq, res *uint) error {
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

func (s *RpcFsServer) Fclose(fdint uint, res *bool) error {
	if fd, err := s.getFile(fdint); err != nil {
		return err
	} else {
		err := fd.Close()
		delete(s.files, fdint)
		return err
	}
}

func (s *RpcFsServer) Fread(req *FreadReq, res *FreadRes) (err error) {
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

func (s *RpcFsServer) Fwrite(req *FwriteReq, res *FwriteRes) (err error) {
	var fd *os.File
	if fd, err = s.getFile(req.Fd); err != nil {
		return
	}
	if res.N, err = fd.Write(req.B); err != nil {
		return
	}
	return
}

func (s *RpcFsServer) Fseek(req *FseekReq, res *FseekRes) (err error) {
	var fd *os.File
	if fd, err = s.getFile(req.Fd); err != nil {
		return
	}
	if res.Pos, err = fd.Seek(req.Offset, req.Whence); err != nil {
		return
	}
	return
}

func (s *RpcFsServer) Fstat(fdint uint, info *fs.FileInfo) error {
	fd, err := s.getFile(fdint)
	if err != nil {
		return err
	}
	if *info, err = fd.Stat(); err != nil {
		return err
	}
	return nil
}

func (s *RpcFsServer) ReadDir(req *ReaddirReq, res *ReaddirRes) error {
	entries, err := os.ReadDir(req.Path)
	if err != nil {
		return err
	}
	*res = ReaddirRes{
		Entries: entries,
	}
	return nil
}

func (s *RpcFsServer) Glob(pattern string, items *[]string) error {
	if matches, err := fs.Glob(s.rootFs, pattern); err != nil {
		return err
	} else {
		*items = matches
		return nil
	}
}

func (s *RpcFsServer) getFile(fdint uint) (*os.File, error) {
	fd := s.files[fdint]
	if fd == nil {
		return nil, os.ErrInvalid
	}
	return fd, nil
}
