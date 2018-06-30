package server

import (
	"github.com/go-chassis/mesher/protocol/dubbo/dubbo"
	"github.com/go-chassis/mesher/protocol/dubbo/proxy"
	"github.com/go-chassis/mesher/protocol/dubbo/utils"
	"fmt"
	"github.com/ServiceComb/go-chassis/core/lager"
	"net"
	"sync"
)

type SndTask struct{}

func (this SndTask) Svc(arg interface{}) interface{} {
	dubboConn := arg.(*DubboConnection)
	dubboConn.MsgSndLoop()
	return nil
}

type RecvTask struct {
}

func (this RecvTask) Svc(arg interface{}) interface{} {
	dubboConn := arg.(*DubboConnection)
	dubboConn.MsgRecvLoop()
	return nil
}

type ProcessTask struct {
	conn    *DubboConnection
	req     *dubbo.Request
	bufBody []byte
}

func (this ProcessTask) Svc(arg interface{}) interface{} {
	if this.conn != nil {
		this.conn.ProcessBody(this.req, this.bufBody)
	}
	return nil
}

type DubboConnection struct {
	msgque     *util.MsgQueue
	remoteAddr string
	conn       *net.TCPConn
	codec      dubbo.DubboCodec
	mtx        sync.Mutex
	routineMgr *util.RoutineManager
	closed     bool
}

func NewDubboConnetction(conn *net.TCPConn, routineMgr *util.RoutineManager) *DubboConnection {
	tmp := new(DubboConnection)
	tmp.conn = conn
	tmp.codec = dubbo.DubboCodec{}
	tmp.msgque = util.NewMsgQueue()
	tmp.remoteAddr = conn.RemoteAddr().String()
	tmp.closed = false
	if routineMgr == nil {
		tmp.routineMgr = util.NewRoutineManager()
	}
	return tmp
}

func (this *DubboConnection) Open() {
	this.routineMgr.Spawn(SndTask{}, this, fmt.Sprintf("Snd-%s->%s", this.conn.LocalAddr().String(), this.conn.RemoteAddr().String()))
	this.routineMgr.Spawn(RecvTask{}, this, fmt.Sprintf("Recv-%s->%s", this.conn.LocalAddr().String(), this.conn.RemoteAddr().String()))
}

func (this *DubboConnection) Close() {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	if this.closed {
		return
	}
	this.closed = true
	this.msgque.Deavtive()
	this.conn.Close()
}

func (this *DubboConnection) MsgRecvLoop() {
	//通知处理应答消息
	for {
		//先处理消息头
		buf := make([]byte, dubbo.HeaderLength)
		size, err := this.conn.Read(buf)
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				lager.Logger.Error("Dubbo server Recv head:", err)
				continue
			}
			lager.Logger.Error("Dubbo server Recv head:", err)
			break
		}

		if size < dubbo.HeaderLength {
			lager.Logger.Info("Invalid msg head")
			continue
		}
		req := new(dubbo.Request)
		bodyLen := 0
		ret := this.codec.DecodeDubboReqHead(req, buf, &bodyLen)
		if ret != dubbo.Success {
			lager.Logger.Info("Invalid msg head")
			continue
		}
		body := make([]byte, bodyLen)
		count := 0
		for {
			redBuff := body[count:]
			size, err = this.conn.Read(redBuff)

			if err != nil {
				//通知关闭连接
				lager.Logger.Error("Recv:", err)
				goto exitloop
			}
			count += size
			if count == bodyLen {
				break
			}
		}
		this.routineMgr.Spawn(ProcessTask{this, req, body}, nil, fmt.Sprintf("ProcessTask-%d", req.GetMsgID()))
	}
exitloop:
	this.Close()
}

func (this *DubboConnection) ProcessBody(req *dubbo.Request, bufBody []byte) {
	var buffer util.ReadBuffer
	buffer.SetBuffer(bufBody)
	this.codec.DecodeDubboReqBody(req, &buffer)
	this.HandleMsg(req)
}

func (this *DubboConnection) HandleMsg(req *dubbo.Request) {
	//这里发送Rest请求以及收发送应答
	ctx := &dubbo.InvokeContext{req, &dubbo.DubboRsp{}, nil, "", this.remoteAddr}
	ctx.Rsp.Init()
	ctx.Rsp.SetID(req.GetMsgID())
	if req.IsHeartbeat() {
		ctx.Rsp.SetValue(nil)
		ctx.Rsp.SetEvent(true)
		ctx.Rsp.SetStatus(dubbo.Ok)
	} else {
		//这里重新分配MSGID
		srcMsgID := ctx.Req.GetMsgID()
		dstMsgID := dubbo.GenerateMsgID()
		lager.Logger.Info(fmt.Sprintf("dubbo2dubbo srcMsgID=%d, newMsgID=%d", srcMsgID, dstMsgID))
		ctx.Req.SetMsgID(dstMsgID)

		err := dubboproxy.Handle(ctx)
		if err != nil {
			ctx.Rsp.SetErrorMsg(err.Error())
			lager.Logger.Error("request ", err)
			ctx.Rsp.SetStatus(dubbo.ServerError)
		}
		ctx.Req.SetMsgID(srcMsgID)
		ctx.Rsp.SetID(srcMsgID)
	}
	if req.IsTwoWay() {
		this.msgque.Enqueue(ctx.Rsp)
	}
}

func (this *DubboConnection) MsgSndLoop() {
	for {
		msg, err := this.msgque.Dequeue()
		if err != nil {
			lager.Logger.Error("MsgSndLoop Dequeue ", err)
			break
		}
		var buffer util.WriteBuffer
		buffer.Init(0)
		this.codec.EncodeDubboRsp(msg.(*dubbo.DubboRsp), &buffer)
		_, err = this.conn.Write(buffer.GetValidData())
		if err != nil {
			lager.Logger.Error("Send exception,", err)
			break
		}
	}
	this.Close()
}
