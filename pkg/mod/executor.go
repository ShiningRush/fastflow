package mod

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/shiningrush/fastflow/pkg/entity"
	"github.com/shiningrush/fastflow/pkg/entity/run"
	"github.com/shiningrush/fastflow/pkg/event"
	"github.com/shiningrush/fastflow/pkg/log"
	"github.com/shiningrush/goevent"
)

const (
	ReasonSuccessAfterCanceled = "success after canceled"
	ReasonParentCancel         = "parent success but already be canceled"
)

// DefExecutor
type DefExecutor struct {
	cancelMap    sync.Map
	workerNumber int
	workerQueue  chan *entity.TaskInstance
	workerWg     sync.WaitGroup
	initWg       sync.WaitGroup
	timeout      time.Duration
	initQueue    chan *initPayload

	closeCh chan struct{}
	lock    sync.RWMutex
}

// initPayload
type initPayload struct {
	dagIns  *entity.DagInstance
	taskIns *entity.TaskInstance
}

// NewDefExecutor
func NewDefExecutor(timeout time.Duration, workers int) *DefExecutor {
	return &DefExecutor{
		workerNumber: workers,
		workerQueue:  make(chan *entity.TaskInstance),
		timeout:      timeout,
		initQueue:    make(chan *initPayload),
		closeCh:      make(chan struct{}, 1),
	}
}

// Init
func (e *DefExecutor) Init() {
	e.initWg.Add(1)
	go e.watchInitQueue()

	for i := 0; i < e.workerNumber; i++ {
		e.workerWg.Add(1)
		go e.subWorkerQueue()
	}
}

func (e *DefExecutor) subWorkerQueue() {
	for taskIns := range e.workerQueue {
		e.workerDo(taskIns)
	}
	e.workerWg.Done()
}

// CancelTaskIns
func (e *DefExecutor) CancelTaskIns(taskInsIds []string) error {
	for _, id := range taskInsIds {
		if cancel, ok := e.cancelMap.Load(id); ok {
			e.cancelMap.Delete(id)
			cancel.(context.CancelFunc)()
		}
	}

	return nil
}

func (e *DefExecutor) watchInitQueue() {
	for p := range e.initQueue {
		e.initWorkerTask(p.dagIns, p.taskIns)
	}
	e.initWg.Done()
}

func (e *DefExecutor) initWorkerTask(dagIns *entity.DagInstance, taskIns *entity.TaskInstance) {
	if _, ok := e.cancelMap.Load(taskIns.ID); ok {
		log.Warnf("task instance[%s][%s] is already running", taskIns.ID, taskIns.Status)
		return
	}

	defTimeout := e.timeout
	if taskIns.TimeoutSecs != 0 {
		defTimeout = time.Duration(taskIns.TimeoutSecs) * time.Second
	}
	c, cancel := context.WithTimeout(context.TODO(), defTimeout)
	dagIns.ShareData.Save = func(data *entity.ShareData) error {
		return GetStore().PatchDagIns(&entity.DagInstance{BaseInfo: entity.BaseInfo{ID: taskIns.DagInsID}, ShareData: data})
	}
	taskIns.InitialDep(
		run.NewDefExecuteContext(c, dagIns.ShareData, taskIns.Trace, dagIns.VarsGetter(), dagIns.VarsIterator()),
		func(instance *entity.TaskInstance) error {
			return GetStore().PatchTaskIns(instance)
		})
	e.cancelMap.Store(taskIns.ID, cancel)
	e.workerQueue <- taskIns
}

