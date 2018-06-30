package util

import (
	"github.com/ServiceComb/go-chassis/core/invocation"
	"github.com/go-chassis/mesher/common"
	"github.com/go-chassis/mesher/config"
)

func EqualPolicy(inv *invocation.Invocation, p *config.Policy) bool {
	if inv.MicroServiceName != p.Destination {
		return false
	}
	for k, v := range p.Tags {
		if k == common.BuildInTagApp {
			if v == "" {
				v = common.DefaultApp
			}
			if v != inv.AppID {
				return false
			}
			continue
		}
		if k == common.BuildInTagVersion {
			if v == "" {
				v = common.DefaultVersion
			}
			if v != inv.Version {
				return false
			}
			continue
		}
		t, ok := inv.Metadata[k]
		if !ok {
			return false
		}
		if _, ok := t.(string); !ok {
			return false
		}
	}
	for k, v := range inv.Metadata {
		if v != p.Tags[k] {
			return false
		}
	}
	return true

}
