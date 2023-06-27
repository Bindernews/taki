package main

import (
	"fmt"
	"io"
	"net/rpc"
	"os"

	"github.com/bindernews/taki/pkg/rpcfs"
	"github.com/bindernews/taki/v1"
)

func main() {
	fmt.Println(taki.SERVER_START_LINE)
	conn := &StdioRw{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}
	rpc.RegisterName(rpcfs.RPC_FILE_CLS, rpcfs.NewRpcFsServer("/"))
	rpc.RegisterName("ServerApi", &taki.ServerImpl{})

	go rpc.ServeConn(conn)
}

type StdioRw struct {
	io.Reader
	io.Writer
}

func (rw *StdioRw) Close() error {
	return nil
}
