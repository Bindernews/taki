package imager

import (
	"context"
	"io"
	"net/rpc"

	"github.com/bindernews/taki/pkg/fsdiff"
	"github.com/bindernews/taki/pkg/tkserver"
)

type ClientApi struct {
	*rpc.Client
	ctx context.Context
}

func NewClientApi(ctx context.Context, conn io.ReadWriteCloser) *ClientApi {
	return &ClientApi{
		Client: rpc.NewClient(conn),
		ctx:    ctx,
	}
}

// Attemts to determine a path to access the root of the target container from
// the debug container.
func (c *ClientApi) GetTargetRoots() ([]string, error) {
	req := tkserver.Empty{}
	res := tkserver.GetRootsRes{}
	if err := c.RpcCall("GetRoots", req, &res); err != nil {
		return nil, err
	}
	return res.Roots, nil
}

func (c *ClientApi) GenerateDiff(meta *fsdiff.DirMeta) (err error) {
	req := tkserver.GenerateDiffReq{Base: meta}
	res := tkserver.Empty{}
	if err = c.RpcCall("GenerateDiff", &req, &res); err != nil {
		return
	}
	return
}

func (c *ClientApi) SetConfig(config *tkserver.ServerConfig) (err error) {
	res := tkserver.Empty{}
	return c.RpcCall("SetConfig", config, &res)
}

func (c *ClientApi) TarStart() error {
	return c.RpcCall("TarStart", tkserver.Empty{}, &tkserver.Empty{})
}

func (c *ClientApi) TarProgress() (float64, error) {
	var progress float64 = 0
	err := c.RpcCall("TarProgress", tkserver.Empty{}, &progress)
	return progress, err
}

func (c *ClientApi) RpcCall(method string, args, reply any) error {
	methodReal := tkserver.TAKI_SERVER_CLASS + "." + method
	call := c.Go(methodReal, args, reply, nil)
	select {
	case <-call.Done:
		if call.Error != nil {
			return call.Error
		}
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
	return nil
}
