package mod

import (
	"fmt"
	"testing"
	"time"

	"github.com/shiningrush/fastflow/pkg/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDefCommander_RunDag(t *testing.T) {
	tests := []struct {
		caseDesc      string
		giveDagId     string
		giveVars      map[string]string
		giveDag       *entity.Dag
		giveGetErr    error
		giveCreateErr error
		wantErr       error
		wantDagIns    *entity.DagInstance
	}{
		{
			caseDesc:  "normal",
			giveDagId: "test-dag",
			giveDag: &entity.Dag{
				BaseInfo: entity.BaseInfo{
					ID: "test-dag",
				},
				Status: entity.DagStatusNormal,
			},
			wantDagIns: &entity.DagInstance{
				DagID:     "test-dag",
				Vars:      entity.DagInstanceVars{},
				Trigger:   entity.TriggerManually,
				Status:    entity.DagInstanceStatusInit,
				ShareData: &entity.ShareData{},
			},
		},
		{
			caseDesc:   "get failed",
			giveDagId:  "test-dag",
			giveGetErr: fmt.Errorf("get failed"),
			giveDag: &entity.Dag{
				BaseInfo: entity.BaseInfo{
					ID: "test-dag",
				},
				Status: entity.DagStatusNormal,
			},
			wantErr: fmt.Errorf("get failed"),
		},
		{
			caseDesc:  "run failed",
			giveDagId: "test-dag",
			giveDag: &entity.Dag{
				Status: entity.DagStatusStopped,
			},
			wantErr: fmt.Errorf("you cannot run a stopeed dag"),
		},
		{
			caseDesc:      "created failed",
			giveDagId:     "test-dag",
			giveCreateErr: fmt.Errorf("created failed"),
			giveDag: &entity.Dag{
				Status: entity.DagStatusNormal,
			},
			wantErr: fmt.Errorf("created failed"),
			wantDagIns: &entity.DagInstance{
				Vars:      entity.DagInstanceVars{},
				Trigger:   entity.TriggerManually,
				Status:    entity.DagInstanceStatusInit,
				ShareData: &entity.ShareData{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			isCalled := false
			mStore := &MockStore{}
			mStore.On("GetDag", mock.Anything).Run(func(args mock.Arguments) {
				isCalled = true
				assert.Equal(t, tc.giveDagId, args.Get(0))
			}).Return(tc.giveDag, tc.giveGetErr)
			mStore.On("CreateDagIns", mock.Anything).Run(func(args mock.Arguments) {
				assert.Equal(t, tc.wantDagIns, args.Get(0))
			}).Return(tc.giveCreateErr)
			SetStore(mStore)

			c := &DefCommander{}
			dagIns, err := c.RunDag(tc.giveDagId, tc.giveVars)
			assert.Equal(t, tc.wantErr, err)
			if err == nil {
				assert.Equal(t, tc.wantDagIns, dagIns)
			}
			assert.True(t, isCalled)
		})
	}
}

func TestDefCommander_OpDagIns(t *testing.T) {
	tests := []struct {
		caseDesc      string
		giveDagInsID  string
		giveListRet   []*entity.TaskInstance
		giveListErr   error
		giveOp        string
		wantErr       error
		wantListInput []*ListTaskInstanceInput
		wantTaskInsId string
	}{
		{
			caseDesc:     "retry-normal",
			giveDagInsID: "dagInsId",
			wantListInput: []*ListTaskInstanceInput{
				{
					DagInsID: "dagInsId",
					Status:   []entity.TaskInstanceStatus{entity.TaskInstanceStatusFailed, entity.TaskInstanceStatusCanceled},
				},
				{
					IDs: []string{"testTaskId", "testTaskId2"},
				},
			},
			giveListRet: []*entity.TaskInstance{
				{
					BaseInfo: entity.BaseInfo{
						ID: "testTaskId",
					},
					Status: entity.TaskInstanceStatusFailed,
				},
				{
					BaseInfo: entity.BaseInfo{
						ID: "testTaskId2",
					},
					Status: entity.TaskInstanceStatusFailed,
				},
			},
			wantTaskInsId: "testTaskId",
		},
		{
			caseDesc:     "continue-normal",
			giveDagInsID: "dagInsId",
			giveOp:       entity.CommandNameContinue,
			wantListInput: []*ListTaskInstanceInput{
				{
					DagInsID: "dagInsId",
					Status:   []entity.TaskInstanceStatus{entity.TaskInstanceStatusBlocked},
				},
				{
					IDs: []string{"testTaskId", "testTaskId2"},
				},
			},
			giveListRet: []*entity.TaskInstance{
				{
					BaseInfo: entity.BaseInfo{
						ID: "testTaskId",
					},
					Status: entity.TaskInstanceStatusBlocked,
				},
				{
					BaseInfo: entity.BaseInfo{
						ID: "testTaskId2",
					},
					Status: entity.TaskInstanceStatusFailed,
				},
			},
			wantTaskInsId: "testTaskId",
		},
		{
			caseDesc:     "list failed",
			giveDagInsID: "dagInsId",
			wantListInput: []*ListTaskInstanceInput{
				{
					DagInsID: "dagInsId",
					Status:   []entity.TaskInstanceStatus{entity.TaskInstanceStatusFailed, entity.TaskInstanceStatusCanceled},
				},
			},
			giveListErr: fmt.Errorf("list failed"),
			wantErr:     fmt.Errorf("list failed"),
		},
		{
			caseDesc:     "no failed and canceled task ins",
			giveDagInsID: "dagInsId",
			wantListInput: []*ListTaskInstanceInput{
				{
					DagInsID: "dagInsId",
					Status:   []entity.TaskInstanceStatus{entity.TaskInstanceStatusFailed, entity.TaskInstanceStatusCanceled},
				},
			},
			giveListRet: []*entity.TaskInstance{},
			wantErr:     fmt.Errorf("no [failed canceled] task instance"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			isCalled := false

			listCnt := 0
			mStore := &MockStore{}
			mStore.On("ListTaskInstance", mock.Anything).Run(func(args mock.Arguments) {
				isCalled = true
				assert.Equal(t, tc.wantListInput[listCnt], args.Get(0))
				listCnt++
			}).Return(tc.giveListRet, tc.giveListErr)
			mStore.On("GetDagInstance", mock.Anything).Return(&entity.DagInstance{
				Status: entity.DagInstanceStatusFailed,
			}, nil)
			mStore.On("PatchDagIns", mock.Anything).Run(func(args mock.Arguments) {
			}).Return(nil)
			SetStore(mStore)

			mKeep := &MockKeeper{}
			mKeep.On("IsAlive", mock.Anything).Return(true, nil)
			mKeep.On("AliveNodes").Run(func(args mock.Arguments) {
			}).Return([]string{"alive-node-1"}, nil)
			SetKeeper(mKeep)

			c := &DefCommander{}
			var err error
			if tc.giveOp == entity.CommandNameContinue {
				err = c.ContinueDagIns(tc.giveDagInsID)
			} else {
				err = c.RetryDagIns(tc.giveDagInsID)
			}
			assert.Equal(t, tc.wantErr, err)
			assert.True(t, isCalled)
		})
	}
}

func TestDefCommander_OpTask(t *testing.T) {
	tests := []struct {
		caseDesc             string
		giveTaskInsID        []string
		giveIsAlive          bool
		giveAliveNodes       []string
		giveAliveNodesErr    error
		giveOp               string
		wantErr              error
		wantUpdateDagIns     *entity.DagInstance
		wantAliveNodesCalled bool
	}{
		{
			caseDesc:      "normal",
			giveTaskInsID: []string{"test task"},
			giveIsAlive:   true,
			wantUpdateDagIns: &entity.DagInstance{
				Cmd: &entity.Command{
					Name:             entity.CommandNameRetry,
					TargetTaskInsIDs: []string{"test task"},
				},
			},
		},
		{
			caseDesc:      "normal-continue",
			giveTaskInsID: []string{"test task"},
			giveIsAlive:   true,
			giveOp:        entity.CommandNameContinue,
			wantUpdateDagIns: &entity.DagInstance{
				Cmd: &entity.Command{
					Name:             entity.CommandNameContinue,
					TargetTaskInsIDs: []string{"test task"},
				},
			},
		},
		{
			caseDesc:       "unhealthy worker",
			giveTaskInsID:  []string{"test task"},
			giveIsAlive:    false,
			giveAliveNodes: []string{"1", "2"},
			wantUpdateDagIns: &entity.DagInstance{
				Worker: "2",
				Cmd: &entity.Command{
					Name:             entity.CommandNameRetry,
					TargetTaskInsIDs: []string{"test task"},
				},
			},
			wantAliveNodesCalled: true,
		},
		{
			caseDesc:             "get alive nodes failed",
			giveTaskInsID:        []string{"test task"},
			giveIsAlive:          false,
			giveAliveNodes:       []string{"1", "2"},
			giveAliveNodesErr:    fmt.Errorf("get failed"),
			wantAliveNodesCalled: true,
			wantErr:              fmt.Errorf("get failed"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			mStore := &MockStore{}
			mStore.On("ListTaskInstance", mock.Anything).Run(func(args mock.Arguments) {
				assert.Equal(t, &ListTaskInstanceInput{
					IDs: tc.giveTaskInsID,
				}, args.Get(0))
			}).Return([]*entity.TaskInstance{
				{},
			}, nil)
			mStore.On("GetDagInstance", mock.Anything).Return(&entity.DagInstance{
				Status: entity.DagInstanceStatusFailed,
			}, nil)
			mStore.On("PatchDagIns", mock.Anything).Run(func(args mock.Arguments) {
				assert.Equal(t, tc.wantUpdateDagIns, args.Get(0))
			}).Return(nil)
			SetStore(mStore)

			isCalled := false
			mKeep := &MockKeeper{}
			mKeep.On("IsAlive", mock.Anything).Return(tc.giveIsAlive, nil)
			mKeep.On("AliveNodes").Run(func(args mock.Arguments) {
				isCalled = true
			}).Return(tc.giveAliveNodes, tc.giveAliveNodesErr)
			SetKeeper(mKeep)

			c := &DefCommander{}
			var err error
			if tc.giveOp == entity.CommandNameContinue {
				err = c.ContinueTask(tc.giveTaskInsID)
			} else {
				err = c.RetryTask(tc.giveTaskInsID)
			}
			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantAliveNodesCalled, isCalled)
		})
	}
}

