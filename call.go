package tcp_go

type Call struct {
	SeqId uint16
	Error error
	Reply interface{}
	Done  chan *Call
}

func (call *Call) HasDone() {
	call.Done <- call
}

func (cc *clientConn) RegisterCall() (call *Call, err error) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	if cc.closeFlag {
		err = ErrShutdown
		return
	}
	call = new(Call)
	call.SeqId = cc.seqId
	call.Done = make(chan *Call, 1)
	cc.pending[cc.seqId] = call
	cc.seqId++
	// 防止循环回来出现0的情况
	if cc.seqId == 0 {
		cc.seqId++
	}
	return
}

func (cc *clientConn) RemoveCall(seqId uint16) (call *Call) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	call = cc.pending[seqId]
	delete(cc.pending, seqId)
	return
}

// 请求返回结果的通道返回 err
func (cc *clientConn) terminateCalls(err error) {
	cc.sending.Lock()
	defer cc.sending.Unlock()
	cc.mu.Lock()
	defer cc.mu.Unlock()
	for _, call := range cc.pending {
		call.Error = err
		call.HasDone()
	}
}
