package mod

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/shiningrush/fastflow/pkg/entity"
	"github.com/shiningrush/fastflow/pkg/event"
	"github.com/shiningrush/fastflow/pkg/log"
	"github.com/shiningrush/fastflow/pkg/utils"
	"github.com/shiningrush/goevent"
	"github.com/spaolacci/murmur3"
)

// DefParser
type DefParser struct {
	workerNumber int
	workerQueue  []chan *entity.TaskInstance
	workerWg     sync.WaitGroup
	taskTrees    sync.Map
	taskTimeout  time.Duration

	closeCh chan struct{}
	lock    sync.RWMutex
}

// NewDefParser
func NewDefParser(workerNumber int, taskTimeout time.Duration) *DefParser {
	return &DefParser{
		workerNumber: workerNumber,
		workerWg:     sync.WaitGroup{},
		closeCh:      make(chan struct{}),
		taskTimeout:  taskTimeout,
	}
}

// Init
func (p *DefParser) Init() {
	p.workerWg.Add(1)
	go p.startWatcher(p.watchScheduledDagIns)
	p.workerWg.Add(1)
	go p.startWatcher(p.watchDagInsCmd)

	for i := 0; i < p.workerNumber; i++ {
		p.workerWg.Add(1)
		ch := make(chan *entity.TaskInstance, 50)
		p.workerQueue = append(p.workerQueue, ch)
		go p.goWorker(ch)
	}
	if err := p.initialRunningDagIns(); err != nil {
		log.Fatalf("parser init dags failed: %s", err)
	}
}

func (p *DefParser) startWatcher(do func() error) {
	timerCh := time.Tick(time.Second)
	closed := false
	for !closed {
		select {
		case <-p.closeCh:
			closed = true
		case <-timerCh:
			if err := do(); err != nil {
				p.handleErr(err)
			}
		}
	}
	p.workerWg.Done()
}

func (p *DefParser) watchScheduledDagIns() (err error) {
	start := time.Now()
	e := &event.ParseScheduleDagInsCompleted{}
	defer func() {
		if err != nil {
			err = fmt.Errorf("watch scheduled dag ins failed: %w", err)
			e.Error = err
		}
		e.ElapsedMs = time.Now().Sub(start).Milliseconds()
		goevent.Publish(e)
	}()

	dagIns, err := GetStore().ListDagInstance(&ListDagInstanceInput{
		Worker: GetKeeper().WorkerKey(),
		Status: []entity.DagInstanceStatus{
			entity.DagInstanceStatusScheduled,
		},
	})
	if err != nil {
		return
	}
	for i := range dagIns {
		if err = p.parseScheduleDagIns(dagIns[i]); err != nil {
			return
		}
		p.InitialDagIns(dagIns[i])
	}
	return
}

func (p *DefParser) watchDagInsCmd() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("watch dag command failed: %w", err)
		}
	}()

	dagIns, err := GetStore().ListDagInstance(&ListDagInstanceInput{
		Worker: GetKeeper().WorkerKey(),
		HasCmd: true,
	})
	if err != nil {
		return err
	}
	for i := range dagIns {
		if err = p.parseCmd(dagIns[i]); err != nil {
			return err
		}
	}
	return nil
}

func (p *DefParser) goWorker(queue <-chan *entity.TaskInstance) {
	for taskIns := range queue {
		if err := p.workerDo(taskIns); err != nil {
			p.handleErr(fmt.Errorf("worker do failed: %w", err))
		}
	}
	p.workerWg.Done()
}

func (p *DefParser) initialRunningDagIns() error {
	dagIns, err := GetStore().ListDagInstance(&ListDagInstanceInput{
		Worker: GetKeeper().WorkerKey(),
		Status: []entity.DagInstanceStatus{
			entity.DagInstanceStatusRunning,
		},
	})
	if err != nil {
		return err
	}

	for _, d := range dagIns {
		p.InitialDagIns(d)
	}
	return nil
}

