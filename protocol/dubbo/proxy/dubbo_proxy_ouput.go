package dubboproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	mesherCommon "github.com/go-chassis/mesher/common"
	"github.com/go-chassis/mesher/config"
	"github.com/go-chassis/mesher/protocol/dubbo/client"
	"github.com/go-chassis/mesher/protocol/dubbo/dubbo"
	"github.com/go-chassis/mesher/protocol/dubbo/schema"
	"github.com/go-chassis/mesher/protocol/dubbo/utils"
	"github.com/go-chassis/mesher/resolver"

	"github.com/ServiceComb/go-chassis/client/rest"
	"github.com/ServiceComb/go-chassis/core/common"
	chassisconfig "github.com/ServiceComb/go-chassis/core/config"
	"github.com/ServiceComb/go-chassis/core/handler"
	"github.com/ServiceComb/go-chassis/core/invocation"
	"github.com/ServiceComb/go-chassis/core/lager"
	"github.com/ServiceComb/go-chassis/core/loadbalancer"
	"github.com/ServiceComb/go-chassis/core/util/string"
	"github.com/ServiceComb/go-chassis/third_party/forked/afex/hystrix-go/hystrix"
	"github.com/go-chassis/mesher/protocol"
)

var p *sync.Pool
var dr = resolver.GetDestinationResolver()
var sr = resolver.GetSourceResolver()

var ()

const (
	ProxyTag = "mesherproxy"
)

var DubboListenAddr string
var (
	lock sync.RWMutex
)

type ProxyError struct {
	Message string
}

func (e ProxyError) Error() string {
	return e.Message
}

func ConvertDubboReqToHttpReq(ctx *dubbo.InvokeContext, dubboReq *dubbo.Request) *rest.Request {
	restReq := &rest.Request{Req: &http.Request{
		URL:    &url.URL{},
		Header: make(http.Header),
	}}
	args := dubboReq.GetArguments()
	operateID := dubboReq.GetMethodName()
	iName := dubboReq.GetAttachment(dubbo.PathKey, "")

	methd := schema.GetMethodByInterface(iName, operateID)
	if methd == nil {
		lager.Logger.Error("GetMethodByInterface failed:", &util.BaseError{"Cannot find the method"})
		return nil
	}
	ctx.Method = methd
	restReq.SetMethod(methd.Verb)

	var (
		i         = 0
		qureyNum  = 0
		paramsStr = "?"
		body      = []byte{}
	)

	for i = 0; i < len(args); i++ {
		_, in := methd.GetParamNameAndWhere(i)
		paraSchema := methd.GetParamSchema(i)
		v := args[i]
		if in == schema.InBody {
			b, _ := json.Marshal(v.GetValue())
			body = append(body, b...)
		} else {
			var fmtStr string
			var value string
			if paraSchema.Dtype == util.SchemaArray {
				value = util.ArrayToQueryString(paraSchema.Name, v.GetValue())
				fmtStr += value
			} else {
				value, _ = util.ObjectToString(paraSchema.Dtype, v.GetValue()) // (v.GetValue()).(string)
				if qureyNum == 0 {
					fmtStr = fmt.Sprintf("%s=%s", paraSchema.Name, url.QueryEscape(value))
					qureyNum++
				} else {
					fmtStr = fmt.Sprintf("&%s=%s", paraSchema.Name, url.QueryEscape(value))
				}
			}
			paramsStr += fmtStr
		}
	}
	restReq.SetBody(body)

	uri := methd.Path
	if paramsStr != "?" {
		uri += paramsStr
	}
	restReq.SetURI(uri)
	tmpName := schema.GetSvcNameByInterface(iName)
	if tmpName == "" {
		lager.Logger.Error("GetSvcNameByInterface failed:", &util.BaseError{"Cannot find the svc"})
		return nil
	}
	restReq.Req.URL.Host = tmpName // must after setURI
	return restReq
}

