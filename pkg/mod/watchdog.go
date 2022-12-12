package mod

import (
	"fmt"
	"github.com/realeyeeos/fastflow/pkg/entity"
	"github.com/realeyeeos/fastflow/pkg/log"
	"sync"
	"time"
)

const DefFailedReason = "force failed by watch dog because it execute too long"

// DefWatchDog
type DefWatchDog struct {
	dagScheduledTimeout time.Duration

	wg      sync.WaitGroup
	closeCh chan struct{}
}

// NewDefWatchDog
func NewDefWatchDog(dagScheduledTimeout time.Duration) *DefWatchDog {
	return &DefWatchDog{
		dagScheduledTimeout: dagScheduledTimeout,
		closeCh:             make(chan struct{}),
	}
}

// Init
func (wd *DefWatchDog) Init() {
	wd.wg.Add(1)
	go wd.watchWrapper(wd.handleExpiredTaskIns)
	wd.wg.Add(1)
	go wd.watchWrapper(wd.handleLeftBehindDagIns)
}

// Close
func (wd *DefWatchDog) Close() {
	close(wd.closeCh)
	wd.wg.Wait()
}

func (wd *DefWatchDog) watchWrapper(do func() error) {
	timerCh := time.Tick(time.Second)
	closed := false
	for !closed {
		select {
		case <-wd.closeCh:
			closed = true
		case <-timerCh:
			if err := do(); err != nil {
				wd.handleErr(err)
			}
		}
	}
	wd.wg.Done()
}

func (wd *DefWatchDog) handleExpiredTaskIns() error {
	taskIns, err := GetStore().ListTaskInstance(&ListTaskInstanceInput{
		Status:  []entity.TaskInstanceStatus{entity.TaskInstanceStatusRunning},
		Expired: true,
	})
	if err != nil {
		return err
	}
	if len(taskIns) == 0 {
		return nil
	}

	for i := range taskIns {
		if err := GetStore().PatchDagIns(&entity.DagInstance{
			BaseInfo: entity.BaseInfo{ID: taskIns[i].DagInsID},
			Status:   entity.DagInstanceStatusFailed,
		}); err != nil {
			return fmt.Errorf("patch expired dag instance[%s] failed: %s", taskIns[i].DagInsID, err)
		}

		if err := GetStore().PatchTaskIns(&entity.TaskInstance{
			BaseInfo: entity.BaseInfo{ID: taskIns[i].ID},
			Status:   entity.TaskInstanceStatusFailed,
			Reason:   DefFailedReason,
		}); err != nil {
			return fmt.Errorf("patch expired task[%s] failed: %s", taskIns[i].ID, err)
		}
	}
	return nil
}

func (wd *DefWatchDog) handleLeftBehindDagIns() error {
	dagIns, err := GetStore().ListDagInstance(&ListDagInstanceInput{
		Status:     []entity.DagInstanceStatus{entity.DagInstanceStatusScheduled},
		UpdatedEnd: time.Now().Add(-1 * wd.dagScheduledTimeout).Unix()},
	)
	if err != nil {
		return err
	}
	if len(dagIns) == 0 {
		return nil
	}

	for i := range dagIns {
		dagIns[i].Status = entity.DagInstanceStatusInit
	}
	if err := GetStore().BatchUpdateDagIns(dagIns); err != nil {
		return err
	}
	return nil
}

func (wd *DefWatchDog) handleErr(err error) {
	log.Error("here are some errors",
		"module", "watchdog",
		"err", err)
}
