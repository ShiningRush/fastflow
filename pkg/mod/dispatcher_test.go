package mod

import (
	"fmt"
	"github.com/realeyeeos/fastflow/pkg/entity"
	"github.com/realeyeeos/fastflow/pkg/log"
	"github.com/realeyeeos/fastflow/pkg/utils/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestDefDispatcher_Do(t *testing.T) {
	tests := []struct {
		caseDesc              string
		giveListRet           []*entity.DagInstance
		giveListErr           error
		giveAliveNodes        []string
		giveAliveErr          error
		giveBatchUpdateErr    error
		wantErr               error
		wantAliveNodeCalled   bool
		wantBatchUpdateCalled bool
		wantBatchUpdateInput  []*entity.DagInstance
	}{
		{
			caseDesc: "sanity",
			giveListRet: []*entity.DagInstance{
				{},
				{},
				{},
				{},
			},
			giveAliveNodes:      []string{"worker-1", "worker-2", "worker-3"},
			wantAliveNodeCalled: true,
			wantBatchUpdateInput: []*entity.DagInstance{
				{
					Status: entity.DagInstanceStatusScheduled,
					Worker: "worker-1",
				},
				{
					Status: entity.DagInstanceStatusScheduled,
					Worker: "worker-2",
				},
				{
					Status: entity.DagInstanceStatusScheduled,
					Worker: "worker-3",
				},
				{
					Status: entity.DagInstanceStatusScheduled,
					Worker: "worker-1",
				},
			},
			wantBatchUpdateCalled: true,
		},
		{
			caseDesc:    "list failed",
			giveListErr: fmt.Errorf("list failed"),
			wantErr:     fmt.Errorf("list failed"),
		},
		{
			caseDesc:    "no dag instance",
			giveListRet: []*entity.DagInstance{},
		},
		{
			caseDesc:            "get alive nodes failed",
			giveListRet:         []*entity.DagInstance{{}},
			giveAliveErr:        fmt.Errorf("get alive nodes failed"),
			wantErr:             fmt.Errorf("get alive nodes failed"),
			wantAliveNodeCalled: true,
		},
		{
			caseDesc:            "no alive node",
			giveListRet:         []*entity.DagInstance{{}},
			giveAliveNodes:      []string{},
			wantErr:             data.ErrNoAliveNodes,
			wantAliveNodeCalled: true,
		},
		{
			caseDesc:           "batch update failed",
			giveListRet:        []*entity.DagInstance{{}},
			giveAliveNodes:     []string{"node"},
			giveBatchUpdateErr: fmt.Errorf("batch update failed"),
			wantErr:            fmt.Errorf("batch update failed"),
			wantBatchUpdateInput: []*entity.DagInstance{
				{Status: entity.DagInstanceStatusScheduled, Worker: "node"},
			},
			wantAliveNodeCalled:   true,
			wantBatchUpdateCalled: true,
		},
	}

	for _, tc := range tests {
		calledList, calledAlive, calledBatch := false, false, false
		litInput := &ListDagInstanceInput{
			Status: []entity.DagInstanceStatus{entity.DagInstanceStatusInit},
			Limit:  1000,
		}
		d := NewDefDispatcher()
		mStore := &MockStore{}
		mStore.On("ListDagInstance", mock.Anything).Run(func(args mock.Arguments) {
			calledList = true
			assert.Equal(t, litInput, args.Get(0), tc.caseDesc)
		}).Return(tc.giveListRet, tc.giveListErr)
		mStore.On("BatchUpdateDagIns", mock.Anything).Run(func(args mock.Arguments) {
			calledBatch = true
			assert.Equal(t, tc.wantBatchUpdateInput, args.Get(0), tc.caseDesc)
		}).Return(tc.giveBatchUpdateErr)
		SetStore(mStore)

		mKeeper := &MockKeeper{}
		mKeeper.On("AliveNodes").Run(func(args mock.Arguments) {
			calledAlive = true
		}).Return(tc.giveAliveNodes, tc.giveAliveErr)
		SetKeeper(mKeeper)

		err := d.Do()
		assert.Equal(t, tc.wantErr, err, tc.caseDesc)
		assert.True(t, calledList, tc.caseDesc)
		assert.Equal(t, tc.wantAliveNodeCalled, calledAlive, tc.caseDesc)
		assert.Equal(t, tc.wantBatchUpdateCalled, calledBatch, tc.caseDesc)
	}
}

func TestDefDispatcher_InitAndClose(t *testing.T) {
	tests := []struct {
		caseDesc              string
		giveListRet           []*entity.DagInstance
		giveListErr           error
		giveAliveNodes        []string
		giveAliveErr          error
		giveBatchUpdateErr    error
		wantAliveNodeCalled   bool
		wantBatchUpdateCalled bool
		wantLogCalled         bool
		wantBatchUpdateInput  []*entity.DagInstance
	}{
		{
			caseDesc: "sanity",
			giveListRet: []*entity.DagInstance{
				{},
			},
			giveAliveNodes:      []string{"node"},
			wantAliveNodeCalled: true,
			wantBatchUpdateInput: []*entity.DagInstance{
				{
					Status: entity.DagInstanceStatusScheduled,
					Worker: "node",
				},
			},
			wantBatchUpdateCalled: true,
		},
		{
			caseDesc:      "list failed",
			giveListErr:   fmt.Errorf("list failed"),
			wantLogCalled: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			calledList, calledAlive, calledBatch, calledLog := false, false, false, false
			litInput := &ListDagInstanceInput{
				Status: []entity.DagInstanceStatus{entity.DagInstanceStatusInit},
				Limit:  1000,
			}
			d := NewDefDispatcher()
			mStore := &MockStore{}
			mStore.On("ListDagInstance", mock.Anything).Run(func(args mock.Arguments) {
				calledList = true
				assert.Equal(t, litInput, args.Get(0), tc.caseDesc)
			}).Return(tc.giveListRet, tc.giveListErr)
			mStore.On("BatchUpdateDagIns", mock.Anything).Run(func(args mock.Arguments) {
				calledBatch = true
				assert.Equal(t, tc.wantBatchUpdateInput, args.Get(0), tc.caseDesc)
			}).Return(tc.giveBatchUpdateErr)
			SetStore(mStore)

			mKeeper := &MockKeeper{}
			mKeeper.On("AliveNodes").Run(func(args mock.Arguments) {
				calledAlive = true
			}).Return(tc.giveAliveNodes, tc.giveAliveErr)
			SetKeeper(mKeeper)

			mLogger := &log.MockLogger{}
			mLogger.On("Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				calledLog = true
			})
			log.SetLogger(mLogger)
			d.Init()
			time.Sleep(time.Second)
			d.Close()
			assert.True(t, calledList, tc.caseDesc)
			assert.Equal(t, calledLog, tc.wantLogCalled, tc.caseDesc)
			assert.Equal(t, tc.wantAliveNodeCalled, calledAlive, tc.caseDesc)
			assert.Equal(t, tc.wantBatchUpdateCalled, calledBatch, tc.caseDesc)
		})
	}
	log.SetLogger(&log.StdoutLogger{})
}