// InitialDagIns
func (p *DefParser) InitialDagIns(dagIns *entity.DagInstance) {
	tasks, err := GetStore().ListTaskInstance(&ListTaskInstanceInput{
		DagInsID: dagIns.ID,
	})
	if err != nil {
		log.Errorf("dag instance[%s] list task instance failed: %s", dagIns.ID, err)
		return
	}

	if len(tasks) == 0 {
		return
	}

	root, err := BuildRootNode(MapTaskInsToGetter(tasks))
	if err != nil {
		log.Errorf("dag instance[%s] build task tree failed: %s", dagIns.ID, err)
		return
	}

	tree := &TaskTree{
		DagIns: dagIns,
		Root:   root,
	}
	executableTaskIds := tree.Root.GetExecutableTaskIds()
	if len(executableTaskIds) == 0 {
		sts, taskInsId := tree.Root.ComputeStatus()
		switch sts {
		case TreeStatusSuccess:
			tree.DagIns.Success()
		case TreeStatusBlocked:
			tree.DagIns.Block(fmt.Sprintf("initial blocked because task ins[%s]", taskInsId))
		case TreeStatusFailed:
			tree.DagIns.Fail(fmt.Sprintf("initial failed because task ins[%s]", taskInsId))
		case TreeStatusCanceled:
			tree.DagIns.Cancel(fmt.Sprintf("initial canceled because task ins[%s]", taskInsId))
		default:
			log.Warn("initial a dag which has no executable tasks",
				utils.LogKeyDagInsID, dagIns.ID)
			return
		}

		if err := GetStore().PatchDagIns(&entity.DagInstance{
			BaseInfo: entity.BaseInfo{ID: dagIns.ID},
			Status:   dagIns.Status}); err != nil {
			log.Errorf("patch dag instance[%s] failed: %s", dagIns.ID, err)
			return
		}
		return
	}

	p.taskTrees.Store(dagIns.ID, tree)
	taskMap := getTasksMap(tasks)
	for _, tid := range executableTaskIds {
		GetExecutor().Push(dagIns, taskMap[tid])
	}
}

func getTasksMap(tasks []*entity.TaskInstance) map[string]*entity.TaskInstance {
	tmpMap := map[string]*entity.TaskInstance{}
	for i := range tasks {
		tmpMap[tasks[i].ID] = tasks[i]
	}
	return tmpMap
}

func (p *DefParser) executeNext(taskIns *entity.TaskInstance) error {
	tree, ok := p.getTaskTree(taskIns.DagInsID)
	if !ok {
		return fmt.Errorf("dag instance[%s] does not found task tree", taskIns.DagInsID)
	}
	ids, find := tree.Root.GetNextTaskIds(taskIns)
	if !find {
		return fmt.Errorf("task instance[%s] does not found normal node", taskIns.ID)
	}
	// only the tasks which is not success has no next task ids
	if len(ids) == 0 {
		treeStatus, taskId := tree.Root.ComputeStatus()
		switch treeStatus {
		case TreeStatusRunning:
			return nil
		case TreeStatusFailed:
			tree.DagIns.Fail(fmt.Sprintf("task[%s] failed", taskId))
		case TreeStatusCanceled:
			tree.DagIns.Cancel(fmt.Sprintf("task[%s] canceled", taskId))
		case TreeStatusBlocked:
			tree.DagIns.Block(fmt.Sprintf("task[%s] blocked", taskId))
		case TreeStatusSuccess:
			tree.DagIns.Success()
		}

		// tree has already completed, delete from map
		p.taskTrees.Delete(taskIns.DagInsID)
		if err := GetStore().PatchDagIns(&entity.DagInstance{
			BaseInfo: entity.BaseInfo{ID: tree.DagIns.ID},
			Status:   tree.DagIns.Status,
			Reason:   tree.DagIns.Reason,
		}); err != nil {
			return err
		}

		return nil
	}
	if taskIns.Reason == ReasonSuccessAfterCanceled {
		return p.cancelChildTasks(tree, ids)
	}

	return p.pushTasks(tree.DagIns, ids)
}

