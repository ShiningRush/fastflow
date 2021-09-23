package mod

import (
	"fmt"
	"github.com/shiningrush/fastflow/pkg/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestDefWatchDog_HandleExpiredTaskIns(t *testing.T) {
	tests := []struct {
		caseDesc            string
		giveWd              *DefWatchDog
		giveListTasks       []*entity.TaskInstance
		giveListTasksErr    error
		giveDagPatchErr     error
		giveTaskPatchErr    error
		wantErr             error
		wantListInput       *ListTaskInstanceInput
		wantPatchDagCalled  bool
		wantPatchDag        map[int]*entity.DagInstance
		wantPatchTaskCalled bool
		wantPatchTask       map[int]*entity.TaskInstance
	}{
		{
			caseDesc: "normal",
			giveWd: &DefWatchDog{
				closeCh: make(chan struct{}),
			},
			giveListTasks: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "1"}, DagInsID: "dag-1", Status: entity.TaskInstanceStatusInit},
				{BaseInfo: entity.BaseInfo{ID: "2"}, DagInsID: "dag-2", Status: entity.TaskInstanceStatusRunning},
			},
			wantListInput: &ListTaskInstanceInput{
				Status:  []entity.TaskInstanceStatus{entity.TaskInstanceStatusRunning},
				Expired: true,
			},
			wantPatchDag: map[int]*entity.DagInstance{
				0: {
					BaseInfo: entity.BaseInfo{ID: "dag-1"},
					Status:   entity.DagInstanceStatusFailed,
				},
				1: {
					BaseInfo: entity.BaseInfo{ID: "dag-2"},
					Status:   entity.DagInstanceStatusFailed,
				},
			},
			wantPatchDagCalled: true,
			wantPatchTask: map[int]*entity.TaskInstance{
				0: {
					BaseInfo: entity.BaseInfo{ID: "1"},
					Status:   entity.TaskInstanceStatusFailed,
					Reason:   DefFailedReason,
				},
				1: {
					BaseInfo: entity.BaseInfo{ID: "2"},
					Status:   entity.TaskInstanceStatusFailed,
					Reason:   DefFailedReason,
				},
			},
			wantPatchTaskCalled: true,
		},
		{
			caseDesc: "list failed",
			giveWd: &DefWatchDog{
				closeCh: make(chan struct{}),
			},
			giveListTasksErr: fmt.Errorf("list failed"),
			wantErr:          fmt.Errorf("list failed"),
			wantListInput: &ListTaskInstanceInput{
				Status:  []entity.TaskInstanceStatus{entity.TaskInstanceStatusRunning},
				Expired: true,
			},
		},
		{
			caseDesc: "patch dag failed",
			giveWd: &DefWatchDog{
				closeCh: make(chan struct{}),
			},
			giveListTasks: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "1"}, DagInsID: "dag-1", Status: entity.TaskInstanceStatusInit},
				{BaseInfo: entity.BaseInfo{ID: "2"}, DagInsID: "dag-2", Status: entity.TaskInstanceStatusRunning},
			},
			wantListInput: &ListTaskInstanceInput{
				Status:  []entity.TaskInstanceStatus{entity.TaskInstanceStatusRunning},
				Expired: true,
			},
			wantPatchDag: map[int]*entity.DagInstance{
				0: {
					BaseInfo: entity.BaseInfo{ID: "dag-1"},
					Status:   entity.DagInstanceStatusFailed,
				},
			},
			wantPatchDagCalled: true,
			giveDagPatchErr:    fmt.Errorf("patch failed"),
			wantErr:            fmt.Errorf("patch expired dag instance[dag-1] failed: patch failed"),
		},
		{
			caseDesc: "patch task failed",
			giveWd: &DefWatchDog{
				closeCh: make(chan struct{}),
			},
			giveListTasks: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "1"}, DagInsID: "dag-1", Status: entity.TaskInstanceStatusInit},
				{BaseInfo: entity.BaseInfo{ID: "2"}, DagInsID: "dag-2", Status: entity.TaskInstanceStatusRunning},
			},
			wantListInput: &ListTaskInstanceInput{
				Status:  []entity.TaskInstanceStatus{entity.TaskInstanceStatusRunning},
				Expired: true,
			},
			wantPatchDag: map[int]*entity.DagInstance{
				0: {
					BaseInfo: entity.BaseInfo{ID: "dag-1"},
					Status:   entity.DagInstanceStatusFailed,
				},
			},
			wantPatchDagCalled: true,
			wantPatchTask: map[int]*entity.TaskInstance{
				0: {
					BaseInfo: entity.BaseInfo{ID: "1"},
					Status:   entity.TaskInstanceStatusFailed,
					Reason:   DefFailedReason,
				},
			},
			wantPatchTaskCalled: true,
			giveTaskPatchErr:    fmt.Errorf("patch failed"),
			wantErr:             fmt.Errorf("patch expired task[1] failed: patch failed"),
		},
		{
			caseDesc: "no record",
			giveWd: &DefWatchDog{
				closeCh: make(chan struct{}),
			},
			giveListTasks: []*entity.TaskInstance{},
			wantListInput: &ListTaskInstanceInput{
				Status:  []entity.TaskInstanceStatus{entity.TaskInstanceStatusRunning},
				Expired: true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			calledList, calledDagPatch, calledTaskPatch := false, false, false
			mStore := &MockStore{}
			patchDagCnt, patchTaskCnt := 0, 0
			mStore.On("ListTaskInstance", mock.Anything).Run(func(args mock.Arguments) {
				calledList = true
				assert.Equal(t, tc.wantListInput, args.Get(0))
			}).Return(tc.giveListTasks, tc.giveListTasksErr)
			mStore.On("PatchDagIns", mock.Anything).Run(func(args mock.Arguments) {
				calledDagPatch = true
				assert.Equal(t, tc.wantPatchDag[patchDagCnt], args.Get(0))
				patchDagCnt++
			}).Return(tc.giveDagPatchErr)

			mStore.On("PatchTaskIns", mock.Anything).Run(func(args mock.Arguments) {
				calledTaskPatch = true
				assert.Equal(t, tc.wantPatchTask[patchTaskCnt], args.Get(0))
				patchTaskCnt++
			}).Return(tc.giveTaskPatchErr)
			SetStore(mStore)

			err := tc.giveWd.handleExpiredTaskIns()
			assert.Equal(t, tc.wantErr, err)
			assert.True(t, calledList)
			assert.Equal(t, tc.wantPatchTaskCalled, calledTaskPatch)
			assert.Equal(t, tc.wantPatchDagCalled, calledDagPatch)
		})
	}
}

