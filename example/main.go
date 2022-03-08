package main

import (
	"context"
	"github.com/slimValue/tcp_client"
	"go.uber.org/zap"
	"io"
)

var (
	addr = "127.0.0.1:9999"
)

func main() {
	// 初始化
	conn := tcp_go.NewTcpClient(addr,
		tcp_go.ReconnectOption(),
		tcp_go.SetLogger(logger),
		tcp_go.CodecOption(customCodec{}),
		tcp_go.EventHookOption(NewEvent(logger)),
	)

	conn.Start()
}

// 协议codec
type customCodec struct{}

func (c customCodec) Decode(raw io.Reader) (msg interface{}, err error) {
	data := make([]byte, 1024)
	if _, err = io.ReadFull(raw, data); err != nil {
		return
	}
	msg = string(data)
	return
}

func (c customCodec) Encode(msg interface{}) (data []byte, err error) {
	str, _ := msg.(string)
	data = []byte(str)
	return
}

// event事件
type Event struct {
	logger tcp_go.Logger
}

func NewEvent(lg tcp_go.Logger) *Event {
	return &Event{logger: lg}
}

func (ev *Event) OnConnect(w tcp_go.IConn) bool {
	ev.logger.Debugf("on connect")
	return true
}

func (ev *Event) OnClose(w tcp_go.IConn) {
	ev.logger.Debugf("on close")
}

func (ev *Event) OnError(w tcp_go.IConn) {
	ev.logger.Debugf("on error")
}

func (ev *Event) OnReceive(ctx context.Context, w tcp_go.IConn) {
	imsg := tcp_go.MessageFromContext(ctx)
	pmsg, ok := imsg.(string)
	if !ok {
		ev.logger.Errorf(tcp_go.ErrInterface2ProtoIllegal.Error())
		return
	}
	ev.logger.Infof("receive %s", pmsg)
}

// 设置日志
var (
	logger    tcp_go.Logger
	zapLogger *zap.Logger
)

func init() {
	zapLogger, _ = zap.NewDevelopment()
	logger = zapLogger.Sugar()

	defer func() {
		_ = zapLogger.Sync()
	}()
}