func ConvertRestRspToDubboRsp(ctx *dubbo.InvokeContext, resp *rest.Response, dubboRsp *dubbo.DubboRsp) {
	var v interface{}
	var err error
	status := resp.GetStatusCode()
	body := resp.ReadBody()
	if status >= http.StatusBadRequest {
		dubboRsp.SetStatus(dubbo.ServerError)
		if dubboRsp.GetErrorMsg() == "" && body != nil {
			dubboRsp.SetErrorMsg(string(body))
		}
		return
	}
	dubboRsp.SetStatus(dubbo.Ok)
	if body != nil {
		rspSchema := (*(ctx.Method)).GetRspSchema(status)
		if rspSchema != nil {
			v, err = util.RestByteToValue(rspSchema.DType, body)
			if err != nil {
				dubboRsp.SetStatus(dubbo.BadResponse)
				dubboRsp.SetErrorMsg(err.Error())
			} else {
				dubboRsp.SetValue(v)
			}
		} else {
			dubboRsp.SetErrorMsg(string(body))
			dubboRsp.SetStatus(dubbo.ServerError)
		}
	}

}

func Handle(ctx *dubbo.InvokeContext) error {
	interfaceName := ctx.Req.GetAttachment(dubbo.PathKey, "")
	svc := schema.GetSvcByInterface(interfaceName)
	if svc == nil {
		return &util.BaseError{ErrMsg: "can't find the svc by " + interfaceName}
	}

	inv := new(invocation.Invocation)
	inv.SourceServiceID = chassisconfig.SelfServiceID
	inv.SourceMicroService = ctx.Req.GetAttachment(common.HeaderSourceName, "")
	inv.Args = ctx.Req

	inv.MicroServiceName = svc.ServiceName
	inv.Version = svc.Version
	inv.AppID = svc.AppID
	value := ctx.Req.GetAttachment(ProxyTag, "")
	if value == "" { //come from proxyedDubboSvc
		inv.Protocol = schema.GetSupportProto(svc)
	} else {
		inv.Protocol = "dubbo"
	}
	inv.URLPathFormat = ""
	inv.Reply = &dubboclient.WrapResponse{nil} //&rest.Response{Resp: &ctx.Response}
	var err error
	var c *handler.Chain

	if inv.Protocol == "dubbo" {
		//发送请求
		value := ctx.Req.GetAttachment(ProxyTag, "")
		if value == "" { //come from proxyedDubboSvc
			ctx.Req.SetAttachment(common.HeaderSourceName, chassisconfig.SelfServiceName)
			ctx.Req.SetAttachment(ProxyTag, "true")

			if config.Mode == mesherCommon.ModeSidecar {
				c, err = handler.GetChain(common.Consumer, mesherCommon.ChainConsumerOutgoing)
				if err != nil {
					lager.Logger.Error("Get Consumer chain failed.", err)
					return err
				}
			}

			c.Next(inv, func(ir *invocation.InvocationResponse) error {
				return handleDubboRequest(inv, ctx, ir)
			})
		} else { //come from other mesher
			ctx.Req.SetAttachment(ProxyTag, "")
			c, err = handler.GetChain(common.Provider, mesherCommon.ChainProviderIncoming)
			if err != nil {
				lager.Logger.Error("Get Provider Chain failed.", err)
				return err
			}
			c.Next(inv, func(ir *invocation.InvocationResponse) error {
				return handleDubboRequest(inv, ctx, ir)
			})
		}
	} else {
		return ProxyRestHandler(ctx)
	}
	return nil
}

func handleDubboRequest(inv *invocation.Invocation, ctx *dubbo.InvokeContext, ir *invocation.InvocationResponse) error {
	if ir != nil {
		if ir.Err != nil {
			switch ir.Err.(type) {
			case hystrix.FallbackNullError:
				ctx.Rsp.SetStatus(dubbo.Ok)
			case hystrix.CircuitError:
				ctx.Rsp.SetStatus(dubbo.ServiceError)
			case loadbalancer.LBError:
				ctx.Rsp.SetStatus(dubbo.ServiceNotFound)
			default:
				ctx.Rsp.SetStatus(dubbo.ServerError)
			}
			ctx.Rsp.SetErrorMsg(ir.Err.Error())
			return ir.Err
		}
		if inv.Endpoint == "" {
			ctx.Rsp.SetStatus(dubbo.ServerError)
			ctx.Rsp.SetErrorMsg(protocol.ErrUnknown.Error())
			return protocol.ErrUnknown
		}
	} else {
		ctx.Rsp.SetStatus(dubbo.ServerError)
		ctx.Rsp.SetErrorMsg(protocol.ErrUnExpectedHandlerChainResponse.Error())
		return protocol.ErrUnExpectedHandlerChainResponse
	}
	if ir.Result != nil {
		ctx.Rsp = ir.Result.(*dubboclient.WrapResponse).Resp
	} else {
		err := protocol.ErrNilResult
		lager.Logger.Error("CAll Chain  failed", err)
		return err
	}

	return nil
}

