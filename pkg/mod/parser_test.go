package mod

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/shiningrush/fastflow/pkg/entity"
	"github.com/shiningrush/fastflow/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDefParser_cancelChildTask(t *testing.T) {
	tests := []struct {
		caseDesc            string
		giveParser          *DefParser
		giveDagIns          *entity.DagInstance
		giveTasks           []*entity.TaskInstance
		giveIds             []string
		givePatchTaskErr    error
		givePatchDagErr     error
		wantPatchTaskCalled bool
		wantPatchDagCalled  bool
		wantPatchTasks      []*entity.TaskInstance
		wantPatchDagIns     *entity.DagInstance
		wantErr             error
		wantRoot            *TaskNode
		wantDeleteTree      bool
	}{
		{
			caseDesc:   "sanity",
			giveParser: &DefParser{},
			giveTasks: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "task1"}, TaskID: "task1", Status: entity.TaskInstanceStatusSuccess, DependOn: []string{}},
				{BaseInfo: entity.BaseInfo{ID: "task2"}, TaskID: "task2", Status: entity.TaskInstanceStatusSuccess, DependOn: []string{"task1"}},
				{BaseInfo: entity.BaseInfo{ID: "task3"}, TaskID: "task3", Status: entity.TaskInstanceStatusRunning, DependOn: []string{"task1"}},
				{BaseInfo: entity.BaseInfo{ID: "task4"}, TaskID: "task4", Status: entity.TaskInstanceStatusInit, DependOn: []string{"task2"}},
				{BaseInfo: entity.BaseInfo{ID: "task5"}, TaskID: "task5", Status: entity.TaskInstanceStatusInit, DependOn: []string{"task2"}},
			},
			giveDagIns:          &entity.DagInstance{Status: entity.DagInstanceStatusRunning},
			giveIds:             []string{"task4", "task5"},
			wantPatchTaskCalled: true,
			wantPatchTasks: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "task4"}, Status: entity.TaskInstanceStatusCanceled, Reason: ReasonParentCancel},
				{BaseInfo: entity.BaseInfo{ID: "task5"}, Status: entity.TaskInstanceStatusCanceled, Reason: ReasonParentCancel},
			},
			wantPatchDagCalled: true,
			wantPatchDagIns:    &entity.DagInstance{Status: entity.DagInstanceStatusCanceled, Reason: "task instance[task4,task5] canceled"},
			wantRoot: &TaskNode{
				TaskInsID: virtualTaskRootID,
				Status:    entity.TaskInstanceStatusSuccess,
				children: []*TaskNode{
					{
						TaskInsID: "task1",
						Status:    entity.TaskInstanceStatusSuccess,
						children: []*TaskNode{
							{
								TaskInsID: "task2",
								Status:    entity.TaskInstanceStatusSuccess,
								children: []*TaskNode{
									{TaskInsID: "task4", Status: entity.TaskInstanceStatusCanceled},
									{TaskInsID: "task5", Status: entity.TaskInstanceStatusCanceled},
								},
							},
							{
								TaskInsID: "task3",
								Status:    entity.TaskInstanceStatusRunning,
							},
						},
					},
				},
			},
		},
		{
			caseDesc:   "patch failed",
			giveParser: &DefParser{},
			giveTasks: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "task1"}, TaskID: "task1", Status: entity.TaskInstanceStatusSuccess, DependOn: []string{}},
				{BaseInfo: entity.BaseInfo{ID: "task2"}, TaskID: "task2", Status: entity.TaskInstanceStatusInit, DependOn: []string{"task1"}},
			},
			giveDagIns:          &entity.DagInstance{Status: entity.DagInstanceStatusRunning},
			giveIds:             []string{"task2"},
			wantPatchTaskCalled: true,
			wantPatchTasks: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "task2"}, Status: entity.TaskInstanceStatusCanceled, Reason: ReasonParentCancel},
			},
			wantPatchDagCalled: true,
			wantPatchDagIns:    &entity.DagInstance{Status: entity.DagInstanceStatusCanceled, Reason: "task instance[task2] canceled"},
			wantRoot: &TaskNode{
				TaskInsID: virtualTaskRootID,
				Status:    entity.TaskInstanceStatusSuccess,
				children: []*TaskNode{
					{
						TaskInsID: "task1",
						Status:    entity.TaskInstanceStatusSuccess,
						children: []*TaskNode{
							{
								TaskInsID: "task2",
								Status:    entity.TaskInstanceStatusCanceled,
							},
						},
					},
				},
			},
			givePatchDagErr: fmt.Errorf("patch failed"),
			wantErr:         fmt.Errorf("patch failed"),
			wantDeleteTree:  true,
		},
		{
			caseDesc:   "dag can not modify",
			giveParser: &DefParser{},
			giveTasks: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "task1"}, TaskID: "task1", Status: entity.TaskInstanceStatusSuccess, DependOn: []string{}},
				{BaseInfo: entity.BaseInfo{ID: "task2"}, TaskID: "task2", Status: entity.TaskInstanceStatusInit, DependOn: []string{"task1"}},
			},
			giveDagIns:          &entity.DagInstance{Status: entity.DagInstanceStatusFailed},
			giveIds:             []string{"task2"},
			wantPatchTaskCalled: true,
			wantPatchTasks: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "task2"}, Status: entity.TaskInstanceStatusCanceled, Reason: ReasonParentCancel},
			},
			wantRoot: &TaskNode{
				TaskInsID: virtualTaskRootID,
				Status:    entity.TaskInstanceStatusSuccess,
				children: []*TaskNode{
					{
						TaskInsID: "task1",
						Status:    entity.TaskInstanceStatusSuccess,
						children: []*TaskNode{
							{
								TaskInsID: "task2",
								Status:    entity.TaskInstanceStatusCanceled,
							},
						},
					},
				},
			},
			wantDeleteTree: true,
		},
		{
			caseDesc:   "dag can not modify",
			giveParser: &DefParser{},
			giveTasks: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "task1"}, TaskID: "task1", Status: entity.TaskInstanceStatusSuccess, DependOn: []string{}},
				{BaseInfo: entity.BaseInfo{ID: "task2"}, TaskID: "task2", Status: entity.TaskInstanceStatusInit, DependOn: []string{"task1"}},
			},
			giveDagIns:          &entity.DagInstance{Status: entity.DagInstanceStatusFailed},
			giveIds:             []string{"task2"},
			wantPatchTaskCalled: true,
			wantPatchTasks: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "task2"}, Status: entity.TaskInstanceStatusCanceled, Reason: ReasonParentCancel},
			},
			wantRoot: &TaskNode{
				TaskInsID: virtualTaskRootID,
				Status:    entity.TaskInstanceStatusSuccess,
				children: []*TaskNode{
					{
						TaskInsID: "task1",
						Status:    entity.TaskInstanceStatusSuccess,
						children: []*TaskNode{
							{
								TaskInsID: "task2",
								Status:    entity.TaskInstanceStatusCanceled,
							},
						},
					},
				},
			},
			givePatchTaskErr: fmt.Errorf("patch task failed"),
			wantErr:          fmt.Errorf("patch task failed"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			root, err := BuildRootNode(MapTaskInsToGetter(tc.giveTasks))
			assert.NoError(t, err)
			tree := &TaskTree{
				DagIns: tc.giveDagIns,
				Root:   root,
			}
			mStore := &MockStore{}
			calledPatchDag, calledPatchTask := false, false
			patchTaskCnt := 0
			mStore.On("PatchTaskIns", mock.Anything).Run(func(args mock.Arguments) {
				calledPatchTask = true
				assert.Equal(t, tc.wantPatchTasks[patchTaskCnt], args.Get(0))
				patchTaskCnt++
			}).Return(tc.givePatchTaskErr)
			mStore.On("PatchDagIns", mock.Anything).Run(func(args mock.Arguments) {
				calledPatchDag = true
				assert.Equal(t, tc.wantPatchDagIns, args.Get(0))
			}).Return(tc.givePatchDagErr)
			SetStore(mStore)

			tc.giveParser.taskTrees.Store(tc.giveDagIns.ID, tree)
			err = tc.giveParser.cancelChildTasks(tree, tc.giveIds)
			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantPatchTaskCalled, calledPatchTask)
			assert.Equal(t, tc.wantPatchDagCalled, calledPatchDag)
			checkParentAndRemoveIt(t, tree.Root, nil)
			assert.Equal(t, tc.wantRoot, tree.Root)

			_, ok := tc.giveParser.taskTrees.Load(tc.giveDagIns.ID)
			assert.Equal(t, tc.wantDeleteTree, !ok)
		})
	}
}

