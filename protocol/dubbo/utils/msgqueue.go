package util

import (
	"container/list"
	"sync"
)

const (
	MaxBufferMsg = 65535
	Actived      = 0
	Deactived    = 1
)

// thread safe queue
type MsgQueue struct {
	msgList      *list.List
	mtx          *sync.Mutex
	msgCount     int
	maxMsgNum    int
	state        int
	notEmptyCond *sync.Cond
	notFullCond  *sync.Cond
}

func NewMsgQueue() *MsgQueue {
	tmp := new(MsgQueue)
	tmp.msgList = list.New()
	tmp.mtx = new(sync.Mutex)
	tmp.msgCount = 0
	tmp.maxMsgNum = MaxBufferMsg
	tmp.state = Actived
	tmp.notEmptyCond = sync.NewCond(tmp.mtx)
	tmp.notFullCond = sync.NewCond(tmp.mtx)
	return tmp
}

func (this *MsgQueue) Enqueue(msg interface{}) error {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	if this.state == Deactived {
		return &BaseError{"Queue is deactive"}
	}
	if this.waitNotFullCond() == -1 {
		return &BaseError{"Enqueue time out"}
	}
	this.msgList.PushFront(msg)
	this.msgCount++
	this.notEmptyCond.Signal()
	return nil
}

func (this *MsgQueue) Dequeue() (interface{}, error) {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	if this.waitNotEmptyCond() == -1 {
		return nil, &BaseError{"Queue is deactive"}
	}

	iter := this.msgList.Back()
	v := iter.Value
	this.msgList.Remove(iter)
	this.msgCount--
	this.notFullCond.Signal()
	return v, nil
}

func (this *MsgQueue) isEmpty() bool {
	if this.msgCount == 0 {
		return true
	} else {
		return false
	}
}

func (this *MsgQueue) isFull() bool {
	if this.msgCount >= this.maxMsgNum {
		return true
	} else {
		return false
	}
}

func (this *MsgQueue) waitNotFullCond() int {
	var result = 0

	if this.isFull() {
		this.notFullCond.Wait()
		if this.state != Actived {
			result = -1
			return result
		}
	}
	return result
}

func (this *MsgQueue) Deavtive() {
	this.state = Deactived
	this.notEmptyCond.Broadcast()
	this.notFullCond.Broadcast()
}

func (this *MsgQueue) waitNotEmptyCond() int {
	var result = 0

	if this.isEmpty() {
		this.notEmptyCond.Wait()
		if this.state != Actived {
			result = -1
			return result
		}
	}
	return result
}
