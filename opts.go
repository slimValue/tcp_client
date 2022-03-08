package tcp_go

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"sync"
	"time"
)

type ICodec interface {
	Decode(reader io.Reader) (interface{}, error)
	Encode(interface{}) ([]byte, error)
}

type IEventHook interface {
	OnConnect(IConn) bool
	OnReceive(context.Context, IConn)
	OnClose(IConn)
	OnError(IConn)
}

// Logger is used for logging formatted messages.
type Logger interface {
	// Debugf logs messages at DEBUG level.
	Debugf(format string, args ...interface{})
	// Infof logs messages at INFO level.
	Infof(format string, args ...interface{})
	// Warnf logs messages at WARN level.
	Warnf(format string, args ...interface{})
	// Errorf logs messages at ERROR level.
	Errorf(format string, args ...interface{})
	// Fatalf logs messages at FATAL level.
	Fatalf(format string, args ...interface{})
}

type options struct {
	tlsCfg    *tls.Config
	codec     ICodec
	eventHook IEventHook

	workerSize      int
	bufferSize      int
	reconnect       bool
	connectInterval time.Duration

	logger Logger
}

type clientConn struct {
	addr      string
	opts      options
	rawConn   net.Conn
	once      *sync.Once
	wg        *sync.WaitGroup
	sendCh    chan []byte
	mu        sync.Mutex // guards following
	heart     uint64
	ctx       context.Context
	cancel    context.CancelFunc
	logger    Logger
	closeFlag bool

	pending map[uint16]*Call
	sending sync.Mutex
	seqId   uint16
}

type Option func(*options)

func SetLogger(lg Logger) Option {
	return func(o *options) {
		o.logger = lg
	}
}

// 绑定codec协议
func CodecOption(codec ICodec) Option {
	return func(o *options) {
		o.codec = codec
	}
}

// 绑定事件
func EventHookOption(event IEventHook) Option {
	return func(o *options) {
		o.eventHook = event
	}
}

// 是否重连
func ReconnectOption() Option {
	return func(o *options) {
		o.reconnect = true
	}
}

// 重连等待间隔
func ConnectIntervalOption(interval time.Duration) Option {
	return func(o *options) {
		o.connectInterval = interval
	}
}