func TestDefParser_executeNext(t *testing.T) {
	tests := []struct {
		caseDesc           string
		giveParser         *DefParser
		giveTaskIns        *entity.TaskInstance
		giveTaskTreeMap    map[string]*TaskTree
		giveListErr        error
		givePatchErr       error
		wantError          error
		wantPushedTaskIds  []string
		wantDelete         bool
		wantPatchCalled    bool
		wantPatchStatus    entity.DagInstanceStatus
		wantPushCalled     bool
		wantListTaskCalled bool
	}{
		{
			caseDesc:   "parent succeed",
			giveParser: &DefParser{},
			giveTaskTreeMap: map[string]*TaskTree{
				"dag1": {
					DagIns: &entity.DagInstance{
						BaseInfo: entity.BaseInfo{ID: "dag1"},
						Status:   entity.DagInstanceStatusRunning,
					},
					Root: &TaskNode{
						TaskInsID: "task-ins-id",
						Status:    entity.TaskInstanceStatusRunning,
						children: []*TaskNode{
							{TaskInsID: "child-task-id-1", Status: entity.TaskInstanceStatusInit},
							{TaskInsID: "child-task-id-2", Status: entity.TaskInstanceStatusInit},
						},
					},
				},
			},
			giveTaskIns: &entity.TaskInstance{
				BaseInfo: entity.BaseInfo{
					ID: "task-ins-id",
				},
				DagInsID: "dag1",
				Status:   entity.TaskInstanceStatusSuccess,
			},
			wantPushedTaskIds: []string{
				"child-task-id-1",
				"child-task-id-2",
			},
			wantPushCalled:     true,
			wantListTaskCalled: true,
		},
		{
			caseDesc:   "all success",
			giveParser: &DefParser{},
			giveTaskTreeMap: map[string]*TaskTree{
				"dag1": {
					DagIns: &entity.DagInstance{
						BaseInfo: entity.BaseInfo{ID: "dag1"},
						Status:   entity.DagInstanceStatusRunning,
					},
					Root: &TaskNode{
						TaskInsID: "task-ins-id",
						Status:    entity.TaskInstanceStatusSuccess,
						children: []*TaskNode{
							{TaskInsID: "child-task-id-1", Status: entity.TaskInstanceStatusInit},
							{TaskInsID: "child-task-id-2", Status: entity.TaskInstanceStatusSuccess},
						},
					},
				},
			},
			giveTaskIns: &entity.TaskInstance{
				BaseInfo: entity.BaseInfo{
					ID: "child-task-id-1",
				},
				DagInsID: "dag1",
				Status:   entity.TaskInstanceStatusSuccess,
			},
			wantPatchCalled: true,
			wantPatchStatus: entity.DagInstanceStatusSuccess,
			wantDelete:      true,
		},
		{
			caseDesc:   "branch failed",
			giveParser: &DefParser{},
			giveTaskTreeMap: map[string]*TaskTree{
				"dag1": {
					DagIns: &entity.DagInstance{
						BaseInfo: entity.BaseInfo{ID: "dag1"},
						Status:   entity.DagInstanceStatusRunning,
					},
					Root: &TaskNode{
						TaskInsID: "task-ins-id",
						Status:    entity.TaskInstanceStatusSuccess,
						children: []*TaskNode{
							{TaskInsID: "child-task-id-1", Status: entity.TaskInstanceStatusFailed},
							{TaskInsID: "child-task-id-2", Status: entity.TaskInstanceStatusRunning},
						},
					},
				},
			},
			giveTaskIns: &entity.TaskInstance{
				BaseInfo: entity.BaseInfo{
					ID: "child-task-id-2",
				},
				DagInsID: "dag1",
				Status:   entity.TaskInstanceStatusSuccess,
			},
			wantPatchCalled: true,
			wantPatchStatus: entity.DagInstanceStatusFailed,
			wantDelete:      true,
		},
		{
			caseDesc:   "parent failed",
			giveParser: &DefParser{},
			giveTaskTreeMap: map[string]*TaskTree{
				"dag1": {
					DagIns: &entity.DagInstance{
						BaseInfo: entity.BaseInfo{ID: "dag1"},
						Status:   entity.DagInstanceStatusRunning,
					},
					Root: &TaskNode{
						TaskInsID: "task-ins-id",
						Status:    entity.TaskInstanceStatusRunning,
						children: []*TaskNode{
							{TaskInsID: "child-task-id-1", Status: entity.TaskInstanceStatusInit},
							{TaskInsID: "child-task-id-2", Status: entity.TaskInstanceStatusInit},
						},
					},
				},
			},
			giveTaskIns: &entity.TaskInstance{
				BaseInfo: entity.BaseInfo{
					ID: "task-ins-id",
				},
				DagInsID: "dag1",
				Status:   entity.TaskInstanceStatusFailed,
			},
			wantPatchStatus: entity.DagInstanceStatusFailed,
			wantPatchCalled: true,
			wantDelete:      true,
		},
		{
			caseDesc:   "branch succeed but dag failed",
			giveParser: &DefParser{},
			giveTaskTreeMap: map[string]*TaskTree{
				"dag1": {
					DagIns: &entity.DagInstance{
						BaseInfo: entity.BaseInfo{ID: "dag1"},
						Status:   entity.DagInstanceStatusRunning,
					},
					Root: &TaskNode{
						TaskInsID: "task-ins-id",
						Status:    entity.TaskInstanceStatusSuccess,
						children: []*TaskNode{
							{TaskInsID: "child-task-id-1", Status: entity.TaskInstanceStatusFailed},
							{TaskInsID: "child-task-id-2", Status: entity.TaskInstanceStatusRunning},
						},
					},
				},
			},
			giveTaskIns: &entity.TaskInstance{
				BaseInfo: entity.BaseInfo{
					ID: "child-task-id-2",
				},
				DagInsID: "dag1",
				Status:   entity.TaskInstanceStatusSuccess,
			},
			wantDelete:      true,
			wantPatchStatus: entity.DagInstanceStatusFailed,
			wantPatchCalled: true,
		},
		{
			caseDesc:   "child block but parent not success",
			giveParser: &DefParser{},
			giveTaskTreeMap: map[string]*TaskTree{
				"dag1": {
					DagIns: &entity.DagInstance{
						BaseInfo: entity.BaseInfo{ID: "dag1"},
						Status:   entity.DagInstanceStatusRunning,
					},
					Root: &TaskNode{
						TaskInsID: "task-ins-id",
						Status:    entity.TaskInstanceStatusRunning,
						children: []*TaskNode{
							{TaskInsID: "child-task-id-1", Status: entity.TaskInstanceStatusInit},
							{TaskInsID: "child-task-id-2", Status: entity.TaskInstanceStatusInit},
						},
					},
				},
			},
			giveTaskIns: &entity.TaskInstance{
				BaseInfo: entity.BaseInfo{
					ID: "child-task-id-1",
				},
				DagInsID: "dag1",
				Status:   entity.TaskInstanceStatusBlocked,
			},
			wantError: errors.New("task instance[child-task-id-1] does not found normal node"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			for k, v := range tc.giveTaskTreeMap {
				tc.giveParser.taskTrees.Store(k, v)
			}

			preTask := &entity.TaskInstance{}
			calledPatch, calledList, calledPush := false, false, false
			mStore := &MockStore{}
			mStore.On("PatchDagIns", mock.Anything).Run(func(args mock.Arguments) {
				calledPatch = true
				assert.Equal(t, tc.wantPatchStatus, args.Get(0).(*entity.DagInstance).Status)
			}).Return(tc.givePatchErr)
			mStore.On("ListTaskInstance", mock.Anything).Run(func(args mock.Arguments) {
				calledList = true
			}).Return([]*entity.TaskInstance{preTask}, tc.giveListErr)
			SetStore(mStore)

			mExecutor := &MockExecutor{}
			mExecutor.On("Push", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				calledPush = true
				assert.Equal(t, tc.giveTaskTreeMap[tc.giveTaskIns.DagInsID].DagIns, args.Get(0))
				assert.Equal(t, preTask, args.Get(1))
			})
			SetExecutor(mExecutor)

			err := tc.giveParser.executeNext(tc.giveTaskIns)
			assert.Equal(t, tc.wantError, err)
			assert.Equal(t, tc.wantPatchCalled, calledPatch)
			assert.Equal(t, tc.wantPushCalled, calledPush)
			assert.Equal(t, tc.wantListTaskCalled, calledList)
			_, find := tc.giveParser.taskTrees.Load(tc.giveTaskIns.DagInsID)
			assert.Equal(t, tc.wantDelete, !find)
		})
	}
}

