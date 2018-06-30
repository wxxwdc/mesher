package http

import (
	"context"
	"io"
	"net/http"
	"sync"

	"errors"
	"github.com/ServiceComb/go-chassis/client/rest"
	chassisCommon "github.com/ServiceComb/go-chassis/core/common"
	chassisconfig "github.com/ServiceComb/go-chassis/core/config"
	"github.com/ServiceComb/go-chassis/core/fault"
	"github.com/ServiceComb/go-chassis/core/handler"
	"github.com/ServiceComb/go-chassis/core/invocation"
	"github.com/ServiceComb/go-chassis/core/lager"
	"github.com/ServiceComb/go-chassis/core/loadbalancer"
	"github.com/ServiceComb/go-chassis/core/util/string"
	"github.com/ServiceComb/go-chassis/third_party/forked/afex/hystrix-go/hystrix"
	"github.com/go-chassis/mesher/common"
	"github.com/go-chassis/mesher/metrics"
	"github.com/go-chassis/mesher/protocol"
	"github.com/go-chassis/mesher/resolver"
	"time"
)

var p *sync.Pool
var dr = resolver.GetDestinationResolver()
var sr = resolver.GetSourceResolver()

var (
	ErrRestFaultAbort = errors.New("injecting abort")
	ErrRestFault      = errors.New("injecting abort and delay")
	ErrNilResponse    = errors.New("http response is nil")
)

func preHandler(req *http.Request) *invocation.Invocation {
	inv := p.Get().(*invocation.Invocation)
	inv.Args = &rest.Request{Req: req}
	inv.Reply = rest.NewResponse()
	inv.Protocol = "rest"
	inv.URLPathFormat = req.URL.Path
	inv.Ctx = context.Background()
	return inv
}

func consumerPreHandler(req *http.Request) *invocation.Invocation {
	inv := preHandler(req)
	inv.SourceServiceID = chassisconfig.SelfServiceID
	inv.SourceMicroService = chassisconfig.SelfServiceName
	req.Header.Set(chassisCommon.HeaderSourceName, inv.SourceMicroService)
	return inv
}

func providerPreHandler(req *http.Request) *invocation.Invocation {
	inv := preHandler(req)
	inv.MicroServiceName = chassisconfig.SelfServiceName
	inv.Version = chassisconfig.SelfVersion
	inv.AppID = chassisconfig.GlobalDefinition.AppID
	inv.SourceMicroService = req.Header.Get(chassisCommon.HeaderSourceName)
	return inv
}

//LocalRequestHandler is for request from local
func LocalRequestHandler(w http.ResponseWriter, r *http.Request) {
	prepareRequest(r)
	inv := consumerPreHandler(r)
	source := stringutil.SplitFirstSep(r.RemoteAddr, ":")

	var err error
	h := make(map[string]string)
	for k := range r.Header {
		h[k] = r.Header.Get(k)
	}
	//Resolve Destination
	if err = dr.Resolve(source, h, r.URL.String(), &inv.MicroServiceName); err != nil {
		handleErrorResponse(inv, w, http.StatusBadRequest, err)
		return
	}

	c, err := handler.GetChain(chassisCommon.Consumer, common.ChainConsumerOutgoing)
	if err != nil {
		handleErrorResponse(inv, w, http.StatusBadGateway, err)
		lager.Logger.Error("Get chain failed", err)
		return
	}
	defer func(begin time.Time) {
		timeTaken := time.Since(begin).Seconds()
		serviceLabelValues := map[string]string{metrics.ServiceName: inv.MicroServiceName, metrics.AppID: inv.AppID, metrics.Version: inv.Version}
		metrics.DefaultPrometheusExporter.Summary(metrics.RequestLatencySeconds, timeTaken, metrics.LabelNames, serviceLabelValues)
	}(time.Now())
	var invRsp *invocation.InvocationResponse
	c.Next(inv, func(ir *invocation.InvocationResponse) error {
		//Send the request to the destination
		invRsp = ir
		if invRsp != nil {
			return invRsp.Err
		}
		return nil
	})
	resp, err := handleRequest(w, inv, invRsp)
	if err != nil {
		lager.Logger.Error("Handle request failed", err)
		return
	}
	metrics.RecordResponse(inv, resp.GetStatusCode())
}