func (p *DefParser) pushTasks(dagIns *entity.DagInstance, ids []string) error {
	tasks, err := GetStore().ListTaskInstance(&ListTaskInstanceInput{
		IDs: ids,
	})
	if err != nil {
		return err
	}
	for _, t := range tasks {
		GetExecutor().Push(dagIns, t)
	}

	return nil
}

func (p *DefParser) cancelChildTasks(tree *TaskTree, ids []string) error {
	walkNode(tree.Root, func(node *TaskNode) bool {
		if utils.StringsContain(ids, node.TaskInsID) {
			node.Status = entity.TaskInstanceStatusCanceled
		}
		return true
	}, false)

	for _, id := range ids {
		if err := GetStore().PatchTaskIns(&entity.TaskInstance{
			BaseInfo: entity.BaseInfo{ID: id},
			Status:   entity.TaskInstanceStatusCanceled,
			Reason:   ReasonParentCancel,
		}); err != nil {
			return err
		}
	}

	// not equal running mean that all tasks already completed
	if sts, _ := tree.Root.ComputeStatus(); sts != TreeStatusRunning {
		p.taskTrees.Delete(tree.DagIns.ID)
	}

	if !tree.DagIns.CanModifyStatus() {
		return nil
	}
	tree.DagIns.Cancel(fmt.Sprintf("task instance[%s] canceled", strings.Join(ids, ",")))
	return GetStore().PatchDagIns(tree.DagIns)
}

func (p *DefParser) getTaskTree(dagInsId string) (*TaskTree, bool) {
	tasks, ok := p.taskTrees.Load(dagInsId)
	if !ok {
		return nil, false
	}

	return tasks.(*TaskTree), true
}

// EntryTaskIns
func (p *DefParser) EntryTaskIns(taskIns *entity.TaskInstance) {
	murmurHash := murmur3.New32()
	// murmur3 hash does not return error, so we don't need to handle it
	_, _ = murmurHash.Write([]byte(taskIns.DagInsID))
	mod := int(murmurHash.Sum32()) % p.workerNumber
	p.sendToChannel(mod, taskIns, true)
}

func (p *DefParser) sendToChannel(mod int, taskIns *entity.TaskInstance, newRoutineWhenFull bool) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	// try to exit the sender goroutine as early as possible.
	// try-receive and try-send select blocks are specially optimized by the standard Go compiler,
	// so they are very efficient.
	select {
	case <-p.closeCh:
		log.Info("parser has already closed, so will not execute next task instances")
		return
	default:
	}

	if !newRoutineWhenFull {
		p.workerQueue[mod] <- taskIns
		return
	}

	select {
	// ensure that same dag instance handled by same worker, so avoid parallel writing
	case p.workerQueue[mod] <- taskIns:
	// if queue is full, we can do it in a new goroutine to prevent deadlock
	default:
		go p.sendToChannel(mod, taskIns, false)
	}
}

func (p *DefParser) workerDo(taskIns *entity.TaskInstance) error {
	return p.executeNext(taskIns)
}

func (p *DefParser) parseScheduleDagIns(dagIns *entity.DagInstance) error {
	if dagIns.Status == entity.DagInstanceStatusScheduled {
		dag, err := GetStore().GetDag(dagIns.DagID)
		if err != nil {
			return err
		}
		tasks, err := GetStore().ListTaskInstance(&ListTaskInstanceInput{
			DagInsID: dagIns.ID,
		})
		if err != nil {
			return err
		}

		// the init of tasks is not complete, should continue/start it.
		if len(dag.Tasks) != len(tasks) {
			var needInitTaskIns []*entity.TaskInstance
			for i := range dag.Tasks {
				notFound := true
				for j := range tasks {
					if dag.Tasks[i].ID == tasks[j].TaskID {
						notFound = false
					}
				}

				if notFound {
					renderParams, err := dagIns.Vars.Render(dag.Tasks[i].Params)
					if err != nil {
						return err
					}
					dag.Tasks[i].Params = renderParams
					if dag.Tasks[i].TimeoutSecs == 0 {
						dag.Tasks[i].TimeoutSecs = int(p.taskTimeout.Seconds())
					}
					needInitTaskIns = append(needInitTaskIns, entity.NewTaskInstance(dagIns.ID, dag.Tasks[i]))
				}
			}
			if err := GetStore().BatchCreatTaskIns(needInitTaskIns); err != nil {
				return err
			}
		}

		dagIns.Run()
		if err := GetStore().PatchDagIns(&entity.DagInstance{
			BaseInfo: dagIns.BaseInfo,
			Status:   dagIns.Status,
			Reason:   dagIns.Reason,
		}, "Reason"); err != nil {
			return err
		}
	}
	return nil
}