func TestDefParser_EntryTaskIns(t *testing.T) {
	tests := []struct {
		caseDesc       string
		giveParser     *DefParser
		giveTaskIns    []*entity.TaskInstance
		giveWorkerNum  int
		giveWorkerCost time.Duration

		closeWhenTasks int

		wantTaskMap map[int][]*entity.TaskInstance
	}{
		{
			caseDesc:      "sanity",
			giveParser:    &DefParser{},
			giveWorkerNum: 6,
			giveTaskIns: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "task-1"}, DagInsID: "dag-ins-1"},
				{BaseInfo: entity.BaseInfo{ID: "task-2"}, DagInsID: "dag-ins-2"},
				{BaseInfo: entity.BaseInfo{ID: "task-3"}, DagInsID: "dag-ins-3"},
				{BaseInfo: entity.BaseInfo{ID: "task-4"}, DagInsID: "dag-ins-4"},
				{BaseInfo: entity.BaseInfo{ID: "task-5"}, DagInsID: "dag-ins-4"},
				{BaseInfo: entity.BaseInfo{ID: "task-6"}, DagInsID: "dag-ins-3"},
				{BaseInfo: entity.BaseInfo{ID: "task-7"}, DagInsID: "dag-ins-2"},
				{BaseInfo: entity.BaseInfo{ID: "task-8"}, DagInsID: "dag-ins-1"},
			},
			wantTaskMap: map[int][]*entity.TaskInstance{
				0: {
					{BaseInfo: entity.BaseInfo{ID: "task-1"}, DagInsID: "dag-ins-1"},
					{BaseInfo: entity.BaseInfo{ID: "task-2"}, DagInsID: "dag-ins-2"},
					{BaseInfo: entity.BaseInfo{ID: "task-7"}, DagInsID: "dag-ins-2"},
					{BaseInfo: entity.BaseInfo{ID: "task-8"}, DagInsID: "dag-ins-1"},
				},
				3: {
					{BaseInfo: entity.BaseInfo{ID: "task-3"}, DagInsID: "dag-ins-3"},
					{BaseInfo: entity.BaseInfo{ID: "task-6"}, DagInsID: "dag-ins-3"},
				},
				5: {
					{BaseInfo: entity.BaseInfo{ID: "task-4"}, DagInsID: "dag-ins-4"},
					{BaseInfo: entity.BaseInfo{ID: "task-5"}, DagInsID: "dag-ins-4"},
				},
			},
		},
		{
			caseDesc:       "test queue full",
			giveParser:     &DefParser{},
			giveWorkerNum:  1,
			giveWorkerCost: 10 * time.Millisecond,
			giveTaskIns: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "task-1"}, DagInsID: "dag-ins-1"},
				{BaseInfo: entity.BaseInfo{ID: "task-2"}, DagInsID: "dag-ins-2"},
				{BaseInfo: entity.BaseInfo{ID: "task-3"}, DagInsID: "dag-ins-3"},
			},
			wantTaskMap: map[int][]*entity.TaskInstance{
				0: {
					{BaseInfo: entity.BaseInfo{ID: "task-1"}, DagInsID: "dag-ins-1"},
					{BaseInfo: entity.BaseInfo{ID: "task-2"}, DagInsID: "dag-ins-2"},
					{BaseInfo: entity.BaseInfo{ID: "task-3"}, DagInsID: "dag-ins-3"},
				},
			},
		},
		{
			caseDesc:       "test queue close",
			giveParser:     &DefParser{},
			giveWorkerNum:  1,
			closeWhenTasks: 1,
			giveTaskIns: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "task-1"}, DagInsID: "dag-ins-1"},
				{BaseInfo: entity.BaseInfo{ID: "task-2"}, DagInsID: "dag-ins-2"},
				{BaseInfo: entity.BaseInfo{ID: "task-3"}, DagInsID: "dag-ins-3"},
			},
			wantTaskMap: map[int][]*entity.TaskInstance{
				0: {
					{BaseInfo: entity.BaseInfo{ID: "task-1"}, DagInsID: "dag-ins-1"},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			mutex := sync.Mutex{}
			ret := map[int][]*entity.TaskInstance{}
			tc.giveParser.closeCh = make(chan struct{})
			tc.giveParser.workerNumber = tc.giveWorkerNum
			for i := 0; i < tc.giveParser.workerNumber; i++ {
				// if you make a block channel, will cause select failed
				q := make(chan *entity.TaskInstance, 1)
				tc.giveParser.workerQueue = append(tc.giveParser.workerQueue, q)
				tc.giveParser.workerWg.Add(1)
				go func(idx int, queue <-chan *entity.TaskInstance) {
					for task := range queue {
						if tc.giveWorkerCost > 0 {
							time.Sleep(tc.giveWorkerCost)
						}
						mutex.Lock()
						ret[idx] = append(ret[idx], task)
						mutex.Unlock()
					}
					tc.giveParser.workerWg.Done()
				}(i, q)
			}

			for idx, task := range tc.giveTaskIns {
				if tc.closeWhenTasks > 0 && idx >= tc.closeWhenTasks {
					tc.giveParser.Close()
				}
				tc.giveParser.EntryTaskIns(task)
			}

			time.Sleep(time.Millisecond * 500)
			tc.giveParser.Close()
			for key := range tc.wantTaskMap {
				assert.ElementsMatch(t, tc.wantTaskMap[key], ret[key])
			}
		})
	}
}

