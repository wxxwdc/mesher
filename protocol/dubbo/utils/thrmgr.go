package util

import (
	"github.com/ServiceComb/go-chassis/core/lager"
	"sync"
)

//realise a thread group wait
type ThreadGroupWait struct {
	count int
	mtx   sync.Mutex
	cond  *sync.Cond
}

func NewThreadGroupWait() *ThreadGroupWait {
	tmp := new(ThreadGroupWait)
	tmp.count = 1
	tmp.cond = sync.NewCond(&tmp.mtx)
	return tmp
}

func (this *ThreadGroupWait) Add(count int) {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	this.count++
}

func (this *ThreadGroupWait) Done() {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	this.count--
	if this.count < 0 {
		this.cond.Broadcast()
	}
}

func (this *ThreadGroupWait) Wait() {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	this.cond.Wait()
}

//Routine task interface
type RoutineTask interface {
	Svc(agrs interface{}) interface{}
}

//route manager
type RoutineManager struct {
	wg *ThreadGroupWait
}

func NewRoutineManager() *RoutineManager {
	tmp := new(RoutineManager)
	tmp.wg = NewThreadGroupWait()
	return tmp
}

func (this *RoutineManager) Wait() {
	this.wg.Wait()
}

func (this *RoutineManager) Spawn(task RoutineTask, agrs interface{}, routineName string) {
	lager.Logger.Info("Routine spawn:" + routineName)
	this.wg.Add(1)
	go this.spawn(task, agrs, routineName)
}

func (this *RoutineManager) spawn(task RoutineTask, agrs interface{}, routineName string) {
	task.Svc(agrs)
	this.wg.Done()
	lager.Logger.Info("Routine exit:" + routineName)
}

func (this *RoutineManager) Done() {
	this.wg.Done()
}