func preHandleToRest(ctx *dubbo.InvokeContext) (*rest.Request, *invocation.Invocation, string) {
	restReq := ConvertDubboReqToHttpReq(ctx, ctx.Req)
	if restReq == nil {
		return nil, nil, ""
	}
	inv := new(invocation.Invocation)
	inv.SourceServiceID = chassisconfig.SelfServiceID
	inv.Args = restReq
	inv.Protocol = "rest"
	inv.Reply = rest.NewResponse()
	inv.URLPathFormat = restReq.GetURI()
	inv.SchemaID = ""
	inv.OperationID = ""
	inv.Ctx = context.Background()
	source := stringutil.SplitFirstSep(ctx.RemoteAddr, ":")
	return restReq, inv, source
}

func ProxyRestHandler(ctx *dubbo.InvokeContext) error {
	var err error
	var c *handler.Chain

	req, inv, source := preHandleToRest(ctx)
	if req == nil {
		return &util.BaseError{ErrMsg: "request is invalid "}
	}

	source = "127.0.0.1" //"10.57.75.87"
	//Resolve Source
	si := sr.Resolve(source)
	h := make(map[string]string)
	for k := range req.Req.Header {
		h[k] = req.GetHeader(k)
	}
	//Resolve Destination
	if err = dr.Resolve(source, h, inv.URLPathFormat, &inv.MicroServiceName); err != nil {
		return err
	}

	if config.Mode == mesherCommon.ModeSidecar {
		c, err = handler.GetChain(common.Consumer, mesherCommon.ChainConsumerOutgoing)
		if err != nil {
			lager.Logger.Error("Get chain failed.", err)
			return err
		}
		if si == nil {
			lager.Logger.Info("Can not resolve " + source + " to Source info")
		}
	}

	c.Next(inv, func(ir *invocation.InvocationResponse) error {
		//Send the request to the destination
		return handleRequest(ctx, req, inv.Reply.(*rest.Response), ctx.Rsp, inv, ir)
	})
	ConvertRestRspToDubboRsp(ctx, inv.Reply.(*rest.Response), ctx.Rsp)
	return nil
}

func handleRequest(ctx *dubbo.InvokeContext, req *rest.Request, resp *rest.Response,
	dubboRsp *dubbo.DubboRsp, inv *invocation.Invocation, ir *invocation.InvocationResponse) error {
	if ir != nil {
		if ir.Err != nil {
			switch ir.Err.(type) {
			case hystrix.FallbackNullError:
				resp.SetStatusCode(http.StatusOK)
				dubboRsp.SetErrorMsg(ir.Err.Error())
			case hystrix.CircuitError:
				ir.Status = http.StatusServiceUnavailable
				resp.SetStatusCode(http.StatusServiceUnavailable)
				dubboRsp.SetErrorMsg(ir.Err.Error())
			case loadbalancer.LBError:
				ir.Status = http.StatusBadGateway
				resp.SetStatusCode(http.StatusBadGateway)
				dubboRsp.SetErrorMsg(ir.Err.Error())
			default:
				ir.Status = http.StatusInternalServerError
				resp.SetStatusCode(http.StatusInternalServerError)
				dubboRsp.SetErrorMsg(ir.Err.Error())
			}
			return ir.Err
		}
		if inv.Endpoint == "" {
			ir.Status = http.StatusInternalServerError
			resp.SetStatusCode(http.StatusInternalServerError)
			dubboRsp.SetErrorMsg(ir.Err.Error())
			return protocol.ErrUnknown
		}
	} else {
		dubboRsp.SetErrorMsg(protocol.ErrUnExpectedHandlerChainResponse.Error())
		return protocol.ErrUnExpectedHandlerChainResponse
	}

	ir.Status = resp.GetStatusCode()
	return nil
}