func TestDefParser_InitialDagIns(t *testing.T) {
	tests := []struct {
		caseDesc        string
		giveParser      *DefParser
		giveDagIns      *entity.DagInstance
		giveTaskIns     []*entity.TaskInstance
		giveListTaskErr error
		wantPushTasks   []*entity.TaskInstance
		wantErrorLog    bool
		wantPatchDagIns *entity.DagInstance
		wantPatchCalled bool
	}{
		{
			caseDesc:   "normal",
			giveParser: &DefParser{},
			giveDagIns: &entity.DagInstance{BaseInfo: entity.BaseInfo{ID: "test-dag"}},
			giveTaskIns: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "root-1-ins"}, TaskID: "root-1", Status: entity.TaskInstanceStatusInit},
				{BaseInfo: entity.BaseInfo{ID: "r1-child-1-ins"}, TaskID: "r1-child-1", Status: entity.TaskInstanceStatusInit, DependOn: []string{"root-1"}},
				{BaseInfo: entity.BaseInfo{ID: "r1-child-2-ins"}, TaskID: "r1-child-2", Status: entity.TaskInstanceStatusInit, DependOn: []string{"root-1"}},
				{BaseInfo: entity.BaseInfo{ID: "root-2-ins"}, TaskID: "root-2", Status: entity.TaskInstanceStatusInit},
			},
			wantPushTasks: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "root-1-ins"}, TaskID: "root-1", Status: entity.TaskInstanceStatusInit},
				{BaseInfo: entity.BaseInfo{ID: "root-2-ins"}, TaskID: "root-2", Status: entity.TaskInstanceStatusInit},
			},
		},
		{
			caseDesc:   "patch success dag",
			giveParser: &DefParser{},
			giveDagIns: &entity.DagInstance{BaseInfo: entity.BaseInfo{ID: "test-dag"}},
			giveTaskIns: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "root-1-ins"}, TaskID: "root-1", Status: entity.TaskInstanceStatusSuccess},
				{BaseInfo: entity.BaseInfo{ID: "r1-child-1-ins"}, TaskID: "r1-child-1", Status: entity.TaskInstanceStatusSuccess, DependOn: []string{"root-1"}},
				{BaseInfo: entity.BaseInfo{ID: "r1-child-2-ins"}, TaskID: "r1-child-2", Status: entity.TaskInstanceStatusSuccess, DependOn: []string{"root-1"}},
				{BaseInfo: entity.BaseInfo{ID: "root-2-ins"}, TaskID: "root-2", Status: entity.TaskInstanceStatusSuccess},
			},
			wantPatchDagIns: &entity.DagInstance{
				BaseInfo: entity.BaseInfo{
					ID: "test-dag",
				},
				Status: entity.DagInstanceStatusSuccess,
			},
			wantPatchCalled: true,
		},
		{
			caseDesc:   "patch failed dag",
			giveParser: &DefParser{},
			giveDagIns: &entity.DagInstance{BaseInfo: entity.BaseInfo{ID: "test-dag"}},
			giveTaskIns: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "root-1-ins"}, TaskID: "root-1", Status: entity.TaskInstanceStatusFailed},
				{BaseInfo: entity.BaseInfo{ID: "r1-child-1-ins"}, TaskID: "r1-child-1", Status: entity.TaskInstanceStatusSuccess, DependOn: []string{"root-1"}},
				{BaseInfo: entity.BaseInfo{ID: "r1-child-2-ins"}, TaskID: "r1-child-2", Status: entity.TaskInstanceStatusSuccess, DependOn: []string{"root-1"}},
				{BaseInfo: entity.BaseInfo{ID: "root-2-ins"}, TaskID: "root-2", Status: entity.TaskInstanceStatusSuccess},
			},
			wantPatchDagIns: &entity.DagInstance{
				BaseInfo: entity.BaseInfo{
					ID: "test-dag",
				},
				Status: entity.DagInstanceStatusFailed,
			},
			wantPatchCalled: true,
		},
		{
			caseDesc:   "patch failed dag",
			giveParser: &DefParser{},
			giveDagIns: &entity.DagInstance{BaseInfo: entity.BaseInfo{ID: "test-dag"}},
			giveTaskIns: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "root-1-ins"}, TaskID: "root-1", Status: entity.TaskInstanceStatusSuccess},
				{BaseInfo: entity.BaseInfo{ID: "r1-child-1-ins"}, TaskID: "r1-child-1", Status: entity.TaskInstanceStatusBlocked, DependOn: []string{"root-1"}},
				{BaseInfo: entity.BaseInfo{ID: "r1-child-2-ins"}, TaskID: "r1-child-2", Status: entity.TaskInstanceStatusSuccess, DependOn: []string{"root-1"}},
				{BaseInfo: entity.BaseInfo{ID: "root-2-ins"}, TaskID: "root-2", Status: entity.TaskInstanceStatusSuccess},
			},
			wantPatchDagIns: &entity.DagInstance{
				BaseInfo: entity.BaseInfo{
					ID: "test-dag",
				},
				Status: entity.DagInstanceStatusBlocked,
			},
			wantPatchCalled: true,
		},
		{
			caseDesc:   "list failed",
			giveParser: &DefParser{},
			giveDagIns: &entity.DagInstance{BaseInfo: entity.BaseInfo{ID: "test-dag"}},
			giveTaskIns: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "root-1-ins"}, TaskID: "root-1", Status: entity.TaskInstanceStatusInit},
				{BaseInfo: entity.BaseInfo{ID: "r1-child-1-ins"}, TaskID: "r1-child-1", Status: entity.TaskInstanceStatusInit, DependOn: []string{"root-1"}},
				{BaseInfo: entity.BaseInfo{ID: "r1-child-2-ins"}, TaskID: "r1-child-2", Status: entity.TaskInstanceStatusInit, DependOn: []string{"root-1"}},
				{BaseInfo: entity.BaseInfo{ID: "root-2-ins"}, TaskID: "root-2", Status: entity.TaskInstanceStatusInit},
			},
			giveListTaskErr: fmt.Errorf("list failed"),
			wantErrorLog:    true,
		},
		{
			caseDesc:   "build tree failed",
			giveParser: &DefParser{},
			giveDagIns: &entity.DagInstance{BaseInfo: entity.BaseInfo{ID: "test-dag"}},
			giveTaskIns: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "root-1-ins"}, TaskID: "root-1", Status: entity.TaskInstanceStatusInit},
				{BaseInfo: entity.BaseInfo{ID: "root-1-ins"}, TaskID: "root-1", Status: entity.TaskInstanceStatusInit, DependOn: []string{"root-1"}},
			},
			wantErrorLog: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			listCalled, errorCalled, patchCalled := false, false, false
			mStore := &MockStore{}
			mStore.On("ListTaskInstance", mock.Anything).Run(func(args mock.Arguments) {
				listCalled = true
			}).Return(tc.giveTaskIns, tc.giveListTaskErr)
			mStore.On("PatchDagIns", mock.Anything).Run(func(args mock.Arguments) {
				patchCalled = true
				assert.Equal(t, tc.wantPatchDagIns, args.Get(0))
			}).Return(nil)
			SetStore(mStore)

			mLog := &log.MockLogger{}
			mLog.On("Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				errorCalled = true
			})
			log.SetLogger(mLog)

			var pushedTasks []*entity.TaskInstance
			mExec := &MockExecutor{}
			mExec.On("Push", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				assert.Equal(t, tc.giveDagIns, args.Get(0))
				pushedTasks = append(pushedTasks, args.Get(1).(*entity.TaskInstance))
			})
			SetExecutor(mExec)

			tc.giveParser.InitialDagIns(tc.giveDagIns)
			assert.Equal(t, tc.wantPushTasks, pushedTasks)
			assert.True(t, listCalled)
			assert.Equal(t, tc.wantErrorLog, errorCalled)
			assert.Equal(t, tc.wantPatchCalled, patchCalled)
		})
	}
}

