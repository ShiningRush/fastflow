package mod

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/shiningrush/fastflow/pkg/entity"
	"github.com/shiningrush/fastflow/pkg/entity/run"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type TestParam struct {
	Field1 string `json:"field1"`
}

func TestDefExecutor_CancelTaskIns(t *testing.T) {
	tests := []struct {
		giveExecutor     *DefExecutor
		giveDagInsId     string
		giveTaskInsId    []string
		giveTaskMap      []string
		givePatchErr     error
		wantErr          error
		wantPatchTaskIns *entity.TaskInstance
		wantPatchCalled  bool
		wantEntryCalled  bool
		wantDeleteMap    bool
	}{
		{
			giveDagInsId:  "testDag",
			giveExecutor:  &DefExecutor{},
			giveTaskInsId: []string{"test1"},
			giveTaskMap:   []string{"test1", "test2"},
			wantPatchTaskIns: &entity.TaskInstance{
				BaseInfo: entity.BaseInfo{
					ID: "test1",
				},
				DagInsID: "testDag",
				Status:   entity.TaskInstanceStatusCanceled,
			},
			wantPatchCalled: true,
			wantEntryCalled: true,
			wantDeleteMap:   true,
		},
	}

	for _, tc := range tests {
		for i := range tc.giveTaskMap {
			_, can := context.WithCancel(context.TODO())
			tc.giveExecutor.cancelMap.Store(tc.giveTaskMap[i], can)
		}

		err := tc.giveExecutor.CancelTaskIns(tc.giveTaskInsId)
		assert.Equal(t, tc.wantErr, err)
		for _, id := range tc.giveTaskInsId {
			_, ok := tc.giveExecutor.cancelMap.Load(id)
			assert.Equal(t, tc.wantDeleteMap, !ok)
		}
	}
}

func TestDefExecutor_initWorkerTask(t *testing.T) {
	tests := []struct {
		giveExecutor     *DefExecutor
		giveDagIns       *entity.DagInstance
		giveTask         *entity.TaskInstance
		giveTraceOpt     run.TraceOp
		wantPatchTaskIns []*entity.TaskInstance
		wantTimeout      time.Duration
		wantPatchCnt     int
	}{
		{
			giveExecutor: &DefExecutor{
				workerQueue: make(chan *entity.TaskInstance, 1),
				timeout:     time.Second,
				initQueue:   make(chan *initPayload, 1),
			},
			giveDagIns: &entity.DagInstance{
				ShareData: &entity.ShareData{},
			},
			giveTask:    &entity.TaskInstance{BaseInfo: entity.BaseInfo{ID: "test-id"}},
			wantTimeout: time.Second,
			wantPatchTaskIns: []*entity.TaskInstance{
				{BaseInfo: entity.BaseInfo{ID: "test-id"}, Traces: []entity.TraceInfo{
					{Time: time.Now().Unix(), Message: "test-trace1"},
				}},
				{BaseInfo: entity.BaseInfo{ID: "test-id"}, Traces: []entity.TraceInfo{
					{Time: time.Now().Unix(), Message: "test-trace1"},
					{Time: time.Now().Unix(), Message: "test-trace2"},
				}},
			},
			wantPatchCnt: 2,
		},
		{
			giveTraceOpt: run.TraceOpPersistAfterAction,
			giveExecutor: &DefExecutor{
				workerQueue: make(chan *entity.TaskInstance, 1),
				timeout:     time.Second,
				initQueue:   make(chan *initPayload, 1),
			},
			giveDagIns: &entity.DagInstance{
				ShareData: &entity.ShareData{},
			},
			giveTask: &entity.TaskInstance{
				TimeoutSecs: 60,
			},
			wantTimeout: time.Minute,
		},
	}

	for _, tc := range tests {
		patchCalledCnt := 0
		mStore := &MockStore{}
		mStore.On("PatchTaskIns", mock.Anything).Run(func(args mock.Arguments) {
			assert.Equal(t, tc.wantPatchTaskIns[patchCalledCnt], args.Get(0))
			patchCalledCnt++
		}).Return(nil)
		SetStore(mStore)

		startTime := time.Now()
		tc.giveExecutor.initWorkerTask(tc.giveDagIns, tc.giveTask)
		_, ok := tc.giveExecutor.cancelMap.Load(tc.giveTask.ID)
		assert.True(t, ok)
		assert.Equal(t, tc.giveDagIns.ShareData, tc.giveTask.Context.ShareData())

		// check trace infos
		tc.giveTask.Context.Trace("test-trace1", tc.giveTraceOpt)
		tc.giveTask.Context.Trace("test-trace2", tc.giveTraceOpt)
		if tc.giveTraceOpt == nil {
			assert.Equal(t, tc.giveTask.Traces[len(tc.giveTask.Traces)-2], entity.TraceInfo{Time: time.Now().Unix(), Message: "test-trace1"})
			assert.Equal(t, tc.giveTask.Traces[len(tc.giveTask.Traces)-1], entity.TraceInfo{Time: time.Now().Unix(), Message: "test-trace2"})
		}
		assert.Equal(t, patchCalledCnt, tc.wantPatchCnt)
		dl, ok := tc.giveTask.Context.Context().Deadline()
		assert.True(t, ok)
		assert.True(t, dl.After(startTime.Add(tc.wantTimeout)))
	}
}

