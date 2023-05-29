package mod

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/shiningrush/fastflow/pkg/entity"
)

// DefCommander used to execute command
type DefCommander struct {
}

// RunDag
func (c *DefCommander) RunDag(dagId string, specVars map[string]string) (*entity.DagInstance, error) {
	dag, err := GetStore().GetDag(dagId)
	if err != nil {
		return nil, err
	}

	dagIns, err := dag.Run(entity.TriggerManually, specVars)
	if err != nil {
		return nil, err
	}

	if err := GetStore().CreateDagIns(dagIns); err != nil {
		return nil, err
	}
	return dagIns, nil
}

// RunDagWithTags
func (c *DefCommander) RunDagWithTags(dagId string, specVars map[string]string, tags map[string]string) (*entity.DagInstance, error) {
	dag, err := GetStore().GetDag(dagId)
	if err != nil {
		return nil, err
	}

	dagIns, err := dag.Run(entity.TriggerManually, specVars)
	if err != nil {
		return nil, err
	}

	dagIns.Tags = entity.NewDagInstanceTags(tags)
	if err := GetStore().CreateDagIns(dagIns); err != nil {
		return nil, err
	}
	return dagIns, nil
}

// RetryDagIns
func (c *DefCommander) RetryDagIns(dagInsId string, ops ...CommandOptSetter) error {
	taskIns, err := GetStore().ListTaskInstance(&ListTaskInstanceInput{
		DagInsID: dagInsId,
		Status:   []entity.TaskInstanceStatus{entity.TaskInstanceStatusFailed, entity.TaskInstanceStatusCanceled},
	})
	if err != nil {
		return err
	}

	if len(taskIns) == 0 {
		return fmt.Errorf("no failed and canceled task instance")
	}

	var taskIds []string
	for _, t := range taskIns {
		taskIds = append(taskIds, t.ID)
	}

	return c.RetryTask(taskIds, ops...)
}

// CancelDagIns
func (c *DefCommander) CancelDagIns(dagInsId string, ops ...CommandOptSetter) error {
	taskIns, err := GetStore().ListTaskInstance(&ListTaskInstanceInput{
		DagInsID: dagInsId,
		Status: []entity.TaskInstanceStatus{
			entity.TaskInstanceStatusInit,
			entity.TaskInstanceStatusRunning,
			entity.TaskInstanceStatusEnding,
			entity.TaskInstanceStatusRetrying,
		},
	})
	if err != nil {
		return err
	}

	if len(taskIns) == 0 {
		return fmt.Errorf("no task instance")
	}

	var taskIds []string
	for _, t := range taskIns {
		taskIds = append(taskIds, t.ID)
	}

	return c.CancelTask(taskIds, ops...)
}

// RetryTask
func (c *DefCommander) RetryTask(taskInsIds []string, ops ...CommandOptSetter) error {
	opt := initOption(ops)
	return executeCommand(taskInsIds, func(dagIns *entity.DagInstance, isWorkerAlive bool) error {
		if !isWorkerAlive {
			aliveNodes, err := GetKeeper().AliveNodes()
			if err != nil {
				return err
			}
			dagIns.Worker = aliveNodes[rand.Intn(len(aliveNodes))]
		}
		return dagIns.Retry(taskInsIds)
	}, opt)
}

// CancelTask
func (c *DefCommander) CancelTask(taskInsIds []string, ops ...CommandOptSetter) error {
	opt := initOption(ops)
	return executeCommand(taskInsIds, func(dagIns *entity.DagInstance, isWorkerAlive bool) error {
		if !isWorkerAlive {
			return fmt.Errorf("worker is not healthy, you can not cancel it")
		}
		return dagIns.Cancel(taskInsIds)
	}, opt)
}

func initOption(opSetter []CommandOptSetter) (opt CommandOption) {
	opt.syncTimeout = 5 * time.Second
	opt.syncInterval = 500 * time.Millisecond
	for _, op := range opSetter {
		op(&opt)
	}
	return
}

func executeCommand(
	taskInsIds []string,
	perform func(dagIns *entity.DagInstance, isWorkerAlive bool) error,
	opt CommandOption) error {
	if len(taskInsIds) == 0 {
		return errors.New("here is no any task by give task's ids")
	}

	taskIns, err := GetStore().ListTaskInstance(&ListTaskInstanceInput{
		IDs: taskInsIds,
	})
	if err != nil {
		return err
	}

	if len(taskInsIds) != len(taskIns) {
		var notFoundIds []string
		for _, id := range taskInsIds {
			find := false
			for _, ins := range taskIns {
				if ins.ID == id {
					find = true
					break
				}
			}
			if !find {
				notFoundIds = append(notFoundIds, id)
			}
		}
		return fmt.Errorf("id[%s] does not found task instance", strings.Join(notFoundIds, ", "))
	}

	dagInsId := taskIns[0].DagInsID
	for _, t := range taskIns {
		if t.DagInsID != dagInsId {
			return fmt.Errorf("task instance[%s] is from different dag instance", t.ID)
		}
	}

	dagIns, err := GetStore().GetDagInstance(dagInsId)
	if err != nil {
		return err
	}

	isWorkerAlive, err := GetKeeper().IsAlive(dagIns.Worker)
	if err != nil {
		return err
	}

	if err := perform(dagIns, isWorkerAlive); err != nil {
		return err
	}
	if err := GetStore().PatchDagIns(&entity.DagInstance{
		BaseInfo: dagIns.BaseInfo,
		Worker:   dagIns.Worker,
		Cmd:      dagIns.Cmd,
	}); err != nil {
		return err
	}

	if opt.isSync {
		return ensureCmdExecuted(dagInsId, opt)
	}

	return nil
}

func ensureCmdExecuted(dagInsId string, opt CommandOption) error {
	timer := time.NewTimer(opt.syncTimeout)
	ticker := time.NewTicker(opt.syncInterval)
	for {
		select {
		case <-ticker.C:
			dag, err := GetStore().GetDagInstance(dagInsId)
			if err != nil {
				return err
			}
			if dag.Cmd == nil {
				return nil
			}
		case <-timer.C:
			return fmt.Errorf("watch command executing timeout")
		}
	}
}
