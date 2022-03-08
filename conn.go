package tcp_go

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"time"
)

const (
	DefBufferSize   = 1024
	ConnectInterval = 2 * time.Second
)

type IConn interface {
	Start()
	Close(isForce bool)
	SetHeart(heart uint64)
	Write(message interface{}) error

	RegisterCall() (call *Call, err error)
	RemoveCall(seqId uint16) (call *Call)
}

func NewTcpClient(addr string, opt ...Option) *clientConn {
	var opts options
	for _, o := range opt {
		o(&opts)
	}
	if opts.codec == nil {
		panic("unknown codec")
	}
	if opts.eventHook == nil {
		panic("unknown eventHook")
	}
	if opts.bufferSize <= 0 {
		opts.bufferSize = DefBufferSize
	}

	if opts.connectInterval == 0 {
		opts.connectInterval = ConnectInterval
	}

	conn := Dial(addr, opts)
	return newClientConnWithOptions(conn, opts)
}

func newClientConnWithOptions(c net.Conn, opts options) *clientConn {
	cc := &clientConn{
		addr:    c.RemoteAddr().String(),
		opts:    opts,
		rawConn: c,
		once:    &sync.Once{},
		wg:      &sync.WaitGroup{},
		sendCh:  make(chan []byte, opts.bufferSize),
		pending: make(map[uint16]*Call),
		seqId:   1,
		logger:  opts.logger,
	}
	cc.ctx, cc.cancel = context.WithCancel(context.Background())
	return cc
}

func (cc *clientConn) Start() {
	cc.logger.Infof("conn start, <%v -> %v>", cc.rawConn.LocalAddr(), cc.rawConn.RemoteAddr())

	eventHook := cc.opts.eventHook
	if eventHook != nil {
		eventHook.OnConnect(cc)
	}

	cc.closeFlag = false

	cc.wg.Add(2)
	go readLoop(cc, cc.wg)
	go writeLoop(cc, cc.wg)
}

// isForce: 是否主动关闭
func (cc *clientConn) Close(isForce bool) {
	cc.once.Do(func() {
		cc.logger.Debugf("conn is closing  <%v -> %v>", cc.rawConn.LocalAddr(), cc.rawConn.RemoteAddr())

		// callback on close
		eventHook := cc.opts.eventHook
		if eventHook != nil {
			eventHook.OnClose(cc)
		}
		cc.closeFlag = true

		cc.terminateCalls(ErrClientClosing)

		cc.rawConn.Close()

		cc.mu.Lock()
		cc.cancel()
		cc.mu.Unlock()

		cc.wg.Wait()

		close(cc.sendCh)

		if cc.opts.reconnect && !isForce {
			cc.reconnect()
		}
	})
}

func Dial(addr string, opts options) (conn net.Conn) {
OuterLoop:
	for {
		var err error
		if opts.tlsCfg != nil {
			conn, err = tls.Dial("tcp", addr, opts.tlsCfg)
		} else {
			conn, err = net.Dial("tcp", addr)
		}

		if err == nil {
			break OuterLoop
		}

		if opts.connectInterval == 0 {
			panic("[tcp连接失败] " + err.Error())
		}

		opts.logger.Debugf("[tcp重连中...] %s" + err.Error())
		time.Sleep(opts.connectInterval)
	}
	return
}

// reconnect reconnects and returns a new *clientConn.
func (cc *clientConn) reconnect() {
	conn := Dial(cc.addr, cc.opts)
	// copy the newly-created *clientConn to cc, so after
	// reconnect returned cc will be updated to new one.
	*cc = *newClientConnWithOptions(conn, cc.opts)
	cc.Start()
}

// Write writes a message to the client.
func (cc *clientConn) Write(message interface{}) error {
	return asyncWrite(cc, message)
}

func asyncWrite(c *clientConn, m interface{}) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = ErrServerClosed
		}
	}()

	var (
		pkt    []byte
		sendCh chan []byte
	)

	pkt, err = c.opts.codec.Encode(m)
	if err != nil {
		c.logger.Errorf("asyncWrite err: %s", err.Error())
		return
	}

	sendCh = c.sendCh
	select {
	case sendCh <- pkt:
		err = nil
	default:
		err = ErrWouldBlock
	}
	return
}

func (cc *clientConn) SetHeart(heart uint64) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.logger.Debugf("heartbeat time: %d", heart)
	cc.heart = heart
}