func TestDefParser_WatchScheduledDagIns(t *testing.T) {
	tests := []struct {
		caseDesc            string
		giveWorkerKey       string
		giveListRet         []*entity.DagInstance
		giveListErr         error
		giveGetErr          error
		wantErr             error
		wantListInput       *ListDagInstanceInput
		wantGetCalled       bool
		wantGetDagInsCalled bool
	}{
		{
			caseDesc:      "sanity",
			giveWorkerKey: "test",
			giveListRet: []*entity.DagInstance{
				{
					Status: entity.DagInstanceStatusRunning,
				},
			},
			wantListInput: &ListDagInstanceInput{
				Worker: "test",
				Status: []entity.DagInstanceStatus{entity.DagInstanceStatusScheduled},
			},
			wantGetDagInsCalled: true,
		},
		{
			caseDesc:      "list failed",
			giveWorkerKey: "test",
			giveListRet: []*entity.DagInstance{
				{
					Status: entity.DagInstanceStatusRunning,
				},
			},
			giveListErr: fmt.Errorf("list failed"),
			wantErr:     fmt.Errorf("watch scheduled dag ins failed: %w", fmt.Errorf("list failed")),
			wantListInput: &ListDagInstanceInput{
				Worker: "test",
				Status: []entity.DagInstanceStatus{entity.DagInstanceStatusScheduled},
			},
		},
		{
			caseDesc:      "get dag failed",
			giveWorkerKey: "test",
			giveListRet: []*entity.DagInstance{
				{
					Status: entity.DagInstanceStatusScheduled,
				},
			},
			giveGetErr: fmt.Errorf("get failed"),
			wantErr:    fmt.Errorf("watch scheduled dag ins failed: %w", fmt.Errorf("get failed")),
			wantListInput: &ListDagInstanceInput{
				Worker: "test",
				Status: []entity.DagInstanceStatus{entity.DagInstanceStatusScheduled},
			},
			wantGetCalled: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			calledList, calledGet, calledKeeper, calledListTask := false, false, false, false
			mStore := &MockStore{}
			mStore.On("ListDagInstance", mock.Anything).Run(func(args mock.Arguments) {
				calledList = true
				assert.Equal(t, tc.wantListInput, args.Get(0))
			}).Return(tc.giveListRet, tc.giveListErr)
			mStore.On("ListTaskInstance", mock.Anything).Run(func(args mock.Arguments) {
				calledListTask = true
			}).Return(nil, nil)
			mStore.On("GetDag", mock.Anything).Run(func(args mock.Arguments) {
				calledGet = true
			}).Return(nil, tc.giveGetErr)
			SetStore(mStore)

			mKeeper := &MockKeeper{}
			mKeeper.On("WorkerKey").Run(func(args mock.Arguments) {
				calledKeeper = true
			}).Return(tc.giveWorkerKey)
			SetKeeper(mKeeper)

			p := &DefParser{}
			err := p.watchScheduledDagIns()
			assert.Equal(t, tc.wantErr, err)
			assert.True(t, calledList)
			assert.True(t, calledKeeper)
			assert.Equal(t, tc.wantGetCalled, calledGet)
			if err == nil {
				assert.True(t, calledListTask)
			}
		})
	}
}

