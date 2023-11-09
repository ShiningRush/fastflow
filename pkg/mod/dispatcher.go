package mod

import (
	"sync"
	"time"

	"github.com/shiningrush/fastflow/pkg/entity"
	"github.com/shiningrush/fastflow/pkg/event"
	"github.com/shiningrush/fastflow/pkg/log"
	"github.com/shiningrush/fastflow/pkg/utils/data"
	"github.com/shiningrush/goevent"
)

// DefDispatcher
type DefDispatcher struct {
	closeCh chan struct{}

	wg sync.WaitGroup
}

// NewDefDispatcher
func NewDefDispatcher() *DefDispatcher {
	return &DefDispatcher{
		closeCh: make(chan struct{}),
	}
}

// Init
func (d *DefDispatcher) Init() {
	d.wg.Add(1)
	go d.WatchInitDags()
}

// WatchInitDags
func (d *DefDispatcher) WatchInitDags() {
	closed := false
	timerCh := time.Tick(time.Second)
	for !closed {
		select {
		case <-d.closeCh:
			closed = true
		case <-timerCh:
			start := time.Now()
			e := &event.DispatchInitDagInsCompleted{}
			if err := d.Do(); err != nil {
				d.handlerErr(err)
				e.Error = err
			}
			e.ElapsedMs = time.Now().Sub(start).Milliseconds()
			goevent.Publish(e)
		}
	}
	d.wg.Done()
}

// Do dispatch
func (d *DefDispatcher) Do() error {
	dagIns, err := GetStore().ListDagInstance(&ListDagInstanceInput{
		Status: []entity.DagInstanceStatus{
			entity.DagInstanceStatusInit,
		},
		Limit: 1000,
	})
	if err != nil {
		return err
	}
	if len(dagIns) == 0 {
		return nil
	}

	nodes, err := GetKeeper().AliveNodes()
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		return data.ErrNoAliveNodes
	}

	for i := range dagIns {
		dagIns[i].Status = entity.DagInstanceStatusScheduled
		dagIns[i].Worker = nodes[i%len(nodes)]
	}

	if err := GetStore().BatchUpdateDagIns(dagIns); err != nil {
		return err
	}
	return nil
}

func (d *DefDispatcher) handlerErr(err error) {
	log.Errorf("dispatch failed",
		"module", "dispatch",
		"err", err)
}

// Close component
func (d *DefDispatcher) Close() {
	close(d.closeCh)
	d.wg.Wait()
}