//RemoteRequestHandler is for request from remote
func RemoteRequestHandler(w http.ResponseWriter, r *http.Request) {
	prepareRequest(r)
	inv := providerPreHandler(r)

	if inv.SourceMicroService == "" {
		source := stringutil.SplitFirstSep(r.RemoteAddr, ":")
		//Resolve Source
		si := sr.Resolve(source)
		if si == nil {
			lager.Logger.Debug("Can not resolve " + source + " to Source info")
		} else {
			inv.SourceMicroService = si.Name
		}
	}

	c, err := handler.GetChain(chassisCommon.Provider, common.ChainProviderIncoming)
	if err != nil {
		handleErrorResponse(inv, w, http.StatusBadGateway, err)
		lager.Logger.Error("Get chain failed", err)
		return
	}

	var invRsp *invocation.InvocationResponse
	c.Next(inv, func(ir *invocation.InvocationResponse) error {
		//Send the request to the destination
		invRsp = ir
		if invRsp != nil {
			return invRsp.Err
		}
		return nil
	})
	if _, err = handleRequest(w, inv, invRsp); err != nil {
		lager.Logger.Error("Handle request failed", err)
	}
}
func copyChassisResp2HttpResp(w http.ResponseWriter, resp *rest.Response) {
	postProcessResponse(resp.Resp)
	copyHeader(w.Header(), resp.Resp.Header)
	w.WriteHeader(resp.Resp.StatusCode)
	if resp == nil {
		lager.Logger.Warn("response is nil because of unknown reason, plz report issue")
		return
	}
	io.Copy(w, resp.Resp.Body)
	resp.Resp.Body.Close()
}
func handleRequest(w http.ResponseWriter, inv *invocation.Invocation, ir *invocation.InvocationResponse) (*rest.Response, error) {
	if ir != nil {
		if ir.Err != nil {
			//handler only mesher errors, ignore http response err
			switch ir.Err.(type) {
			case hystrix.FallbackNullError:
				handleErrorResponse(inv, w, http.StatusOK, nil)
			case loadbalancer.LBError:
				handleErrorResponse(inv, w, http.StatusBadGateway, ir.Err)
			case hystrix.CircuitError:
				handleErrorResponse(inv, w, http.StatusServiceUnavailable, ir.Err)
			case fault.FaultError:
				handleErrorResponse(inv, w, ir.Status, ir.Err)
			default:
				resp, ok := inv.Reply.(*rest.Response)
				if ok { // return raw transport error
					if resp.Resp != nil {
						if resp.Resp == nil {
							//resp.Resp can be nil, for example network error, must handle it
							handleErrorResponse(inv, w, http.StatusBadGateway, ir.Err)
							return nil, ir.Err
						}
						copyChassisResp2HttpResp(w, resp)
						metrics.RecordResponse(inv, resp.Resp.StatusCode)
					} else {
						// unknown error, resp is nil, e.g. connection refused
						handleErrorResponse(inv, w, http.StatusBadGateway, ir.Err)
					}
				} else { // unknown err in handler chain
					handleErrorResponse(inv, w, http.StatusInternalServerError, ir.Err)
				}
			}
			return nil, ir.Err
		}
		if inv.Endpoint == "" {
			handleErrorResponse(inv, w, http.StatusBadGateway, protocol.ErrUnknown)
			return nil, protocol.ErrUnknown
		}
		if ir.Result == nil {
			if ir.Err != nil {
				handleErrorResponse(inv, w, http.StatusBadGateway, ir.Err)
				return nil, ir.Err
			}
			handleErrorResponse(inv, w, http.StatusBadGateway, ErrNilResponse)
			return nil, protocol.ErrUnknown
		}
		resp, ok := ir.Result.(*rest.Response)
		if !ok {
			err := errors.New("invocationResponse result is not type *rest.Response")
			handleErrorResponse(inv, w, http.StatusBadGateway, err)
			return nil, err
		}
		//transparent proxy
		copyChassisResp2HttpResp(w, resp)

		return resp, nil
	} else {
		handleErrorResponse(inv, w, http.StatusBadGateway, protocol.ErrUnExpectedHandlerChainResponse)
		return nil, protocol.ErrUnExpectedHandlerChainResponse
	}

}

//handleErrorResponse return proxy errors, not err from real service
func handleErrorResponse(inv *invocation.Invocation, w http.ResponseWriter, statusCode int, err error) {
	w.WriteHeader(statusCode)
	if err != nil {
		w.Write([]byte(err.Error()))
	}
	metrics.RecordResponse(inv, statusCode)
}

func copyHeader(dst, src http.Header) {
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

func prepareRequest(req *http.Request) {
	if req.ContentLength == 0 {
		req.Body = nil
	}
	req.RequestURI = "" // client is forbidden to set RequestURI
	req.Close = false

	req.Header.Del("Connection")

}

func postProcessResponse(rsp *http.Response) {
	rsp.Header.Del("Connection")
}