func TestDefParser_WatchDagInsCmd(t *testing.T) {
	tests := []struct {
		caseDesc            string
		giveWorkerKey       string
		giveListRet         []*entity.DagInstance
		giveListErr         error
		giveUpdateErr       error
		wantErr             error
		wantListInput       *ListDagInstanceInput
		wantUpdateCalled    bool
		wantGetDagInsCalled bool
	}{
		{
			caseDesc:      "sanity",
			giveWorkerKey: "test",
			giveListRet: []*entity.DagInstance{
				{
					Cmd: &entity.Command{
						Name: "test",
					},
				},
			},
			wantListInput: &ListDagInstanceInput{
				Worker: "test",
				HasCmd: true,
			},
			wantUpdateCalled:    true,
			wantGetDagInsCalled: true,
		},
		{
			caseDesc:      "list failed",
			giveWorkerKey: "test",
			giveListRet: []*entity.DagInstance{
				{
					Cmd: &entity.Command{
						Name: "test",
					},
				},
			},
			giveListErr: fmt.Errorf("list failed"),
			wantErr:     fmt.Errorf("watch dag command failed: %w", fmt.Errorf("list failed")),
			wantListInput: &ListDagInstanceInput{
				Worker: "test",
				HasCmd: true,
			},
		},
		{
			caseDesc:      "update failed",
			giveWorkerKey: "test",
			giveListRet: []*entity.DagInstance{
				{
					Cmd: &entity.Command{
						Name: "test",
					},
				},
			},
			giveUpdateErr: fmt.Errorf("update failed"),
			wantErr:       fmt.Errorf("watch dag command failed: %w", fmt.Errorf("update failed")),
			wantListInput: &ListDagInstanceInput{
				Worker: "test",
				HasCmd: true,
			},
			wantUpdateCalled: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			calledList, calledUpdate, calledKeeper := false, false, false
			mStore := &MockStore{}
			mStore.On("ListDagInstance", mock.Anything).Run(func(args mock.Arguments) {
				calledList = true
				assert.Equal(t, tc.wantListInput, args.Get(0))
			}).Return(tc.giveListRet, tc.giveListErr)
			mStore.On("PatchDagIns", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				calledUpdate = true
			}).Return(tc.giveUpdateErr)
			SetStore(mStore)

			mKeeper := &MockKeeper{}
			mKeeper.On("WorkerKey").Run(func(args mock.Arguments) {
				calledKeeper = true
			}).Return(tc.giveWorkerKey)
			SetKeeper(mKeeper)

			p := &DefParser{}
			err := p.watchDagInsCmd()
			assert.Equal(t, tc.wantErr, err)
			assert.True(t, calledList)
			assert.True(t, calledKeeper)
			assert.Equal(t, tc.wantUpdateCalled, calledUpdate)
		})
	}
}

func TestDefParser_PublisherDo(t *testing.T) {
	tests := []struct {
		giveListRet     []*entity.DagInstance
		giveListErr     error
		giveWorkerKey   string
		wantErr         error
		wantListInput   *ListDagInstanceInput
		wantPublishList []string
	}{
		{
			giveListRet:   []*entity.DagInstance{{BaseInfo: entity.BaseInfo{ID: "test1"}}, {BaseInfo: entity.BaseInfo{ID: "test2"}}},
			giveWorkerKey: "test1",
			wantListInput: &ListDagInstanceInput{
				Worker: "test1",
				Status: []entity.DagInstanceStatus{entity.DagInstanceStatusRunning},
			},
			wantPublishList: []string{"test1", "test2"},
		},
		{
			wantListInput: &ListDagInstanceInput{
				Status: []entity.DagInstanceStatus{entity.DagInstanceStatusRunning},
			},
			giveListErr: fmt.Errorf("list failed"),
			wantErr:     fmt.Errorf("list failed"),
		},
	}

	for _, tc := range tests {
		queue := make(chan string)
		var entryDagInsIds []string
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			for id := range queue {
				entryDagInsIds = append(entryDagInsIds, id)
			}
			wg.Done()
		}()

		parser := DefParser{}
		calledList, calledKeeper := false, false

		mStore := &MockStore{}
		mStore.On("ListDagInstance", mock.Anything).Run(func(args mock.Arguments) {
			calledList = true
			assert.Equal(t, tc.wantListInput, args.Get(0))
		}).Return(tc.giveListRet, tc.giveListErr)
		mStore.On("ListTaskInstance", mock.Anything).Run(func(args mock.Arguments) {
			queue <- args.Get(0).(*ListTaskInstanceInput).DagInsID
		}).Return(nil, nil)
		SetStore(mStore)

		mKeeper := &MockKeeper{}
		mKeeper.On("WorkerKey").Run(func(args mock.Arguments) {
			calledKeeper = true
		}).Return(tc.giveWorkerKey)
		SetKeeper(mKeeper)

		err := parser.initialRunningDagIns()
		close(queue)
		wg.Wait()
		assert.True(t, calledList)
		assert.True(t, calledKeeper)
		assert.Equal(t, tc.wantErr, err)
		if err != nil {
			continue
		}
		assert.Equal(t, tc.wantPublishList, entryDagInsIds)
	}
}

