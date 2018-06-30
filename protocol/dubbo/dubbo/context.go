package dubbo

import (
	"github.com/go-chassis/mesher/protocol/dubbo/schema"
	//	"github.com/servicecomb/go-chassis/core/invocation"
)

type InvokeContext struct {
	Req        *Request
	Rsp        *DubboRsp
	Method     *schema.DefMethod
	SvcName    string
	RemoteAddr string
}
