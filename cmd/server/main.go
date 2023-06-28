package main

import (
	"fmt"
	"io"
	"net/rpc"
	"os"

	"github.com/bindernews/taki/pkg/rpcfs"
	"github.com/bindernews/taki/pkg/tkserver"
)

func main() {
	fmt.Println(tkserver.SERVER_START_LINE)
	conn := &StdioRw{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}
	rpc.RegisterName(rpcfs.RPC_FILE_CLASS, rpcfs.NewRpcFsServer("/"))
	rpc.RegisterName(tkserver.TAKI_SERVER_CLASS, &tkserver.TakiServer{})

	go rpc.ServeConn(conn)
}

type StdioRw struct {
	io.Reader
	io.Writer
}

func (rw *StdioRw) Close() error {
	return nil
}
