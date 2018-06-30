package handler

import (
	"errors"

	"fmt"
	"github.com/ServiceComb/go-chassis/core/handler"
	"github.com/ServiceComb/go-chassis/core/invocation"
	"github.com/go-chassis/mesher/cmd"
	"github.com/go-chassis/mesher/common"
)

const LocalSelection = "local-selection"

type LocalSelectionHandler struct {
}

func (ls *LocalSelectionHandler) Handle(chain *handler.Chain, inv *invocation.Invocation, cb invocation.ResponseCallBack) {
	// if work as sidecar and handler request from remote,then endpoint should be localhost:port
	inv.Endpoint = cmd.Configs.PortsMap[inv.Protocol]
	if inv.Endpoint == "" {
		r := &invocation.InvocationResponse{
			Err: errors.New(
				fmt.Sprintf("[%s] is not supported, [%s] didn't set env [%s] or cmd parameter --service-ports before mesher start",
					inv.Protocol, inv.MicroServiceName, common.EnvServicePorts)),
		}
		cb(r)
		return
	}
	chain.Next(inv, func(r *invocation.InvocationResponse) error {
		return cb(r)
	})
}

func (ls *LocalSelectionHandler) Name() string {
	return LocalSelection
}
func New() handler.Handler {
	return &LocalSelectionHandler{}
}
func init() {
	handler.RegisterHandler(LocalSelection, New)
}