func TestDefParser_ParseScheduleDagIns(t *testing.T) {
	tests := []struct {
		caseDesc                 string
		giveDagIns               *entity.DagInstance
		giveDagRet               *entity.Dag
		giveDagErr               error
		giveTasksIns             []*entity.TaskInstance
		giveTasksInsErr          error
		giveBatchCreateTaskErr   error
		giveUpdateDagInsErr      error
		wantErr                  error
		wantGetDagInput          string
		wantTaskInsInput         *ListTaskInstanceInput
		wantBatchCreateTaskInput []*entity.TaskInstance
		wantPatchDagInsInput     *entity.DagInstance
	}{
		{
			caseDesc:   "sanity",
			giveDagIns: &entity.DagInstance{BaseInfo: entity.BaseInfo{ID: "dagInsId"}, DagID: "dagId", Status: entity.DagInstanceStatusScheduled},
			giveDagRet: &entity.Dag{
				Tasks: []entity.Task{
					{ID: "task1", Name: "t1", TimeoutSecs: 10, Params: map[string]interface{}{"test": "value1"}},
					{ID: "task2", Name: "t2", DependOn: []string{"task1"}},
				},
			},
			wantGetDagInput:  "dagId",
			wantTaskInsInput: &ListTaskInstanceInput{DagInsID: "dagInsId"},
			wantBatchCreateTaskInput: []*entity.TaskInstance{
				{TaskID: "task1", Name: "t1", TimeoutSecs: 10, DagInsID: "dagInsId", Params: map[string]interface{}{"test": "value1"}, Status: entity.TaskInstanceStatusInit},
				{TaskID: "task2", Name: "t2", DependOn: []string{"task1"}, DagInsID: "dagInsId", Status: entity.TaskInstanceStatusInit},
			},
			wantPatchDagInsInput: &entity.DagInstance{BaseInfo: entity.BaseInfo{ID: "dagInsId"}, Status: entity.DagInstanceStatusRunning},
		},
		{
			caseDesc:   "task ins is not correct",
			giveDagIns: &entity.DagInstance{BaseInfo: entity.BaseInfo{ID: "dagInsId"}, DagID: "dagId", Status: entity.DagInstanceStatusScheduled},
			giveDagRet: &entity.Dag{
				Tasks: []entity.Task{
					{ID: "task1"},
					{ID: "task2"},
				},
			},
			giveTasksIns: []*entity.TaskInstance{
				{TaskID: "task1"},
			},
			wantGetDagInput:  "dagId",
			wantTaskInsInput: &ListTaskInstanceInput{DagInsID: "dagInsId"},
			wantBatchCreateTaskInput: []*entity.TaskInstance{
				{TaskID: "task2", DagInsID: "dagInsId", Status: entity.TaskInstanceStatusInit},
			},
			wantPatchDagInsInput: &entity.DagInstance{BaseInfo: entity.BaseInfo{ID: "dagInsId"}, Status: entity.DagInstanceStatusRunning},
		},
		{
			caseDesc:        "get dag failed",
			giveDagIns:      &entity.DagInstance{BaseInfo: entity.BaseInfo{ID: "dagInsId"}, DagID: "dagId", Status: entity.DagInstanceStatusScheduled},
			giveDagErr:      fmt.Errorf("get dag failed"),
			wantErr:         fmt.Errorf("get dag failed"),
			wantGetDagInput: "dagId",
		},
		{
			caseDesc:   "list task failed",
			giveDagIns: &entity.DagInstance{BaseInfo: entity.BaseInfo{ID: "dagInsId"}, DagID: "dagId", Status: entity.DagInstanceStatusScheduled},
			giveDagRet: &entity.Dag{
				Tasks: []entity.Task{
					{ID: "task1"},
					{ID: "task2"},
				},
			},
			giveTasksInsErr:  fmt.Errorf("list task failed"),
			wantGetDagInput:  "dagId",
			wantTaskInsInput: &ListTaskInstanceInput{DagInsID: "dagInsId"},
			wantErr:          fmt.Errorf("list task failed"),
		},
		{
			caseDesc:   "batch update failed",
			giveDagIns: &entity.DagInstance{BaseInfo: entity.BaseInfo{ID: "dagInsId"}, DagID: "dagId", Status: entity.DagInstanceStatusScheduled},
			giveDagRet: &entity.Dag{
				Tasks: []entity.Task{
					{ID: "task2"},
				},
			},
			giveBatchCreateTaskErr: fmt.Errorf("batch update failed"),
			wantErr:                fmt.Errorf("batch update failed"),
			wantGetDagInput:        "dagId",
			wantTaskInsInput:       &ListTaskInstanceInput{DagInsID: "dagInsId"},
			wantBatchCreateTaskInput: []*entity.TaskInstance{
				{TaskID: "task2", DagInsID: "dagInsId", Status: entity.TaskInstanceStatusInit},
			},
		},
		{
			caseDesc:   "update dag ins failed",
			giveDagIns: &entity.DagInstance{BaseInfo: entity.BaseInfo{ID: "dagInsId"}, DagID: "dagId", Status: entity.DagInstanceStatusScheduled},
			giveDagRet: &entity.Dag{
				Tasks: []entity.Task{
					{ID: "task2"},
				},
			},
			giveUpdateDagInsErr: fmt.Errorf("update dag failed"),
			wantErr:             fmt.Errorf("update dag failed"),
			wantGetDagInput:     "dagId",
			wantTaskInsInput:    &ListTaskInstanceInput{DagInsID: "dagInsId"},
			wantBatchCreateTaskInput: []*entity.TaskInstance{
				{TaskID: "task2", DagInsID: "dagInsId", Status: entity.TaskInstanceStatusInit},
			},
			wantPatchDagInsInput: &entity.DagInstance{BaseInfo: entity.BaseInfo{ID: "dagInsId"}, Status: entity.DagInstanceStatusRunning},
		},
		{
			caseDesc:   "not scheduled dag",
			giveDagIns: &entity.DagInstance{BaseInfo: entity.BaseInfo{ID: "dagInsId"}, DagID: "dagId", Status: entity.DagInstanceStatusRunning},
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			parser := &DefParser{}

			calledDag, calledTaskIns, calledBatchCreate, calledUpdateDagIns := false, false, false, false
			mStore := &MockStore{}
			mStore.On("GetDag", mock.Anything).Run(func(args mock.Arguments) {
				calledDag = true
				assert.Equal(t, tc.wantGetDagInput, args.Get(0))
			}).Return(tc.giveDagRet, tc.giveDagErr)

			mStore.On("ListTaskInstance", mock.Anything).Run(func(args mock.Arguments) {
				calledTaskIns = true
				assert.Equal(t, tc.wantTaskInsInput, args.Get(0))
			}).Return(tc.giveTasksIns, tc.giveTasksInsErr)

			mStore.On("BatchCreatTaskIns", mock.Anything).Run(func(args mock.Arguments) {
				calledBatchCreate = true
				assert.Equal(t, tc.wantBatchCreateTaskInput, args.Get(0))
			}).Return(tc.giveBatchCreateTaskErr)

			mStore.On("PatchDagIns", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				calledUpdateDagIns = true
				assert.Equal(t, tc.wantPatchDagInsInput, args.Get(0))
			}).Return(tc.giveUpdateDagInsErr)
			SetStore(mStore)

			err := parser.parseScheduleDagIns(tc.giveDagIns)
			if err != nil {
				assert.Equal(t, tc.wantErr, err)
				return
			}
			if tc.giveDagIns.Status != entity.DagInstanceStatusScheduled {
				return
			}
			assert.True(t, calledDag)
			assert.True(t, calledTaskIns)
			assert.True(t, calledBatchCreate)
			assert.True(t, calledUpdateDagIns)
		})
	}
}

