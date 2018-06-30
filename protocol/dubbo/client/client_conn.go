package dubboclient

import (
	"fmt"
	"github.com/ServiceComb/go-chassis/core/lager"
	"github.com/go-chassis/mesher/protocol/dubbo/dubbo"
	"github.com/go-chassis/mesher/protocol/dubbo/utils"
	"net"
	"sync"
)

type SndTask struct{}

func (this SndTask) Svc(arg interface{}) interface{} {
	dubboConn := arg.(*DubboClientConnection)
	dubboConn.MsgSndLoop()
	return nil
}

type RecvTask struct {
}

func (this RecvTask) Svc(arg interface{}) interface{} {
	dubboConn := arg.(*DubboClientConnection)
	dubboConn.MsgRecvLoop()
	return nil
}

type ProcessTask struct {
	conn    *DubboClientConnection
	rsp     *dubbo.DubboRsp
	bufBody []byte
}

func (this ProcessTask) Svc(arg interface{}) interface{} {
	if this.conn != nil {
		this.conn.ProcessBody(this.rsp, this.bufBody)
	}
	return nil
}

type DubboClientConnection struct {
	msgque     *util.MsgQueue
	remoteAddr string
	conn       *net.TCPConn
	codec      dubbo.DubboCodec
	client     *DubboClient
	mtx        sync.Mutex
	routineMgr *util.RoutineManager
	closed     bool
}

func NewDubboClientConnetction(conn *net.TCPConn, client *DubboClient, routineMgr *util.RoutineManager) *DubboClientConnection {
	tmp := new(DubboClientConnection)
	conn.SetKeepAlive(true)
	tmp.conn = conn
	tmp.codec = dubbo.DubboCodec{}
	tmp.client = client
	tmp.msgque = util.NewMsgQueue()
	tmp.closed = false
	if routineMgr == nil {
		tmp.routineMgr = util.NewRoutineManager()
	}
	return tmp
}

func (this *DubboClientConnection) Open() {
	this.routineMgr.Spawn(SndTask{}, this, fmt.Sprintf("client Snd-%s->%s", this.conn.LocalAddr().String(), this.conn.RemoteAddr().String()))
	this.routineMgr.Spawn(RecvTask{}, this, fmt.Sprintf("client Recv-%s->%s", this.conn.LocalAddr().String(), this.conn.RemoteAddr().String()))
}

func (this *DubboClientConnection) Close() {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	if this.closed {
		return
	}
	this.closed = true
	this.msgque.Deavtive()
	this.conn.Close()
}

func (this *DubboClientConnection) MsgRecvLoop() {
	//通知处理应答消息
	for {
		//先处理消息头

		buf := make([]byte, dubbo.HeaderLength)
		size, err := this.conn.Read(buf)
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				lager.Logger.Error("client Recv head time errr:", err)
				//time.Sleep(time.Second * 3)
				continue
			}
			lager.Logger.Error("client Recv head errr:", err)
			break
		}

		if size < dubbo.HeaderLength {
			continue
		}
		rsp := new(dubbo.DubboRsp)
		bodyLen := 0
		ret := this.codec.DecodeDubboRsqHead(rsp, buf, &bodyLen)
		if ret != dubbo.Success {
			lager.Logger.Info("Recv DecodeDubboRsqHead failed")
			continue
		}
		body := make([]byte, bodyLen)
		count := 0
		for {
			redBuff := body[count:]
			size, err = this.conn.Read(redBuff)
			if err != nil {
				if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					continue
				}
				//通知关闭连接
				lager.Logger.Error("Recv client body err:", err)
				goto exitloop
			}
			count += size
			if count == bodyLen {
				break
			}
		}
		this.routineMgr.Spawn(ProcessTask{this, rsp, body}, nil, fmt.Sprintf("Client ProcessTask-%d", rsp.GetID()))
	}
exitloop:
	this.Close()
}

func (this *DubboClientConnection) ProcessBody(rsp *dubbo.DubboRsp, bufBody []byte) {
	var buffer util.ReadBuffer
	buffer.SetBuffer(bufBody)
	this.codec.DecodeDubboRspBody(&buffer, rsp)
	this.HandleMsg(rsp)
}

func (this *DubboClientConnection) HandleMsg(rsp *dubbo.DubboRsp) {
	this.client.RspCallBack(rsp)
}

func (this *DubboClientConnection) SendMsg(req *dubbo.Request) {
	//这里发送Rest请求以及收发送应答
	this.msgque.Enqueue(req)
}

func (this *DubboClientConnection) MsgSndLoop() {
	for {
		msg, err := this.msgque.Dequeue()
		if err != nil {
			lager.Logger.Error("MsgSndLoop Dequeue ", err)
			break
		}
		var buffer util.WriteBuffer
		buffer.Init(0)
		this.codec.EncodeDubboReq(msg.(*dubbo.Request), &buffer)
		_, err = this.conn.Write(buffer.GetValidData())
		if err != nil {
			lager.Logger.Error("Send exception,", err)
			break
		}
	}
	this.Close()
}

func (this *DubboClientConnection) Closed() bool {
	return this.closed
}
