package rpcfs_test

import (
	"io/fs"
	"log"
	"testing"

	"github.com/bindernews/taki/rpcfs"
)

func TestInterfaces(t *testing.T) {
	inst := &rpcfs.RpcFs{}
	var i1 fs.GlobFS = inst
	var i2 fs.ReadDirFS = inst
	log.Println(i1, i2)
}
