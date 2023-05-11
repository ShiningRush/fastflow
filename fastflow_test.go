package fastflow

import (
	"fmt"
	"testing"
	"time"

	"github.com/shiningrush/fastflow/pkg/entity"
	"github.com/shiningrush/fastflow/pkg/mod"
	"github.com/shiningrush/fastflow/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/yaml.v3"
)

func Test_checkOption(t *testing.T) {
	tests := []struct {
		giveOpt *InitialOption
		wantErr error
		wantOpt *InitialOption
	}{
		{
			giveOpt: &InitialOption{
				Keeper: &mod.MockKeeper{},
				Store:  &mod.MockStore{},
			},
			wantOpt: &InitialOption{
				Keeper:             &mod.MockKeeper{},
				Store:              &mod.MockStore{},
				ParserWorkersCnt:   100,
				ExecutorWorkerCnt:  1000,
				ExecutorTimeout:    time.Second * 30,
				DagScheduleTimeout: time.Second * 15,
			},
		},
		{
			giveOpt: &InitialOption{},
			wantOpt: &InitialOption{},
			wantErr: fmt.Errorf("keeper cannot be nil"),
		},
		{
			giveOpt: &InitialOption{
				Keeper: &mod.MockKeeper{},
			},
			wantOpt: &InitialOption{
				Keeper: &mod.MockKeeper{},
			},
			wantErr: fmt.Errorf("store cannot be nil"),
		},
	}

	for _, tc := range tests {
		err := checkOption(tc.giveOpt)
		assert.Equal(t, tc.wantErr, err)
		assert.Equal(t, tc.wantOpt, tc.giveOpt)
	}
}

func Test_readDagFromDir(t *testing.T) {
	tests := []struct {
		caseDesc       string
		givePaths      []string
		givePathsErr   error
		givePathDagMap map[string][]byte
		calledEnsured  []bool
		giveDir        string
		wantDag        *entity.Dag
		wantErr        error
	}{
		{
			caseDesc:     "read path failed",
			givePathsErr: fmt.Errorf("read failed"),
			wantErr:      fmt.Errorf("read failed"),
		},
		{
			caseDesc:  "get dag failed",
			givePaths: []string{"dag1", "dag2"},
			givePathDagMap: map[string][]byte{
				"dag1": {},
			},
			wantErr:       fmt.Errorf("read dag2 failed: %w", fmt.Errorf("not found")),
			calledEnsured: []bool{true},
		},
		{
			caseDesc:  "unmarshal dag failed",
			givePaths: []string{"dag1"},
			givePathDagMap: map[string][]byte{
				"dag1": []byte(`tasks: 123`),
			},
			wantErr: fmt.Errorf("unmarshal dag1 failed: %w", &yaml.TypeError{Errors: []string{"line 1: cannot unmarshal !!int `123` into entity.DagTasks"}}),
		},
		{
			caseDesc:  "normal",
			givePaths: []string{"dag1"},
			givePathDagMap: map[string][]byte{
				"dag1": []byte(`
id: test-dag
name: dag-name
desc: "desc"
tasks:
  - id: "task-1"
    actionName: "action"
    params: 
      p1: p1value
`),
			},
			calledEnsured: []bool{true},
			wantDag: &entity.Dag{
				BaseInfo: entity.BaseInfo{
					ID: "test-dag",
				},
				Name: "dag-name",
				Desc: "desc",
				Cron: "",
				Tasks: []entity.Task{
					{
						ID:         "task-1",
						ActionName: "action",
						Params: map[string]interface{}{
							"p1": "p1value",
						},
					},
				},
				Status: entity.DagStatusNormal,
			},
		},
		{
			caseDesc:  "no id",
			givePaths: []string{"/test/filename.yaml"},
			givePathDagMap: map[string][]byte{
				"/test/filename.yaml": []byte(``),
			},
			calledEnsured: []bool{true},
			wantDag: &entity.Dag{
				BaseInfo: entity.BaseInfo{
					ID: "filename",
				},
				Status: entity.DagStatusNormal,
			},
		},
		{
			caseDesc:  "no id(yml)",
			givePaths: []string{"c:/test/dag2.yaml"},
			givePathDagMap: map[string][]byte{
				"c:/test/dag2.yaml": []byte(``),
			},
			calledEnsured: []bool{true},
			wantDag: &entity.Dag{
				BaseInfo: entity.BaseInfo{
					ID: "dag2",
				},
				Status: entity.DagStatusNormal,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			mReader := &utils.MockDagReader{}
			mReader.On("ReadPathsFromDir", mock.Anything).Run(func(args mock.Arguments) {
				assert.Equal(t, tc.giveDir, args.Get(0))
			}).Return(tc.givePaths, tc.givePathsErr)
			mReader.On("ReadDag", mock.Anything).Run(func(args mock.Arguments) {
			}).Return(func(path string) []byte {
				return tc.givePathDagMap[path]
			}, func(path string) error {
				_, ok := tc.givePathDagMap[path]
				if !ok {
					return fmt.Errorf("not found")
				}
				return nil
			})
			utils.DefaultReader = mReader

			var called []bool
			mStore := &mod.MockStore{}
			existedDag := &entity.Dag{}
			mStore.On("GetDag", mock.Anything).Return(existedDag, nil)
			mStore.On("UpdateDag", mock.Anything).Run(func(args mock.Arguments) {
				called = append(called, true)
				if tc.wantDag != nil {
					assert.Equal(t, tc.wantDag, args.Get(0))
				}
			}).Return(nil, nil)
			mod.SetStore(mStore)

			err := readDagFromDir(tc.giveDir)
			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.calledEnsured, called)
		})
	}
}
