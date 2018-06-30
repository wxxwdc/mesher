package util_test

import (
	"github.com/ServiceComb/go-chassis/core/invocation"
	"github.com/go-chassis/mesher/config"
	"gopkg.in/yaml.v2"
	"testing"
)

var file []byte = []byte(`
policy:
 default:
   - destination: ShoppingCart
     tags:
        version: 0.1
        app: CSE
     loadBalancing:
        strategy: roundrobin
 ServiceClient:
   - destination: ShoppingCart
     tags:
        version: 0.1
        app: CSE
     loadBalancing:
        strategy: RoundRobin
   - destination: ShoppingCart
     tags:
        version: 0.1
        app: CSE
        project: X
     loadBalancing:
        strategy: RoundRobin
   - destination: ShoppingCart
     tags:
        version: 0.1
        app: ""
        project: X
     loadBalancing:
        strategy: RoundRobin
   - destination: ShoppingCart
     tags:
        version:
        app: CSE
        project: X
     loadBalancing:
        strategy: RoundRobin
   - destination: ShoppingCart
     tags:
        version: 0.1
        app: CSE
        project: X
     loadBalancing:
        strategy: RoundRobin
 `)

func TestEqualPolicy(t *testing.T) {
	c := &config.MesherConfig{}
	if err := yaml.Unmarshal([]byte(file), c); err != nil {
		t.Error(err)
	}
	i := &invocation.Invocation{
		MicroServiceName: "ShoppingCart",
	}

	i.Version = "0.1"
	i.AppID = "default"

	i.Version = "0.1"
	i.AppID = "CSE"

	i.Metadata = map[string]interface{}{
		"project": "X",
	}

	//CASE :service name and destination are different
	i = &invocation.Invocation{
		MicroServiceName: "sockshop",
	}

	//Empty app tag
	i = &invocation.Invocation{
		MicroServiceName: "ShoppingCart",
	}

	//Empty version tag
	i = &invocation.Invocation{
		MicroServiceName: "ShoppingCart",
	}

	i.AppID = "CSE"

	//metadata parameter value is not in string format
	i = &invocation.Invocation{
		MicroServiceName: "ShoppingCart",
	}

	i.AppID = "CSE"
	i.Version = "0.1"

	i.Metadata = map[string]interface{}{
		"project": 22,
	}

}