func (p *DefParser) parseCmd(dagIns *entity.DagInstance) (err error) {
	if dagIns.Cmd != nil {
		switch dagIns.Cmd.Name {
		case entity.CommandNameRetry:
			err = p.loopTaskThenInitialDagIns(
				dagIns,
				[]entity.TaskInstanceStatus{entity.TaskInstanceStatusFailed, entity.TaskInstanceStatusCanceled},
				func(t *entity.TaskInstance) bool {
					if t.Status != entity.TaskInstanceStatusFailed &&
						t.Status != entity.TaskInstanceStatusCanceled {
						return false
					}

					t.Status = entity.TaskInstanceStatusRetrying
					t.Reason = ""
					return true
				})
			if err != nil {
				return
			}
		case entity.CommandNameCancel:
			if err := GetExecutor().CancelTaskIns(dagIns.Cmd.TargetTaskInsIDs); err != nil {
				return err
			}
		case entity.CommandNameContinue:
			err = p.loopTaskThenInitialDagIns(
				dagIns,
				[]entity.TaskInstanceStatus{entity.TaskInstanceStatusBlocked},
				func(t *entity.TaskInstance) bool {
					if t.Status != entity.TaskInstanceStatusBlocked {
						return false
					}

					t.Status = entity.TaskInstanceStatusContinue
					t.Reason = ""
					return true
				})
			if err != nil {
				return
			}
		default:
			log.Errorf("command[%s] is invalid, ignore it", dagIns.Cmd.Name)
		}

		dagIns.Cmd = nil
		if err := GetStore().PatchDagIns(&entity.DagInstance{
			BaseInfo: dagIns.BaseInfo,
			Status:   dagIns.Status,
			Cmd:      dagIns.Cmd,
			Reason:   dagIns.Reason,
		}, "Cmd", "Reason"); err != nil {
			return err
		}
	}
	return nil
}

func (p *DefParser) loopTaskThenInitialDagIns(
	dagIns *entity.DagInstance,
	status []entity.TaskInstanceStatus,
	loopFunc func(*entity.TaskInstance) bool) (err error) {

	hasAnyTaskChanged := false
	defer func() {
		if err == nil && hasAnyTaskChanged {
			p.InitialDagIns(dagIns)
		}
	}()

	taskIns, err := GetStore().ListTaskInstance(&ListTaskInstanceInput{
		DagInsID: dagIns.ID,
		IDs:      dagIns.Cmd.TargetTaskInsIDs,
		Status:   status,
	})
	if err != nil {
		return err
	}

	for _, t := range taskIns {
		if taskChanged := loopFunc(t); !taskChanged {
			continue
		}

		if err := GetStore().UpdateTaskIns(t); err != nil {
			return err
		}
		hasAnyTaskChanged = true
	}
	dagIns.Run()
	return
}

// Close
func (p *DefParser) Close() {
	p.lock.Lock()
	defer p.lock.Unlock()

	select {
	case <-p.closeCh:
		log.Info("parser has already closed")
		return
	default:
	}
	close(p.closeCh)
	for i := range p.workerQueue {
		close(p.workerQueue[i])
	}
	p.workerWg.Wait()
}

func (p *DefParser) handleErr(err error) {
	log.Error("parser get some error",
		"module", "parser",
		"err", err)
}