func TestDefWatchDog_HandleLeftBehindDagIns(t *testing.T) {
	tests := []struct {
		caseDesc           string
		giveWd             *DefWatchDog
		giveListRet        []*entity.DagInstance
		giveListRetErr     error
		giveBatchUpdateErr error
		wantErr            error
		wantListInput      *ListDagInstanceInput
		wantBatchInput     []*entity.DagInstance
		wantBatchCalled    bool
	}{
		{
			caseDesc: "sanity",
			giveWd: &DefWatchDog{
				dagScheduledTimeout: time.Minute,
				closeCh:             make(chan struct{}),
			},
			giveListRet: []*entity.DagInstance{
				{BaseInfo: entity.BaseInfo{ID: "1"}, Status: entity.DagInstanceStatusInit},
				{BaseInfo: entity.BaseInfo{ID: "2"}, Status: entity.DagInstanceStatusRunning},
			},
			wantListInput: &ListDagInstanceInput{
				Status:     []entity.DagInstanceStatus{entity.DagInstanceStatusScheduled},
				UpdatedEnd: time.Now().Add(-1 * time.Minute).Unix(),
			},
			wantBatchInput: []*entity.DagInstance{
				{BaseInfo: entity.BaseInfo{ID: "1"}, Status: entity.DagInstanceStatusInit},
				{BaseInfo: entity.BaseInfo{ID: "2"}, Status: entity.DagInstanceStatusInit},
			},
			wantBatchCalled: true,
		},
		{
			caseDesc: "list failed",
			giveWd: &DefWatchDog{
				dagScheduledTimeout: time.Minute,
				closeCh:             make(chan struct{}),
			},
			giveListRetErr: fmt.Errorf("list failed"),
			wantErr:        fmt.Errorf("list failed"),
			wantListInput: &ListDagInstanceInput{
				Status:     []entity.DagInstanceStatus{entity.DagInstanceStatusScheduled},
				UpdatedEnd: time.Now().Add(-1 * time.Minute).Unix(),
			},
		},
		{
			caseDesc: "update failed",
			giveWd: &DefWatchDog{
				dagScheduledTimeout: time.Minute,
				closeCh:             make(chan struct{}),
			},
			giveListRet: []*entity.DagInstance{
				{BaseInfo: entity.BaseInfo{ID: "1"}, Status: entity.DagInstanceStatusInit},
			},
			wantListInput: &ListDagInstanceInput{
				Status:     []entity.DagInstanceStatus{entity.DagInstanceStatusScheduled},
				UpdatedEnd: time.Now().Add(-1 * time.Minute).Unix(),
			},
			wantBatchInput: []*entity.DagInstance{
				{BaseInfo: entity.BaseInfo{ID: "1"}, Status: entity.DagInstanceStatusInit},
			},
			giveBatchUpdateErr: fmt.Errorf("batch update failed"),
			wantErr:            fmt.Errorf("batch update failed"),
			wantBatchCalled:    true,
		},
		{
			caseDesc: "no record",
			giveWd: &DefWatchDog{
				dagScheduledTimeout: time.Minute,
				closeCh:             make(chan struct{}),
			},
			giveListRet: []*entity.DagInstance{},
			wantListInput: &ListDagInstanceInput{
				Status:     []entity.DagInstanceStatus{entity.DagInstanceStatusScheduled},
				UpdatedEnd: time.Now().Add(-1 * time.Minute).Unix(),
			},
			wantBatchCalled: false,
		},
	}

	for _, tc := range tests {
		calledList, calledBatch := false, false
		mStore := &MockStore{}
		mStore.On("ListDagInstance", mock.Anything).Run(func(args mock.Arguments) {
			calledList = true
			assert.Equal(t, tc.wantListInput, args.Get(0), tc.caseDesc)
		}).Return(tc.giveListRet, tc.giveListRetErr)

		mStore.On("BatchUpdateDagIns", mock.Anything).Run(func(args mock.Arguments) {
			calledBatch = true
			assert.Equal(t, tc.wantBatchInput, args.Get(0), tc.caseDesc)
		}).Return(tc.giveBatchUpdateErr)
		SetStore(mStore)

		err := tc.giveWd.handleLeftBehindDagIns()
		assert.Equal(t, tc.wantErr, err, tc.caseDesc)
		assert.True(t, calledList, tc.caseDesc)
		assert.Equal(t, tc.wantBatchCalled, calledBatch)
	}
}

func TestDefWatchDog(t *testing.T) {
	calledListDag, calledListTask := false, false
	mStore := &MockStore{}
	mStore.On("ListDagInstance", mock.Anything).Run(func(args mock.Arguments) {
		calledListDag = true
	}).Return(nil, nil)
	mStore.On("ListTaskInstance", mock.Anything).Run(func(args mock.Arguments) {
		calledListTask = true
	}).Return(nil, nil)
	SetStore(mStore)

	wDog := NewDefWatchDog(time.Minute)
	wDog.Init()
	time.Sleep(1 * time.Second)
	wDog.Close()
	assert.True(t, calledListDag, calledListTask)
}