func TestDefCommander_CancelTask(t *testing.T) {

}

func TestDefCommander_initOption(t *testing.T) {
	tests := []struct {
		caseDesc   string
		giveSetter []CommandOptSetter
		wantOpt    CommandOption
	}{
		{
			caseDesc:   "default value",
			giveSetter: []CommandOptSetter{},
			wantOpt: CommandOption{
				syncTimeout:  5 * time.Second,
				syncInterval: 500 * time.Millisecond,
			},
		},
		{
			caseDesc: "specified value",
			giveSetter: []CommandOptSetter{
				CommSync(),
				CommSyncTimeout(time.Second),
			},
			wantOpt: CommandOption{
				isSync:       true,
				syncTimeout:  time.Second,
				syncInterval: 500 * time.Millisecond,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			opt := initOption(tc.giveSetter)
			assert.Equal(t, tc.wantOpt, opt)
		})
	}
}

func TestDefCommander_executeCommand(t *testing.T) {
	tests := []struct {
		caseDesc         string
		giveTaskID       []string
		wantGetTaskID    []string
		giveTaskIns      []*entity.TaskInstance
		giveTaskInsErr   error
		wantDagID        string
		giveDagIns       *entity.DagInstance
		giveDagInsErr    error
		wantWorkerKey    string
		giveAlive        bool
		giveAliveErr     error
		givePerformErr   error
		wantUpdateDagIns *entity.DagInstance
		giveUpdateErr    error
		giveSync         bool
		wantEnsureCalled bool
		wantErr          error
	}{
		{
			caseDesc:      "normal sync",
			giveTaskID:    []string{"test-task"},
			wantGetTaskID: []string{"test-task"},
			giveTaskIns: []*entity.TaskInstance{
				{DagInsID: "dag-id"},
			},
			wantDagID:        "dag-id",
			giveDagIns:       &entity.DagInstance{Worker: "worker"},
			wantWorkerKey:    "worker",
			giveAlive:        true,
			wantUpdateDagIns: &entity.DagInstance{Worker: "test"},
			giveSync:         true,
			wantEnsureCalled: true,
		},
		{
			caseDesc:      "normal async",
			giveTaskID:    []string{"test-task"},
			wantGetTaskID: []string{"test-task"},
			giveTaskIns: []*entity.TaskInstance{
				{DagInsID: "dag-id"},
			},
			wantDagID:        "dag-id",
			giveDagIns:       &entity.DagInstance{Worker: "worker"},
			wantWorkerKey:    "worker",
			giveAlive:        true,
			wantUpdateDagIns: &entity.DagInstance{Worker: "test"},
		},
		{
			caseDesc:       "get task ins failed",
			giveTaskID:     []string{"test-task"},
			wantGetTaskID:  []string{"test-task"},
			giveTaskInsErr: fmt.Errorf("get failed"),
			wantErr:        fmt.Errorf("get failed"),
		},
		{
			caseDesc:      "get dag ins failed",
			giveTaskID:    []string{"test-task"},
			wantGetTaskID: []string{"test-task"},
			giveTaskIns: []*entity.TaskInstance{
				{DagInsID: "dag-id"},
			},
			wantDagID:     "dag-id",
			giveDagInsErr: fmt.Errorf("get failed"),
			wantErr:       fmt.Errorf("get failed"),
		},
		{
			caseDesc:      "get dag ins failed",
			giveTaskID:    []string{"test-task"},
			wantGetTaskID: []string{"test-task"},
			giveTaskIns: []*entity.TaskInstance{
				{DagInsID: "dag-id"},
			},
			wantDagID:     "dag-id",
			giveDagInsErr: fmt.Errorf("get failed"),
			wantErr:       fmt.Errorf("get failed"),
		},
		{
			caseDesc:      "get alive failed",
			giveTaskID:    []string{"test-task"},
			wantGetTaskID: []string{"test-task"},
			giveTaskIns: []*entity.TaskInstance{
				{DagInsID: "dag-id"},
			},
			wantDagID:     "dag-id",
			giveDagIns:    &entity.DagInstance{Worker: "worker"},
			wantWorkerKey: "worker",
			giveAliveErr:  fmt.Errorf("get failed"),
			wantErr:       fmt.Errorf("get failed"),
		},
		{
			caseDesc:      "update dag instance failed",
			giveTaskID:    []string{"test-task"},
			wantGetTaskID: []string{"test-task"},
			giveTaskIns: []*entity.TaskInstance{
				{DagInsID: "dag-id"},
			},
			wantDagID:        "dag-id",
			giveDagIns:       &entity.DagInstance{Worker: "worker"},
			wantWorkerKey:    "worker",
			giveAlive:        true,
			wantUpdateDagIns: &entity.DagInstance{Worker: "test"},
			giveUpdateErr:    fmt.Errorf("update failed"),
			wantErr:          fmt.Errorf("update failed"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			mStore := &MockStore{}
			mStore.On("ListTaskInstance", mock.Anything).Run(func(args mock.Arguments) {
				assert.Equal(t, &ListTaskInstanceInput{
					IDs: tc.giveTaskID,
				}, args.Get(0))
			}).Return(tc.giveTaskIns, tc.giveTaskInsErr)
			dagGetCnt := 0
			mStore.On("GetDagInstance", mock.Anything).Run(func(args mock.Arguments) {
				assert.Equal(t, tc.wantDagID, args.Get(0))
				dagGetCnt++
			}).Return(tc.giveDagIns, tc.giveDagInsErr)
			mStore.On("PatchDagIns", mock.Anything).Run(func(args mock.Arguments) {
				assert.Equal(t, tc.wantUpdateDagIns, args.Get(0))
			}).Return(tc.giveUpdateErr)
			SetStore(mStore)

			mKeep := &MockKeeper{}
			mKeep.On("IsAlive", mock.Anything).Run(func(args mock.Arguments) {
				assert.Equal(t, tc.wantWorkerKey, args.Get(0))
			}).Return(tc.giveAlive, tc.giveAliveErr)
			SetKeeper(mKeep)

			perform := func(dagIns *entity.DagInstance, isWorkerAlive bool) error {
				assert.Equal(t, tc.giveDagIns, dagIns)
				assert.Equal(t, tc.giveAlive, isWorkerAlive)
				dagIns.DagID = "p-dag"
				dagIns.Worker = "test"
				return tc.givePerformErr
			}
			opt := CommandOption{isSync: tc.giveSync, syncTimeout: time.Minute, syncInterval: time.Millisecond}
			err := executeCommand(tc.giveTaskID, perform, opt)
			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantEnsureCalled, dagGetCnt > 1)
		})
	}
}

