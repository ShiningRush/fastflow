package mod

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shiningrush/fastflow/pkg/entity"
	"github.com/shiningrush/fastflow/pkg/entity/run"
	"github.com/shiningrush/fastflow/pkg/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/yaml.v3"
)

type TestParam struct {
	Field1 string `json:"field1"`
}

func NewTestParam() interface{} {
	return &TestParam{}
}

type TestParamInt struct {
	Field1 int `json:"field1"`
}

func NewTestParamInt() interface{} {
	return &TestParamInt{}
}

type TestRenderParam struct {
	Field1    string
	Shk       string
	Shi       int
	Shf       float64
	Shb       bool
	Sdk       string
	Sdi       int
	Sdf       float64
	Sdb       bool
	SnakeCase string `json:"snake_case"`
	CamelCase string
	KebabCase string
}

func NewTestRenderParam() interface{} {
	return &TestRenderParam{}
}

func TestRenderParamYaml(t *testing.T) {
	bytes, err := yaml.Marshal(NewTestRenderParam())
	assert.NoError(t, err)
	t.Log(string(bytes))
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
	relatedDagInstance := &entity.DagInstance{
		Vars: map[string]entity.DagInstanceVar{
			"shk": {Value: "shv"},
			"shi": {Value: "100"},
			"shf": {Value: "3.14159"},
			"shb": {Value: "true"},
		},
		ShareData: &entity.ShareData{Dict: map[string]string{
			"sdk":        "sdv",
			"sdi":        "123",
			"sdf":        "1.2345",
			"sdb":        "true",
			"snake_case": "snake_case_value",
			"camelCase":  "camelCaseValue",
			"kebab-case": "kebab-case-value",
		}},
	}
	tests := []struct {
		caseDesc         string
		giveExecutor     *DefExecutor
		giveTaskIns      *entity.TaskInstance
		giveHookErr      error
		giveParameterNew func() interface{}
		isCancel         bool
		wantParams       interface{}
		wantEntryTask    *entity.TaskInstance
		wantEntryCalled  bool
		wantCalledRun    bool
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
			giveParameterNew: NewTestParam,
			wantParams:       &TestParam{Field1: "test_field"},
			wantEntryCalled:  true,
			wantCalledRun:    true,
			wantEntryTask: &entity.TaskInstance{
				ActionName: "test",
				Params: map[string]interface{}{
					"field1": "test_field",
				},
				Status: entity.TaskInstanceStatusSuccess,
			},
		},
		{
			caseDesc: "param render",
			giveExecutor: &DefExecutor{
				paramRender: render.NewTplRender(
					render.NewCachedTplProvider(1000,
						render.NewParseTplProvider(render.WithMissKeyStrategy(render.MissKeyStrategyError)))),
			},
			giveTaskIns: &entity.TaskInstance{
				ActionName: "test",
				Params: map[string]interface{}{
					"field1":     "test_field",
					"shk":        "{{.vars.shk.Value}}",
					"shi":        "{{.vars.shi.Value}}",
					"shf":        "{{.vars.shf.Value}}",
					"shb":        "{{.vars.shb.Value}}",
					"sdk":        "{{.shareData.sdk}}",
					"sdi":        "{{.shareData.sdi}}",
					"sdf":        "{{.shareData.sdf}}",
					"sdb":        "{{.shareData.sdb}}",
					"snake_case": "{{.shareData.snake_case}}",
					"camelCase":  "{{.shareData.camelCase}}",
				},
				Status: entity.TaskInstanceStatusInit,
				Patch: func(instance *entity.TaskInstance) error {
					return nil
				},
				RelatedDagInstance: relatedDagInstance,
			},
			giveParameterNew: NewTestRenderParam,
			wantParams: &TestRenderParam{
				Field1:    "test_field",
				Shk:       "shv",
				Shi:       100,
				Shf:       3.14159,
				Shb:       true,
				Sdk:       "sdv",
				Sdi:       123,
				Sdf:       1.2345,
				Sdb:       true,
				SnakeCase: "snake_case_value",
				CamelCase: "camelCaseValue",
			},
			wantEntryCalled: true,
			wantCalledRun:   true,
			wantEntryTask: &entity.TaskInstance{
				ActionName: "test",
				Params: map[string]interface{}{
					"field1":     "test_field",
					"shk":        "shv",
					"shi":        "100",
					"shf":        "3.14159",
					"shb":        "true",
					"sdk":        "sdv",
					"sdi":        "123",
					"sdf":        "1.2345",
					"sdb":        "true",
					"snake_case": "snake_case_value",
					"camelCase":  "camelCaseValue",
				},
				Status:             entity.TaskInstanceStatusSuccess,
				RelatedDagInstance: relatedDagInstance,
			},
		},
		{
			caseDesc: "param render failed",
			giveExecutor: &DefExecutor{
				paramRender: render.NewTplRender(
					render.NewCachedTplProvider(1000,
						render.NewParseTplProvider(render.WithMissKeyStrategy(render.MissKeyStrategyError)))),
			},
			giveTaskIns: &entity.TaskInstance{
				ActionName: "test",
				Params: map[string]interface{}{
					"a": map[string]interface{}{
						"b": "{{.a.b.c}}",
					},
				},
				Status: entity.TaskInstanceStatusInit,
				Patch: func(instance *entity.TaskInstance) error {
					return nil
				},
				RelatedDagInstance: relatedDagInstance,
			},
			giveParameterNew: NewTestRenderParam,
			wantParams:       &TestRenderParam{},
			wantEntryCalled:  true,
			wantCalledRun:    false,
			wantEntryTask: &entity.TaskInstance{
				ActionName: "test",
				Params: map[string]interface{}{
					"a": map[string]interface{}{
						"b": "{{.a.b.c}}",
					},
				},
				Status:             entity.TaskInstanceStatusFailed,
				Reason:             "get task params from task instance failed: renderParams failed: execute tpl failed: template: {{.a.b.c}}:1:4: executing \"{{.a.b.c}}\" at <.a.b.c>: map has no entry for key \"a\"",
				RelatedDagInstance: relatedDagInstance,
			},
		},
		{
			caseDesc:     "normal without params",
			giveExecutor: &DefExecutor{},
			giveTaskIns: &entity.TaskInstance{
				ActionName: "noParams",
				Status:     entity.TaskInstanceStatusInit,
			},
			giveParameterNew: NewTestParam,
			wantEntryCalled:  true,
			wantCalledRun:    true,
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
			giveParameterNew: NewTestParam,
			wantEntryCalled:  true,
			wantCalledRun:    true,
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
			giveParameterNew: NewTestParam,
			wantEntryCalled:  true,
			wantCalledRun:    true,
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
			giveParameterNew: NewTestParam,
			wantEntryCalled:  true,
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
			giveParameterNew: NewTestParam,
			wantEntryCalled:  true,
			wantEntryTask: &entity.TaskInstance{
				ActionName: "no_such_action",
				Status:     entity.TaskInstanceStatusCanceled,
				Reason:     "action not found: no_such_action",
			},
		},
		{
			caseDesc:     "convert failed",
			giveExecutor: &DefExecutor{},
			giveTaskIns: &entity.TaskInstance{
				ActionName: "test",
				Params: map[string]interface{}{
					"field1": "qqq",
				},
				Status: entity.TaskInstanceStatusInit,
			},
			giveParameterNew: NewTestParamInt,
			wantEntryCalled:  true,
			wantEntryTask: &entity.TaskInstance{
				ActionName: "test",
				Params: map[string]interface{}{
					"field1": "qqq",
				},
				Status: entity.TaskInstanceStatusFailed,
				Reason: "get task params from task instance failed: 1 error(s) decoding:\n\n" +
					"* cannot parse 'field1' as int: strconv.ParseInt: parsing \"qqq\": invalid syntax",
			},
		},
		{
			caseDesc:         "no status",
			giveExecutor:     &DefExecutor{},
			giveTaskIns:      &entity.TaskInstance{},
			giveParameterNew: NewTestParam,
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			calledRun := false
			testAct := &run.MockAction{}
			testAct.On("Name", mock.Anything, mock.Anything).Return("test")
			testAct.On("Run", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				calledRun = true
				if tc.wantParams != nil {
					assert.Equal(t, tc.wantParams, args.Get(1))
				}
			}).Return(nil)
			testAct.On("ParameterNew", mock.Anything, mock.Anything).Return(tc.giveParameterNew())
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
			}, tc.giveTaskIns.RelatedDagInstance)
			if !tc.isCancel {
				tc.giveExecutor.cancelMap.Store(tc.giveTaskIns.ID, nil)
			}
			tc.giveExecutor.workerDo(tc.giveTaskIns)
			assert.Equal(t, tc.wantCalledRun, calledRun)
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
			assert.Equal(t, tt.want, tt.args.params)
		})
	}
}