func TestDefExecutor_WorkerDo(t *testing.T) {
	calledRun := false
	tests := []struct {
		caseDesc        string
		giveExecutor    *DefExecutor
		giveTaskIns     *entity.TaskInstance
		giveHookErr     error
		isCancel        bool
		wantParams      interface{}
		wantEntryTask   *entity.TaskInstance
		wantEntryCalled bool
	}{
		{
			caseDesc:     "normal",
			giveExecutor: &DefExecutor{},
			giveTaskIns: &entity.TaskInstance{
				ActionName: "test",
				Params: map[string]interface{}{
					"field1": "test_field",
				},
				Status: entity.TaskInstanceStatusInit,
				Patch: func(instance *entity.TaskInstance) error {
					return nil
				},
			},
			wantParams:      &TestParam{Field1: "test_field"},
			wantEntryCalled: true,
			wantEntryTask: &entity.TaskInstance{
				ActionName: "test",
				Params: map[string]interface{}{
					"field1": "test_field",
				},
				Status: entity.TaskInstanceStatusSuccess,
			},
		},
		{
			caseDesc:     "normal without params",
			giveExecutor: &DefExecutor{},
			giveTaskIns: &entity.TaskInstance{
				ActionName: "noParams",
				Status:     entity.TaskInstanceStatusInit,
			},
			wantEntryCalled: true,
			wantEntryTask: &entity.TaskInstance{
				ActionName: "noParams",
				Status:     entity.TaskInstanceStatusSuccess,
			},
		},
		{
			caseDesc:     "no params",
			giveExecutor: &DefExecutor{},
			giveTaskIns: &entity.TaskInstance{
				ActionName: "test",
				Status:     entity.TaskInstanceStatusInit,
			},
			wantEntryCalled: true,
			wantEntryTask: &entity.TaskInstance{
				ActionName: "test",
				Status:     entity.TaskInstanceStatusSuccess,
			},
		},
		{
			caseDesc:     "no params and be canceled",
			giveExecutor: &DefExecutor{},
			isCancel:     true,
			giveTaskIns: &entity.TaskInstance{
				ActionName: "test",
				Status:     entity.TaskInstanceStatusInit,
			},
			wantEntryCalled: true,
			wantEntryTask: &entity.TaskInstance{
				ActionName: "test",
				Status:     entity.TaskInstanceStatusSuccess,
				Reason:     ReasonSuccessAfterCanceled,
			},
		},
		{
			caseDesc:     "action not found",
			giveExecutor: &DefExecutor{},
			giveTaskIns: &entity.TaskInstance{
				ActionName: "no_such_action",
				Status:     entity.TaskInstanceStatusInit,
			},
			wantEntryCalled: true,
			wantEntryTask: &entity.TaskInstance{
				ActionName: "no_such_action",
				Status:     entity.TaskInstanceStatusFailed,
				Reason:     "action not found: no_such_action",
			},
		},
		{
			caseDesc:     "action not found but already canceled",
			giveExecutor: &DefExecutor{},
			isCancel:     true,
			giveTaskIns: &entity.TaskInstance{
				ActionName: "no_such_action",
				Status:     entity.TaskInstanceStatusInit,
			},
			wantEntryCalled: true,
			wantEntryTask: &entity.TaskInstance{
				ActionName: "no_such_action",
				Status:     entity.TaskInstanceStatusCanceled,
				Reason:     "action not found: no_such_action",
			},
		},
		{
			caseDesc:     "unmarshal failed",
			giveExecutor: &DefExecutor{},
			giveTaskIns: &entity.TaskInstance{
				ActionName: "test",
				Params: map[string]interface{}{
					"field1": 1,
				},
				Status: entity.TaskInstanceStatusInit,
			},
			wantEntryCalled: true,
			wantEntryTask: &entity.TaskInstance{
				ActionName: "test",
				Params: map[string]interface{}{
					"field1": 1,
				},
				Status: entity.TaskInstanceStatusSuccess,
			},
		},
		{
			caseDesc:     "no status",
			giveExecutor: &DefExecutor{},
			giveTaskIns:  &entity.TaskInstance{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			testAct := &run.MockAction{}
			testAct.On("Name", mock.Anything, mock.Anything).Return("test")
			testAct.On("Run", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				calledRun = true
				if tc.wantParams != nil {
					assert.Equal(t, tc.wantParams, args.Get(1))
				}
			}).Return(nil)
			testAct.On("ParameterNew", mock.Anything, mock.Anything).Return(&TestParam{})
			testAct.On("RunBefore", mock.Anything, mock.Anything).Return(nil)
			testAct.On("RunAfter", mock.Anything, mock.Anything).Return(nil)

			noParamAct := &run.MockAction{}
			noParamAct.On("Run", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				calledRun = true
				if tc.wantParams != nil {
					assert.Equal(t, tc.wantParams, args.Get(1))
				}
			}).Return(nil)
			noParamAct.On("Name", mock.Anything, mock.Anything).Return("noParams")
			noParamAct.On("ParameterNew", mock.Anything, mock.Anything).Return(nil)
			noParamAct.On("RunBefore", mock.Anything, mock.Anything).Return(nil)
			noParamAct.On("RunAfter", mock.Anything, mock.Anything).Return(nil)

			calledEntry := false
			mParser := &MockParser{}
			mParser.On("EntryTaskIns", mock.Anything).Run(func(args mock.Arguments) {
				calledEntry = true
				args.Get(0).(*entity.TaskInstance).Patch = nil
				assert.Equal(t, tc.wantEntryTask, args.Get(0))
			})
			SetParser(mParser)

			ActionMap = map[string]run.Action{
				"test":     testAct,
				"noParams": noParamAct,
			}

			tc.giveTaskIns.InitialDep(nil, func(instance *entity.TaskInstance) error {
				return nil
			})
			if !tc.isCancel {
				tc.giveExecutor.cancelMap.Store(tc.giveTaskIns.ID, nil)
			}
			tc.giveExecutor.workerDo(tc.giveTaskIns)
			assert.True(t, calledRun)
			assert.Equal(t, tc.wantEntryCalled, calledEntry)
		})
	}
}

