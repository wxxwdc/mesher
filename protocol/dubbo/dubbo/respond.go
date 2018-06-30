package dubbo

const (
	Ok                             = byte(20)
	ClientTimeout                  = byte(30)
	ServerTimeout                  = byte(31)
	BadRequest                     = byte(40)
	BadResponse                    = byte(50)
	ServiceNotFound                = byte(60)
	ServiceError                   = byte(70)
	ServerError                    = byte(80)
	ClentError                     = byte(90)
	ServerThreadPoolExhaustedError = byte(100)
)
const (
	ResponseWithException = byte(0)
	ResponseValue         = byte(1)
	ResponseNullValue     = byte(2)
)

type DubboRsp struct {
	DubboRpcResult
	mID       int64
	mVersion  string
	mStatus   byte
	mEvent    bool
	mErrorMsg string
}

func (p *DubboRsp) Init() {
	p.mID = 0
	p.mVersion = "0.0.0"
	p.mStatus = Ok
	p.mEvent = false
	p.mErrorMsg = ""
	//p.mResult = nil
}
func (p *DubboRsp) IsHeartbeat() bool {
	return p.mEvent
}

func (p *DubboRsp) SetEvent(bEvt bool) {
	p.mEvent = bEvt
}

func (p *DubboRsp) GetStatus() byte {
	return p.mStatus
}

func (p *DubboRsp) SetStatus(status byte) {
	p.mStatus = status
}

func (p *DubboRsp) GetID() int64 {
	return p.mID
}

func (p *DubboRsp) SetID(reqID int64) {
	p.mID = reqID
}

func (p *DubboRsp) GetErrorMsg() string {
	return p.mErrorMsg
}

func (p *DubboRsp) SetErrorMsg(err string) {
	p.mErrorMsg = err
}

type DubboRpcResult struct {
	attchments map[string]string
	exception  interface{}
	value      interface{}
}

func NewDubboRpcResult() *DubboRpcResult {
	return &DubboRpcResult{make(map[string]string), nil, nil}
}
func (p *DubboRpcResult) GetValue() interface{} {
	return p.value
}

func (p *DubboRpcResult) SetValue(v interface{}) {
	p.value = v
}

func (p *DubboRpcResult) GetException() interface{} {
	return p.exception
}

func (p *DubboRpcResult) SetException(e interface{}) {
	p.exception = e
}

func (p *DubboRpcResult) GetAttachments() map[string]string {
	return p.attchments
}

func (p *DubboRpcResult) SetAttachments(attach map[string]string) {
	p.attchments = attach
}