func TestDefParser_ParseCmd(t *testing.T) {
	tests := []struct {
		caseDesc             string
		giveDagIns           *entity.DagInstance
		giveTask             []*entity.TaskInstance
		giveTaskErr          error
		giveUpdateTaskErr    error
		giveCancelTaskErr    error
		giveUpdateDagInsErr  error
		givePushErr          error
		wantErr              error
		wantGetTaskId        string
		wantUpdateDagIns     *entity.DagInstance
		wantUpdateTask       *entity.TaskInstance
		wantCancelTaskId     []string
		wantUpdateTaskCalled bool
		wantUpdateDagCalled  bool
		wantCancelCalled     bool
		wantListCallCnt      int
	}{
		{
			caseDesc: "retry failed task",
			giveDagIns: &entity.DagInstance{
				Status: entity.DagInstanceStatusFailed,
				Cmd:    &entity.Command{Name: entity.CommandNameRetry, TargetTaskInsIDs: []string{"task1"}}},
			wantGetTaskId: "task1",
			giveTask: []*entity.TaskInstance{
				{Status: entity.TaskInstanceStatusFailed, Reason: "failed reason"},
			},
			wantListCallCnt:      2,
			wantUpdateTask:       &entity.TaskInstance{Status: entity.TaskInstanceStatusRetrying},
			wantUpdateTaskCalled: true,
			wantUpdateDagIns:     &entity.DagInstance{Status: entity.DagInstanceStatusRunning},
			wantUpdateDagCalled:  true,
		},
		{
			caseDesc: "retry canceled task",
			giveDagIns: &entity.DagInstance{
				Status: entity.DagInstanceStatusFailed,
				Cmd:    &entity.Command{Name: entity.CommandNameRetry, TargetTaskInsIDs: []string{"task1"}}},
			wantGetTaskId: "task1",
			giveTask: []*entity.TaskInstance{
				{Status: entity.TaskInstanceStatusCanceled, Reason: "canceled reason"},
			},
			wantListCallCnt:      2,
			wantUpdateTask:       &entity.TaskInstance{Status: entity.TaskInstanceStatusRetrying},
			wantUpdateTaskCalled: true,
			wantUpdateDagIns:     &entity.DagInstance{Status: entity.DagInstanceStatusRunning},
			wantUpdateDagCalled:  true,
		},
		{
			caseDesc:      "retry not failed",
			giveDagIns:    &entity.DagInstance{Cmd: &entity.Command{Name: entity.CommandNameRetry, TargetTaskInsIDs: []string{"task1"}}},
			wantGetTaskId: "task1",
			giveTask: []*entity.TaskInstance{
				{Status: entity.TaskInstanceStatusRunning},
			},
			wantListCallCnt:     1,
			wantUpdateDagIns:    &entity.DagInstance{Status: entity.DagInstanceStatusRunning},
			wantUpdateDagCalled: true,
		},
		{
			caseDesc:        "retry get task failed",
			giveDagIns:      &entity.DagInstance{Cmd: &entity.Command{Name: entity.CommandNameRetry, TargetTaskInsIDs: []string{"task1"}}},
			wantGetTaskId:   "task1",
			giveTaskErr:     fmt.Errorf("get task failed"),
			wantErr:         fmt.Errorf("get task failed"),
			wantListCallCnt: 1,
		},
		{
			caseDesc:            "cancel sanity",
			giveDagIns:          &entity.DagInstance{Cmd: &entity.Command{Name: entity.CommandNameCancel, TargetTaskInsIDs: []string{"task1"}}},
			wantCancelTaskId:    []string{"task1"},
			wantCancelCalled:    true,
			wantUpdateDagIns:    &entity.DagInstance{},
			wantUpdateDagCalled: true,
		},
		{
			caseDesc:          "cancel failed",
			giveDagIns:        &entity.DagInstance{Cmd: &entity.Command{Name: entity.CommandNameCancel, TargetTaskInsIDs: []string{"task1"}}},
			giveCancelTaskErr: fmt.Errorf("cancel failed"),
			wantErr:           fmt.Errorf("cancel failed"),
			wantCancelTaskId:  []string{"task1"},
			wantCancelCalled:  true,
		},
		{
			caseDesc: "continue blocked task",
			giveDagIns: &entity.DagInstance{
				Status: entity.DagInstanceStatusBlocked,
				Cmd:    &entity.Command{Name: entity.CommandNameContinue, TargetTaskInsIDs: []string{"task1"}}},
			wantGetTaskId: "task1",
			giveTask: []*entity.TaskInstance{
				{Status: entity.TaskInstanceStatusBlocked},
			},
			wantListCallCnt:      2,
			wantUpdateTask:       &entity.TaskInstance{Status: entity.TaskInstanceStatusContinue},
			wantUpdateTaskCalled: true,
			wantUpdateDagIns:     &entity.DagInstance{Status: entity.DagInstanceStatusRunning},
			wantUpdateDagCalled:  true,
		},
		{
			caseDesc:        "continue get task failed",
			giveDagIns:      &entity.DagInstance{Cmd: &entity.Command{Name: entity.CommandNameContinue, TargetTaskInsIDs: []string{"task1"}}},
			wantGetTaskId:   "task1",
			giveTaskErr:     fmt.Errorf("get task failed"),
			wantErr:         fmt.Errorf("get task failed"),
			wantListCallCnt: 1,
		},
		{
			caseDesc:   "no cmd",
			giveDagIns: &entity.DagInstance{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			calledUpdateTask, calledCancel, calledUpdateDag := false, false, false
			listTaskCallCnt := 0
			mStore := &MockStore{}
			mStore.On("ListTaskInstance", mock.Anything).Run(func(args mock.Arguments) {
				listTaskCallCnt++
				if listTaskCallCnt == 1 {
					status := []entity.TaskInstanceStatus{entity.TaskInstanceStatusFailed, entity.TaskInstanceStatusCanceled}
					if tc.giveDagIns.Cmd.Name == entity.CommandNameContinue {
						status = []entity.TaskInstanceStatus{entity.TaskInstanceStatusBlocked}
					}

					assert.Equal(t, &ListTaskInstanceInput{
						IDs:    tc.giveDagIns.Cmd.TargetTaskInsIDs,
						Status: status,
					}, args.Get(0))
				}
			}).Return(tc.giveTask, tc.giveTaskErr)

			mStore.On("UpdateTaskIns", mock.Anything).Run(func(args mock.Arguments) {
				calledUpdateTask = true
				assert.Equal(t, tc.wantUpdateTask, args.Get(0))
			}).Return(tc.giveUpdateTaskErr)

			mStore.On("PatchDagIns", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				calledUpdateDag = true
				assert.Equal(t, tc.wantUpdateDagIns, args.Get(0))
				assert.Equal(t, "Cmd", args.Get(1))
			}).Return(tc.giveUpdateDagInsErr)

			SetStore(mStore)

			mExecutor := &MockExecutor{}
			mExecutor.On("CancelTaskIns", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				calledCancel = true
				assert.Equal(t, tc.wantCancelTaskId, args.Get(0))
			}).Return(tc.giveCancelTaskErr)
			mExecutor.On("Push", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			})
			SetExecutor(mExecutor)

			parser := &DefParser{}
			err := parser.parseCmd(tc.giveDagIns)
			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantListCallCnt, listTaskCallCnt)
			assert.Equal(t, tc.wantUpdateTaskCalled, calledUpdateTask)
			assert.Equal(t, tc.wantCancelCalled, calledCancel)
			assert.Equal(t, tc.wantUpdateDagCalled, calledUpdateDag)
		})
	}
}

func TestDefParser(t *testing.T) {
	pubDagIns := []*entity.DagInstance{
		{},
	}
	pubTasks := []*entity.TaskInstance{
		{Status: entity.TaskInstanceStatusInit},
	}
	wg := sync.WaitGroup{}
	wg.Add(4)
	mStore := &MockStore{}
	mStore.On("ListDagInstance", mock.Anything).Run(func(args mock.Arguments) {
		wg.Done()
	}).Return(pubDagIns, nil)
	mStore.On("ListTaskInstance", mock.Anything).Run(func(args mock.Arguments) {
		wg.Done()
	}).Return(pubTasks, nil)
	SetStore(mStore)

	mKeeper := &MockKeeper{}
	mKeeper.On("WorkerKey").Run(func(args mock.Arguments) {
		wg.Done()
	}).Return("key")
	SetKeeper(mKeeper)

	mExecutor := &MockExecutor{}
	mExecutor.On("Push", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		wg.Done()
	})
	SetExecutor(mExecutor)

	def := NewDefParser(100, time.Second)
	def.Init()
	wg.Wait()
	def.Close()
}