func TestDefExecutor(t *testing.T) {
	mStore := &MockStore{}
	calledUpdateTask, calledEntryTaskIns := false, false
	mStore.On("PatchTaskIns", mock.Anything).Run(func(args mock.Arguments) {
		calledUpdateTask = true
	}).Return(nil)
	SetStore(mStore)

	mParser := &MockParser{}
	mParser.On("EntryTaskIns", mock.Anything).Run(func(args mock.Arguments) {
		calledEntryTaskIns = true
	})
	SetParser(mParser)

	e := NewDefExecutor(time.Minute, 100)
	e.Init()
	//dagIns := nil
	e.Push(&entity.DagInstance{
		ShareData: &entity.ShareData{},
	}, &entity.TaskInstance{
		Status: entity.TaskInstanceStatusInit,
	})
	e.Close()
	assert.True(t, calledUpdateTask)
	assert.True(t, calledEntryTaskIns)
}

func TestDefExecutor_getFromTaskInstance(t *testing.T) {
	type T struct {
		Bool   bool
		Int    int
		Float  float32
		String string
		Int32  int32
		Uint8  uint8
	}

	type args struct {
		taskIns *entity.TaskInstance
		params  interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "string type",
			args: args{
				taskIns: &entity.TaskInstance{Params: map[string]interface{}{
					"Bool":   "true",
					"Int":    "10",
					"Float":  "1.3",
					"String": "sas",
					"Int32":  "1",
					"Uint8":  "4",
				}},
				params: &T{},
			},
			want: &T{
				Bool:   true,
				Int:    10,
				Float:  1.3,
				String: "sas",
				Int32:  1,
				Uint8:  4,
			},
			wantErr: assert.NoError,
		},
		{
			name: "origin type",
			args: args{
				taskIns: &entity.TaskInstance{Params: map[string]interface{}{
					"Bool":   true,
					"Int":    10,
					"Float":  1.3,
					"String": "sas",
					"Int32":  1,
					"Uint8":  4,
				}},
				params: &T{},
			},
			want: &T{
				Bool:   true,
				Int:    10,
				Float:  1.3,
				String: "sas",
				Int32:  1,
				Uint8:  4,
			},
			wantErr: assert.NoError,
		},
		{
			name: "float str to int",
			args: args{
				taskIns: &entity.TaskInstance{Params: map[string]interface{}{
					"Int": "10.1",
				}},
				params: &T{},
			},
			want:    &T{},
			wantErr: assert.Error,
		},
		{
			name: "out of range",
			args: args{
				taskIns: &entity.TaskInstance{Params: map[string]interface{}{
					"Uint8": "454545",
				}},
				params: &T{},
			},
			want:    &T{},
			wantErr: assert.Error,
		},
		{
			name: "-1 to uint",
			args: args{
				taskIns: &entity.TaskInstance{Params: map[string]interface{}{
					"Uint8": "-1",
				}},
				params: &T{},
			},
			want:    &T{},
			wantErr: assert.Error,
		},
		{
			name: "1 to bool",
			args: args{
				taskIns: &entity.TaskInstance{Params: map[string]interface{}{
					"Bool": "1",
				}},
				params: &T{},
			},
			want: &T{
				Bool: true,
			},
			wantErr: assert.NoError,
		},
		{
			name: "0 to bool",
			args: args{
				taskIns: &entity.TaskInstance{Params: map[string]interface{}{
					"Bool": "0",
				}},
				params: &T{},
			},
			want:    &T{},
			wantErr: assert.NoError,
		},
		{
			name: "str to int",
			args: args{
				taskIns: &entity.TaskInstance{Params: map[string]interface{}{
					"Bool": "qqq",
				}},
				params: &T{},
			},
			want:    &T{},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &DefExecutor{}
			tt.wantErr(t, e.getFromTaskInstance(tt.args.taskIns, tt.args.params), fmt.Sprintf("getFromTaskInstance(%v, %v)", tt.args.taskIns, tt.args.params))
			if !reflect.DeepEqual(tt.args.params, tt.want) {
				t.Errorf("tt.args.params != tt.want,%+v,%+v", tt.args.params, tt.want)
			}
		})
	}
}
