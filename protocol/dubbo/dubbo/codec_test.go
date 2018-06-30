package dubbo

import (
	"testing"

	"github.com/go-chassis/gohessian"
	"github.com/go-chassis/mesher/protocol/dubbo/utils"

	"github.com/stretchr/testify/assert"
)

func TestDubboCodec_DecodeDubboReqBody(t *testing.T) {
	t.Log("If returns of rbf.ReadObject() is nil, should not panic")
	d := &DubboCodec{}
	resp := &DubboRsp{}
	resp.Init()
	resp.SetStatus(ServerError)

	rbf := &util.ReadBuffer{}
	rbf.Init(0)
	rbf.SetBuffer([]byte{hessian.BC_NULL})
	c := make([]byte, 10)
	_, err := rbf.Read(c)
	assert.NoError(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, hessian.BC_NULL, c[0])
	obj, err := rbf.ReadObject()
	assert.Nil(t, err)
	assert.Nil(t, obj)
	d.DecodeDubboRspBody(rbf, resp)
}
