//go:build integration
// +build integration

package mongo

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var mongoConn = "mongodb://root:pwd@127.0.0.1:27017/fastflow?authSource=admin"

func TestKeeper_Sanity(t *testing.T) {
	w1, w2, w3 := initSanityWorker(t)
	// sleep 2 unhealthy period then check
	time.Sleep(time.Second * 10)

	assert.Equal(t, "worker-3", w3.WorkerKey())
	assert.Equal(t, true, w1.IsLeader())
	nodes, err := w2.AliveNodes()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"worker-1", "worker-2", "worker-3"}, nodes)
	log.Println("keeper work well, ready to re-goElect")

	w1.Close()
	time.Sleep(6 * time.Second)
	// should elect new leader
	assert.True(t, w2.IsLeader() || w3.IsLeader())
	nodes, err = w2.AliveNodes()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"worker-2", "worker-3"}, nodes)
	w2.Close()
	w3.Close()
}

func TestKeeper_Crash(t *testing.T) {
	w1, w2, w3 := initSanityWorker(t)
	assert.Equal(t, "worker-1", w1.WorkerKey())
	assert.Equal(t, true, w1.IsLeader())
	log.Println("keeper work well, ready to re-goElect")

	w1.forceClose()
	time.Sleep(6 * time.Second)
	// should goElect new leader
	assert.True(t, w2.IsLeader() || w3.IsLeader())
	nodes, err := w3.AliveNodes()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"worker-2", "worker-3"}, nodes)
	w1.closeCh = make(chan struct{})
	w1.Close()
	w2.Close()
	w3.Close()
}

func TestKeeper_Concurrency(t *testing.T) {
	wg := sync.WaitGroup{}
	stsCh := make(chan struct {
		isLeader   bool
		aliveNodes int
	})
	leaderCount := 0

	go func() {
		for ret := range stsCh {
			if ret.isLeader {
				leaderCount++
			}
		}
	}()

	curCnt := 20
	initCompleted := sync.WaitGroup{}
	initCompleted.Add(curCnt)
	closeCh := make(chan struct{})
	for i := 0; i < curCnt; i++ {
		wg.Add(1)
		go func(i int, closeCh chan struct{}) {
			w := initWorker(t, fmt.Sprintf("worker-%d", i))
			ns, err := w.AliveNodes()
			assert.NoError(t, err)
			stsCh <- struct {
				isLeader   bool
				aliveNodes int
			}{isLeader: w.IsLeader(), aliveNodes: len(ns)}
			initCompleted.Done()
			<-closeCh
			w.Close()
			wg.Done()
		}(i, closeCh)
	}
	initCompleted.Wait()
	w := initWorker(t, "latest-0")
	nodes, err := w.AliveNodes()
	assert.NoError(t, err)
	assert.Equal(t, curCnt+1, len(nodes))
	assert.Equal(t, false, w.IsLeader())
	assert.Equal(t, 1, leaderCount, "leader should always be one")
	w.Close()
	close(closeCh)
	wg.Wait()
}

func TestKeeper_Reconnect(t *testing.T) {
	w1 := initWorker(t, "worker-1")
	assert.True(t, w1.IsLeader())
	w1.forceClose()

	w1 = initWorker(t, "worker-1")
	assert.True(t, w1.IsLeader())
	w1.Close()
}

func initWorker(t *testing.T, key string) *Keeper {
	w := NewKeeper(&KeeperOption{
		Key:                      key,
		ConnStr:                  mongoConn,
		InitFlakeGeneratorSwitch: boolToPointer(true),
	})
	err := w.Init()
	require.NoError(t, err)
	return w
}

func boolToPointer(b bool) *bool {
	return &b
}

func initSanityWorker(t *testing.T) (w1, w2, w3 *Keeper) {
	w1 = initWorker(t, "worker-1")
	w2 = initWorker(t, "worker-2")
	w3 = initWorker(t, "worker-3")
	return
}
