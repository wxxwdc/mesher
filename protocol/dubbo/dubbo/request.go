package dubbo

import (
	"github.com/go-chassis/mesher/protocol/dubbo/utils"
	"sync"
)

var G_CurMSGID int64 = 0
var msgIDMtx = sync.Mutex{}

func GenerateMsgID() int64 {
	msgIDMtx.Lock()
	defer msgIDMtx.Unlock()
	G_CurMSGID++
	return G_CurMSGID
}

type Request struct {
	DubboRpcInvocation
	msgID    int64
	status   byte
	event    bool
	twoWay   bool
	isBroken bool
	data     interface{}
}

func NewDubboRequest() *Request {
	tmp := &Request{}
	tmp.SetMsgID(GenerateMsgID())
	tmp.methodName = ""
	tmp.mVersion = DubboVersion
	tmp.status = Ok
	tmp.event = false
	tmp.twoWay = true
	tmp.isBroken = false
	tmp.arguments = nil
	tmp.attachments = make(map[string]string)
	tmp.urlPath = ""
	return tmp
}

func (p *Request) IsBroken() bool {
	return p.isBroken
}

func (p *Request) SetBroken(broken bool) {
	p.isBroken = broken

}

func (p *Request) SetEvent(event string) {
	p.event = true
	p.data = event
}

func (p *Request) GetMsgID() int64 {
	return p.msgID
}

func (p *Request) SetMsgID(id int64) {
	p.msgID = id
}

func (p *Request) GetStatus() byte {
	return p.status
}

func (p *Request) IsHeartbeat() bool {
	return p.event && HeartBeatEvent == p.data
}

func (p *Request) IsEvent() bool {
	return p.event
}

func (p *Request) SetTwoWay(is bool) {
	p.twoWay = is
}

func (p *Request) IsTwoWay() bool {
	return p.twoWay
}

func (p *Request) SetData(data interface{}) {
	p.data = data
}
func (p *Request) GetData() interface{} {
	return p.data
}

type DubboRpcInvocation struct {
	methodName  string
	mVersion    string
	arguments   []util.Argument
	attachments map[string]string
	urlPath     string
}

func (p *DubboRpcInvocation) SetVersion(ver string) {
	p.mVersion = ver
}

func (p *DubboRpcInvocation) GetAttachment(key string, defaultValue string) string {
	if _, ok := p.attachments[key]; ok {
		return p.attachments[key]
	} else {
		return defaultValue
	}
}

func (p *DubboRpcInvocation) GetAttachments() map[string]string {
	return p.attachments
}

func (p *DubboRpcInvocation) GetMethodName() string {
	return p.methodName
}

func (p *DubboRpcInvocation) SetMethodName(name string) {
	p.methodName = name
}

func (p *DubboRpcInvocation) SetAttachment(key string, value string) {
	if p.attachments == nil {
		p.attachments = make(map[string]string)
	}
	if value == "" { //is empty, remove the key
		delete(p.attachments, value)
	} else {
		p.attachments[key] = value
	}
}

func (p *DubboRpcInvocation) SetAttachments(attachs map[string]string) {
	p.attachments = attachs
}

func (p *DubboRpcInvocation) GetArguments() []util.Argument {
	return p.arguments
}

func (p *DubboRpcInvocation) SetArguments(agrs []util.Argument) {
	p.arguments = agrs
}