// Push task to execute
func (e *DefExecutor) Push(dagIns *entity.DagInstance, taskIns *entity.TaskInstance) {
	isActive, err := taskIns.DoPreCheck(dagIns)
	if err != nil {
		log.Errorf("do task pre-check failed:%s", err)
		return
	}

	if isActive {
		if err := GetStore().PatchTaskIns(&entity.TaskInstance{
			BaseInfo: taskIns.BaseInfo,
			Status:   taskIns.Status,
		}); err != nil {
			log.Errorf("patch task[%s] failed: %s", taskIns.ID, err)
			return
		}

		// if pre-check is active, we should not execute task
		GetParser().EntryTaskIns(taskIns)
		return
	}

	e.lock.RLock()
	defer e.lock.RUnlock()

	// try to exit the sender goroutine as early as possible.
	// try-receive and try-send select blocks are specially optimized by the standard Go compiler,
	// so they are very efficient.
	select {
	case <-e.closeCh:
		log.Info("parser has already closed, so will not execute next task instances")
		return
	default:
	}

	// init task in single queue to prevent double check map
	e.initQueue <- &initPayload{
		dagIns:  dagIns,
		taskIns: taskIns,
	}
}

func (e *DefExecutor) workerDo(taskIns *entity.TaskInstance) {
	switch taskIns.Status {
	case entity.TaskInstanceStatusInit, entity.TaskInstanceStatusEnding, entity.TaskInstanceStatusRetrying:
	default:
		log.Warnf("this task instance[%s] is not executable, status[%s]", taskIns.ID, taskIns.Status)
		return
	}

	goevent.Publish(&event.TaskBegin{
		TaskIns: taskIns,
	})
	err := e.runAction(taskIns)
	e.handleTaskError(taskIns, err)
	e.cancelMap.Delete(taskIns.ID)
	GetParser().EntryTaskIns(taskIns)
	goevent.Publish(&event.TaskCompleted{
		TaskIns: taskIns,
	})
}

func (e *DefExecutor) runAction(taskIns *entity.TaskInstance) error {
	act := ActionMap[taskIns.ActionName]
	if act == nil {
		return fmt.Errorf("action not found: %s", taskIns.ActionName)
	}

	if taskIns.Params == nil {
		return taskIns.Run(nil, act)
	}
	paramAct, ok := act.(run.ParameterAction)
	if !ok {
		return taskIns.Run(nil, act)
	}
	p := paramAct.ParameterNew()
	if p == nil {
		return taskIns.Run(nil, act)
	}
	if err := e.getFromTaskInstance(taskIns, p); err != nil {
		return fmt.Errorf("get task params from task instance failed: %w", err)
	}
	return taskIns.Run(p, act)
}

func (e *DefExecutor) getFromTaskInstance(taskIns *entity.TaskInstance, params interface{}) error {
	bs, err := json.Marshal(taskIns.Params)
	if err != nil {
		return fmt.Errorf("json marshal failed: %w", err)
	}

	if err := json.Unmarshal(bs, params); err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}

	return nil
}

// Close
func (e *DefExecutor) Close() {
	e.lock.Lock()
	defer e.lock.Unlock()

	defer close(e.closeCh)
	e.closeCh <- struct{}{}

	close(e.initQueue)
	e.initWg.Wait()
	close(e.workerQueue)
	e.workerWg.Wait()
}

func (e *DefExecutor) handleTaskError(taskIns *entity.TaskInstance, err error) {
	_, ok := e.cancelMap.Load(taskIns.ID)
	if err != nil {
		taskIns.Reason = err.Error()
		setStatus := entity.TaskInstanceStatusFailed
		if !ok {
			setStatus = entity.TaskInstanceStatusCanceled
		}

		taskIns.Reason = err.Error()
		if err := taskIns.SetStatus(setStatus); err != nil {
			log.Error("set status failed",
				"task_id", taskIns.ID,
				"err", err)
		}
		return
	}

	if ok {
		return
	}

	taskIns.Reason = ReasonSuccessAfterCanceled
	if pErr := taskIns.Patch(&entity.TaskInstance{
		BaseInfo: taskIns.BaseInfo,
		Reason:   ReasonSuccessAfterCanceled}); pErr != nil {
		log.Errorf("tag canceled task instance[%s] failed: %s", taskIns.ID, pErr)
	}
}
