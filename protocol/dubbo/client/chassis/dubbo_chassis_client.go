package chassisclient

import (
	"context"
	"os"
	"sync"

	mesherCommon "github.com/go-chassis/mesher/common"
	dubboClient "github.com/go-chassis/mesher/protocol/dubbo/client"
	"github.com/go-chassis/mesher/protocol/dubbo/dubbo"
	"github.com/go-chassis/mesher/protocol/dubbo/proxy"
	"github.com/go-chassis/mesher/protocol/dubbo/utils"

	"github.com/ServiceComb/go-chassis/core/client"
	"github.com/ServiceComb/go-chassis/core/lager"
)

const Name = "dubbo"

func init() {
	client.InstallPlugin(Name, NewDubboChassisClient)
}

type dubboChassisClient struct {
	once     sync.Once
	opts     client.Options
	reqMutex sync.Mutex
}

func (c *dubboChassisClient) NewRequest(service, schemaID, operationID string, arg interface{}) *client.Request {
	return &client.Request{
		MicroServiceName: service,
		Schema:           schemaID,
		Operation:        operationID,
		Arg:              arg,
	}
}

//NewDubboChassisClient create new client
func NewDubboChassisClient(options client.Options) client.ProtocolClient {

	rc := &dubboChassisClient{
		once: sync.Once{},
		opts: options,
	}
	return client.ProtocolClient(rc)
}

func (c *dubboChassisClient) String() string {
	return "highway_client"
}

func (c *dubboChassisClient) Call(ctx context.Context, addr string, req *client.Request, rsp interface{}) error {
	dubboReq := req.Arg.(*dubbo.Request)

	endPoint := addr
	if endPoint == dubboproxy.DubboListenAddr {
		endPoint = os.Getenv(mesherCommon.EnvSpecificAddr)
	}
	if endPoint == "" {
		return &util.BaseError{" The endpoint is empty"}
	}
	lager.Logger.Info("Dubbo invoke endPont: " + endPoint)
	dubboCli, err := dubboClient.CachedClients.GetClient(endPoint)
	if err != nil {
		lager.Logger.Error("Invalid Request addr ="+endPoint, err)
		return err
	}

	dubboRsp, errSnd := dubboCli.Send(dubboReq)
	if errSnd != nil {
		lager.Logger.Error("Dubbo server exception:", errSnd)
		return errSnd
	}
	resp := rsp.(*dubboClient.WrapResponse)
	resp.Resp = dubboRsp
	return nil
}
