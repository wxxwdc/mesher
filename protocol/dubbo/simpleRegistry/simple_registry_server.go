package simgleRegistry

import (
	"github.com/ServiceComb/go-chassis/core/lager"
	"github.com/ServiceComb/go-chassis/core/server"
	"github.com/go-chassis/mesher/protocol/dubbo/dubbo"
	"github.com/go-chassis/mesher/protocol/dubbo/utils"
	"net"
	"sync"
)

const (
	NAME = "dubboSimpleRegistry"
)

func init() {
	server.InstallPlugin(NAME, newServer)
}

func newServer(opts server.Options) server.ProtocolServer {

	return &SimDubboRegistryServer{
		opts: opts,
	}
}

//-----------------------SimDubboRegistry server---------------------------
type SimDubboRegistryServer struct {
	opts       server.Options
	mux        sync.RWMutex
	exit       chan chan error
	routineMgr *util.RoutineManager
}

func (d *SimDubboRegistryServer) String() string {
	return NAME
}

func (d *SimDubboRegistryServer) Init(opts ...server.Options) error {
	lager.Logger.Info("Dubbo Simple Registry server init.")
	return nil
}

func (d *SimDubboRegistryServer) Register(schema interface{}, options ...server.RegisterOption) (string, error) {
	return "", nil
}

func (d *SimDubboRegistryServer) Stop() error {
	return nil
}

func (d *SimDubboRegistryServer) Start() error {
	d.Init()
	host, _, err := net.SplitHostPort(d.opts.Address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return &util.BaseError{ErrMsg: "invalid host"}
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", d.opts.Address)
	if err != nil {
		lager.Logger.Error("ResolveTCPAddr err: ", err)
		return err
	}
	l, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		lager.Logger.Error("listening falied, reason:", err)
		return err
	}
	go d.AcceptLoop(l)
	return nil
}

func (d *SimDubboRegistryServer) AcceptLoop(l *net.TCPListener) {
	for {
		for {
			conn, err := l.Accept()
			if err != nil {
				lager.Logger.Error("tcp conn error: ", err)
				continue
			}

			lager.Logger.Debug("Received message")

			go handleConn(conn)
		}

	}
	defer l.Close()
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	var buf []byte
	buf = make([]byte, 1024)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			return
		}
		req := &dubbo.Request{}
		codec := dubbo.DubboCodec{}

		var bodyLen int
		codec.DecodeDubboReqHead(req, buf[0:dubbo.HeaderLength], &bodyLen)

		SendVoidRespond(conn, req)
	}
}

func SendVoidRespond(conn net.Conn, req *dubbo.Request) {
	var rsp dubbo.DubboRsp
	var wBuf util.WriteBuffer
	wBuf = util.WriteBuffer{}
	wBuf.Init(1024)
	rsp = dubbo.DubboRsp{}
	rsp.Init()
	rsp.SetEvent(req.IsEvent())
	rsp.SetID(req.GetMsgID())
	rsp.SetValue(nil)
	codec := dubbo.DubboCodec{}
	codec.EncodeDubboRsp(&rsp, &wBuf)
	conn.Write(wBuf.GetValidData())
}
