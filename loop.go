package tcp_go

import (
	"context"
	"net"
	"sync"
)

func readLoop(c *clientConn, wg *sync.WaitGroup) {
	var (
		ctx       context.Context
		rawConn   net.Conn
		codec     ICodec
		cDone     <-chan struct{}
		eventHook IEventHook
		err       error
		msg       interface{}
	)

	rawConn = c.rawConn
	codec = c.opts.codec
	cDone = c.ctx.Done()
	eventHook = c.opts.eventHook

	defer func() {
		if p := recover(); p != nil {
			c.logger.Errorf("client read panics: %v", p)
		}
		wg.Done()
		c.logger.Debugf("readLoop go-routine exited")
		c.Close(false)
	}()

	for {
		select {
		case <-cDone: // connection closed
			c.logger.Debugf("receiving cancel signal from conn")
			return
		default:
			msg, err = codec.Decode(rawConn)
			if err != nil {
				if c.closeFlag != true {
					c.logger.Warnf("error decoding message %s", err.Error())
				} else {
					continue
				}
				return
			}

			if eventHook == nil {
				c.logger.Warnf("no onReceive found for message msg: %v", msg)
				return
			}

			ctx = NewContextWithMessage(context.Background(), msg)
			go eventHook.OnReceive(ctx, c)
		}
	}
}

func writeLoop(c *clientConn, wg *sync.WaitGroup) {
	var (
		rawConn net.Conn
		sendCh  chan []byte
		cDone   <-chan struct{}
		pkt     []byte
		err     error
	)

	rawConn = c.rawConn
	sendCh = c.sendCh
	cDone = c.ctx.Done()

	defer func() {
		if p := recover(); p != nil {
			c.logger.Errorf("client write panics: %v", p)
		}

	OuterFor:
		for {
			select {
			case pkt = <-sendCh:
				if pkt != nil {
					if _, err = rawConn.Write(pkt); err != nil {
						c.logger.Errorf("error writing data %s", err.Error())
					}
				}
			default:
				break OuterFor
			}
		}
		wg.Done()
		c.logger.Debugf("writeLoop go-routine exited")
		c.Close(false)
	}()

	for {
		select {
		case <-cDone: // connection closed
			c.logger.Debugf("receiving cancel signal from conn")
			return
		case pkt = <-sendCh:
			if pkt != nil {
				if _, err = rawConn.Write(pkt); err != nil {
					c.logger.Errorf("error writing data err: ", err.Error())
					return
				}
			}
		}
	}
}
