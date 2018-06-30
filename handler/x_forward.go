package handler

import (
	"github.com/ServiceComb/go-chassis/client/rest"
	"github.com/ServiceComb/go-chassis/core/handler"
	"github.com/ServiceComb/go-chassis/core/invocation"
)

const XForward = "x-forward"

type XForwardHandler struct {
}

func (h *XForwardHandler) Handle(chain *handler.Chain, inv *invocation.Invocation, cb invocation.ResponseCallBack) {
	orgReq, ok := inv.Args.(*rest.Request)
	if ok && orgReq.Req.Header["X-Forwarded-Host"] == nil {
		orgHost := orgReq.Req.Header["Host"]
		orgReq.Req.Header["X-Forwarded-Host"] = orgHost
	}
	chain.Next(inv, func(r *invocation.InvocationResponse) error {
		return cb(r)
	})
}

func (h *XForwardHandler) Name() string { return XForward }
func NewHandler() handler.Handler       { return &XForwardHandler{} }

func init() { handler.RegisterHandler(XForward, NewHandler) }
