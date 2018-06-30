package dubboproxy

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	mesherCommon "github.com/go-chassis/mesher/common"
	"github.com/go-chassis/mesher/protocol/dubbo/client"
	"github.com/go-chassis/mesher/protocol/dubbo/dubbo"
	"github.com/go-chassis/mesher/protocol/dubbo/schema"
	"github.com/go-chassis/mesher/protocol/dubbo/utils"

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

func ConvertDubboRspToRestRsp(dubboRsp *dubbo.DubboRsp, w http.ResponseWriter, ctx *dubbo.InvokeContext) error {
	status := dubboRsp.GetStatus()
	if status == dubbo.Ok {
		w.WriteHeader(http.StatusOK)
		rspSchema := (*(ctx.Method)).GetRspSchema(http.StatusOK)
		if rspSchema != nil {
			v, err := util.ObjectToString(rspSchema.DType, dubboRsp.GetValue())
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.Write([]byte(v))
			}
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	return nil
}

func ConvertHttpReqToDubboReq(restReq *http.Request, ctx *dubbo.InvokeContext, inv *invocation.Invocation) error {
	req := ctx.Req
	uri := restReq.URL
	i := 0
	var dubboArgs []util.Argument
	queryAgrs := uri.Query()
	arg := &util.Argument{}

	svcSchema, methd := schema.GetSchemaMethodBySvcUrl(inv.MicroServiceName, "", inv.Version, inv.AppID,
		strings.ToLower(restReq.Method), string(restReq.URL.String()))
	if methd == nil {
		return &util.BaseError{"Method not been found"}
	}
	req.SetMethodName(methd.OperaID)
	req.SetAttachment(dubbo.DubboVersionKey, dubbo.DubboVersion)
	req.SetAttachment(dubbo.PathKey, svcSchema.Info["x-java-interface"]) //interfaceSchema.JavaClsName
	req.SetAttachment(dubbo.VersionKey, "0.0.0")
	ctx.Method = methd
	var err error

	//处理参数
	dubboArgs = make([]util.Argument, len(methd.Paras))
	for _, v := range methd.Paras {
		var byteTmp []byte
		var bytesTmp [][]byte
		itemType := "string" //默认为string
		if strings.EqualFold(v.Where, "query") {
			byteTmp = []byte(queryAgrs.Get(v.Name))
		} else if restReq.Body != nil {
			byteTmp, _ = ioutil.ReadAll(restReq.Body)
		}
		if byteTmp == nil && v.Required {
			return &util.BaseError{"Param is null"}
		}
		var realJvmType string
		if _, ok := util.SchemeTypeMAP[v.Dtype]; ok {
			arg.JavaType = util.SchemeTypeMAP[v.Dtype]
			if v.Dtype == util.SchemaArray {
				realJvmType = util.JavaList
				if v.Items != nil {
					if val, ok := v.Items["x-java-class"]; ok {
						realJvmType = fmt.Sprintf("L%s;", val)
					}
					if valType, ok := v.Items["type"]; ok {
						realJvmType = fmt.Sprintf("L%s;", valType)
					}
				}
				bytesTmp = util.S2ByteSlice(queryAgrs[v.Name])
			} else if arg.JavaType == util.JavaObject {
				realJvmType = fmt.Sprintf("L%s;", v.ObjRef.JvmClsName)
				if v.AdditionalProps != nil { //处理map
					if val, ok := v.AdditionalProps["x-java-class"]; ok {
						realJvmType = fmt.Sprintf("L%s;", val)
					} else {
						realJvmType = util.JavaMap
					}
				}
			}
			//Lcom.alibaba.dubbo.demo.user; need convert to  Lcom/alibaba/dubbo/demo/User;
			realJvmType = strings.Replace(realJvmType, ".", "/", -1)
		}
		if bytesTmp == nil {
			arg.Value, err = util.RestByteToValue(arg.JavaType, byteTmp)
			if err != nil {
				return err
			}
		} else {
			arg.Value, err = util.RestBytesToLstValue(itemType, bytesTmp)
			if err != nil {
				return err
			}
		}

		if realJvmType != "" {
			arg.JavaType = realJvmType
		}
		dubboArgs[i] = *arg
		i++
	}

	req.SetArguments(dubboArgs)

	return nil
}

func preHandleToDubbo(req *http.Request) (*invocation.Invocation, string) {
	inv := new(invocation.Invocation)
	inv.MicroServiceName = chassisconfig.SelfServiceName
	inv.Version = chassisconfig.SelfVersion
	inv.AppID = chassisconfig.GlobalDefinition.AppID

	inv.Protocol = "dubbo"
	inv.URLPathFormat = req.URL.Path
	inv.Reply = &dubboclient.WrapResponse{nil}
	source := stringutil.SplitFirstSep(req.RemoteAddr, ":")
	return inv, source
}

func TransparentForwardHandler(w http.ResponseWriter, r *http.Request) {
	inv, _ := preHandleToDubbo(r)
	dubboCtx := &dubbo.InvokeContext{dubbo.NewDubboRequest(), &dubbo.DubboRsp{}, nil, "", ""}
	err := ConvertHttpReqToDubboReq(r, dubboCtx, inv)
	if err != nil {
		lager.Logger.Error("Invalid Request :", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	inv.Args = dubboCtx.Req

	c, err := handler.GetChain(common.Provider, mesherCommon.ChainProviderIncoming)
	if err != nil {
		lager.Logger.Error("Get Chain failed.", err)
		return
	}
	c.Next(inv, func(ir *invocation.InvocationResponse) error {
		return handleRequestForDubbo(w, inv, ir)
	})
	dubboRsp := inv.Reply.(*dubboclient.WrapResponse).Resp
	if dubboRsp != nil {
		ConvertDubboRspToRestRsp(dubboRsp, w, dubboCtx)
	}
}

func handleRequestForDubbo(w http.ResponseWriter, inv *invocation.Invocation, ir *invocation.InvocationResponse) error {
	if ir != nil {
		if ir.Err != nil {
			switch ir.Err.(type) {
			case hystrix.FallbackNullError:
				w.WriteHeader(http.StatusOK)
				ir.Status = http.StatusOK
			case hystrix.CircuitError:
				w.WriteHeader(http.StatusServiceUnavailable)
				ir.Status = http.StatusServiceUnavailable
				w.Write([]byte(ir.Err.Error()))
			case loadbalancer.LBError:
				w.WriteHeader(http.StatusBadGateway)
				ir.Status = http.StatusBadGateway
				w.Write([]byte(ir.Err.Error()))
			default:
				w.WriteHeader(http.StatusInternalServerError)
				ir.Status = http.StatusInternalServerError
				w.Write([]byte(ir.Err.Error()))
			}
			return ir.Err
		}
		if inv.Endpoint == "" {
			w.WriteHeader(http.StatusInternalServerError)
			ir.Status = http.StatusInternalServerError
			w.Write([]byte(protocol.ErrUnknown.Error()))
			return protocol.ErrUnknown
		}
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(protocol.ErrUnExpectedHandlerChainResponse.Error()))
		return protocol.ErrUnExpectedHandlerChainResponse
	}

	return nil
}