func TestDefCommander_ensureCmdExecuted(t *testing.T) {
	tests := []struct {
		caseDesc                string
		giveDagInsId            string
		giveOpt                 CommandOption
		removeCmdAfterCalledCnt int
		giveGetErr              error
		giveGetElapsed          time.Duration
		wantCalledCnt           int
		wantErr                 error
		wantDagInsId            string
	}{
		{
			caseDesc:     "get elapsed time less than interval",
			giveDagInsId: "test id",
			giveOpt: CommandOption{
				syncTimeout:  1 * time.Second,
				syncInterval: 100 * time.Millisecond,
			},
			removeCmdAfterCalledCnt: 9,
			wantCalledCnt:           9,
			wantDagInsId:            "test id",
		},
		{
			caseDesc:     "get timeout",
			giveDagInsId: "test id",
			giveOpt: CommandOption{
				syncTimeout:  1 * time.Second,
				syncInterval: 95 * time.Millisecond,
			},
			wantCalledCnt: 10,
			wantDagInsId:  "test id",
			wantErr:       fmt.Errorf("watch command executing timeout"),
		},
		{
			caseDesc:     "get elapsed time more than interval",
			giveDagInsId: "test id",
			giveOpt: CommandOption{
				syncTimeout:  1 * time.Second,
				syncInterval: 100 * time.Millisecond,
			},
			giveGetElapsed:          200 * time.Millisecond,
			removeCmdAfterCalledCnt: 4,
			wantCalledCnt:           4,
			wantDagInsId:            "test id",
		},
		{
			caseDesc:     "get failed",
			giveDagInsId: "test id",
			giveOpt: CommandOption{
				syncTimeout:  1 * time.Second,
				syncInterval: 100 * time.Millisecond,
			},
			giveGetErr:    fmt.Errorf("get failed"),
			wantCalledCnt: 1,
			wantDagInsId:  "test id",
			wantErr:       fmt.Errorf("get failed"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			calledCnt := 0
			mStore := &MockStore{}
			mStore.On("GetDagInstance", mock.Anything).Run(func(args mock.Arguments) {
				assert.Equal(t, tc.wantDagInsId, args.Get(0))
				if tc.giveGetElapsed > 0 {
					time.Sleep(tc.giveGetElapsed)
				}
				calledCnt++
			}).Return(func(string) *entity.DagInstance {
				if tc.removeCmdAfterCalledCnt > 0 && calledCnt == tc.removeCmdAfterCalledCnt {
					return &entity.DagInstance{}
				}
				return &entity.DagInstance{Cmd: &entity.Command{}}
			}, tc.giveGetErr)
			SetStore(mStore)

			err := ensureCmdExecuted(tc.giveDagInsId, tc.giveOpt)
			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantCalledCnt, calledCnt)
		})
	}
}
