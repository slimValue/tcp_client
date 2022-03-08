package tcp_go

import (
	"context"
	"errors"
)

var (
	ErrWouldBlock             = errors.New("would block")
	ErrShutdown               = errors.New("client shutdown")
	ErrServerClosed           = errors.New("server has been closed")
	ErrClientClosing          = errors.New("client is closing")
	ErrInterface2ProtoIllegal = errors.New("interface to protocolMessage is illegal")
)

const (
	messageCtx = "ctx_message"
)

func NewContextWithMessage(ctx context.Context, msg interface{}) context.Context {
	return context.WithValue(ctx, messageCtx, msg)
}

func MessageFromContext(ctx context.Context) interface{} {
	return ctx.Value(messageCtx)
}
