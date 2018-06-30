package dubboclient

import (
	"fmt"
	"github.com/ServiceComb/go-chassis/core/lager"
	"github.com/go-chassis/mesher/protocol/dubbo/dubbo"
	"github.com/go-chassis/mesher/protocol/dubbo/utils"
	"net"
	"sync"
	"time"
)

type DubboClient struct {
	addr          string
	mtx           sync.Mutex
	mapMutex      sync.Mutex
	msgWaitRspMap map[int64]*RespondResult
	conn          *DubboClientConnection
	closed        bool
	routeMgr      *util.RoutineManager
}
type WrapResponse struct {
	Resp *dubbo.DubboRsp
}

var CachedClients *ClientMgr

func init() {
	CachedClients = NewClientMgr()
}

type RespondResult struct {
	Rsp  *dubbo.DubboRsp
	Wait *chan int
}

type ClientMgr struct {
	mapMutex sync.Mutex
	clients  map[string]*DubboClient
}

func NewClientMgr() *ClientMgr {
	tmp := new(ClientMgr)
	tmp.clients = make(map[string]*DubboClient)
	return tmp
}

func (this *ClientMgr) GetClient(addr string) (*DubboClient, error) {
	this.mapMutex.Lock()
	defer this.mapMutex.Unlock()
	if tmp, ok := this.clients[addr]; ok {
		if !tmp.Closed() {
			lager.Logger.Info("GetClient from cached addr:" + addr)
			return tmp, nil
		} else {
			err := tmp.ReOpen()
			lager.Logger.Info("GetClient repopen addr:" + addr)
			if err != nil {
				delete(this.clients, addr)
				return nil, err
			} else {
				return tmp, nil
			}
		}
	}
	lager.Logger.Info("GetClient from new open addr:" + addr)
	tmp := NewDubboClient(addr, nil)
	err := tmp.Open()
	if err != nil {
		return nil, err
	} else {
		this.clients[addr] = tmp
		return tmp, nil
	}
}

func NewDubboClient(addr string, routeMgr *util.RoutineManager) *DubboClient {
	tmp := &DubboClient{}
	tmp.addr = addr

	tmp.conn = nil
	tmp.closed = true
	tmp.msgWaitRspMap = make(map[int64]*RespondResult)
	if routeMgr == nil {
		tmp.routeMgr = util.NewRoutineManager()
	}
	return tmp
}
func (this *DubboClient) GetAddr() string {
	return this.addr
}
func (this *DubboClient) ReOpen() error {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	this.close()
	return this.open()
}

func (this *DubboClient) Open() error {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	return this.open()
}

func (this *DubboClient) open() error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", this.addr)
	if err != nil {
		lager.Logger.Error(err.Error(), err)
		return err
	}
	conn, errDial := net.DialTCP("tcp", nil, tcpAddr)

	if errDial != nil {
		lager.Logger.Error("the addr: "+this.addr, errDial)
		return errDial
	}
	this.conn = NewDubboClientConnetction(conn, this, nil)
	this.conn.Open()
	this.closed = false
	return nil
}

func (this *DubboClient) Close() {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	this.close()
	this.routeMgr.Done()
	this.routeMgr.Wait()
}

func (this *DubboClient) close() {
	if this.closed {
		return
	}
	this.closed = true
	this.mapMutex.Lock()
	for _, v := range this.msgWaitRspMap {
		*v.Wait <- 1
	}
	this.msgWaitRspMap = make(map[int64]*RespondResult) //清空map
	this.mapMutex.Unlock()
	this.conn.Close()
}

func (this *DubboClient) AddWaitMsg(msgID int64, result *RespondResult) {
	this.mapMutex.Lock()
	if this.msgWaitRspMap != nil {
		this.msgWaitRspMap[msgID] = result
	}
	this.mapMutex.Unlock()
}

func (this *DubboClient) RemoveWaitMsg(msgID int64) {
	this.mapMutex.Lock()
	if this.msgWaitRspMap != nil {
		delete(this.msgWaitRspMap, msgID)
	}
	this.mapMutex.Unlock()
}

func (this *DubboClient) Svc(agr interface{}) interface{} {
	this.conn.SendMsg(agr.(*dubbo.Request))
	return nil
}

func (this *DubboClient) Send(dubboReq *dubbo.Request) (*dubbo.DubboRsp, error) {
	this.mapMutex.Lock()
	if this.closed {
		this.open()
	}
	this.mapMutex.Unlock()
	wait := make(chan int)
	result := &RespondResult{nil, &wait}
	msgID := dubboReq.GetMsgID()
	this.AddWaitMsg(msgID, result)

	this.routeMgr.Spawn(this, dubboReq, fmt.Sprintf("SndMsgID-%d", dubboReq.GetMsgID()))
	var timeout bool = false
	select {
	case <-wait:
		timeout = false
	case <-time.After(300 * time.Second):
		timeout = true
	}
	if this.closed {
		lager.Logger.Info("Client been closed.")
		return nil, &util.BaseError{"Client been closed."}
	}
	this.RemoveWaitMsg(msgID)
	if timeout {
		dubboReq.SetBroken(true)
		lager.Logger.Info("Client send timeout.")
		return nil, &util.BaseError{"timeout"}
	} else {
		return result.Rsp, nil
	}
}

func (this *DubboClient) RspCallBack(rsp *dubbo.DubboRsp) {
	msgID := rsp.GetID()
	var result *RespondResult
	this.mapMutex.Lock()
	defer this.mapMutex.Unlock()
	if this.msgWaitRspMap == nil {
		return
	}
	if _, ok := this.msgWaitRspMap[msgID]; ok {
		result = this.msgWaitRspMap[msgID]
		result.Rsp = rsp
		*result.Wait <- 1
	}
}
func (this *DubboClient) Closed() bool {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	if this.conn.Closed() {
		this.close()
	}
	return this.closed
}