func TestDefExecutor_renderParams(t *testing.T) {
	paramRender := render.NewTplRender(
		render.NewCachedTplProvider(1000,
			render.NewParseTplProvider(render.WithMissKeyStrategy(render.MissKeyStrategyError))))

	type fields struct {
		paramRender render.Render
	}
	type args struct {
		taskIns *entity.TaskInstance
	}
	dagIns := &entity.DagInstance{
		Vars: entity.DagInstanceVars{
			"ka": entity.DagInstanceVar{
				Value: "va",
			},
			"kb": entity.DagInstanceVar{
				Value: "vb",
			},
		},
		ShareData: &entity.ShareData{
			Dict: map[string]string{
				"ska":   "skb",
				"skint": "1",
			},
		},
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
		want    map[string]interface{}
	}{
		{
			name: "success",
			fields: fields{
				paramRender: paramRender,
			},
			args: args{
				taskIns: &entity.TaskInstance{
					RelatedDagInstance: dagIns,
					Params: map[string]interface{}{
						"a": "{{.shareData.ska}}",
						"b": "{{.vars.ka.Value}}",
						"c": map[string]interface{}{
							"d": map[string]interface{}{},
						},
					},
				},
			},
			want: map[string]interface{}{
				"a": "skb",
				"b": "va",
				"c": map[string]interface{}{
					"d": map[string]interface{}{},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "map has no entry for key \"as\"",
			fields: fields{
				paramRender: paramRender,
			},
			args: args{
				taskIns: &entity.TaskInstance{
					RelatedDagInstance: dagIns,
					Params: map[string]interface{}{
						"c": map[string]interface{}{
							"d": map[string]interface{}{
								"e": "{{.as.as.as}}",
							},
						},
					},
				},
			},
			wantErr: assert.Error,
			want: map[string]interface{}{
				"c": map[string]interface{}{
					"d": map[string]interface{}{
						"e": "{{.as.as.as}}",
					},
				}},
		},
		{
			name: "function \"hhh\" not defined",
			fields: fields{
				paramRender: paramRender,
			},
			args: args{
				taskIns: &entity.TaskInstance{
					RelatedDagInstance: dagIns,
					Params: map[string]interface{}{
						"c": map[string]interface{}{
							"d": map[string]interface{}{
								"e": "{{hhh}}",
							},
						},
					},
				},
			},
			wantErr: assert.Error,
			want: map[string]interface{}{
				"c": map[string]interface{}{
					"d": map[string]interface{}{
						"e": "{{hhh}}",
					},
				}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &DefExecutor{
				paramRender: tt.fields.paramRender,
			}
			err := e.renderParams(tt.args.taskIns)
			tt.wantErr(t, err, fmt.Sprintf("renderParams(%v)", tt.args.taskIns))
			assert.Equal(t, tt.want, tt.args.taskIns.Params)
		})
	}
}
