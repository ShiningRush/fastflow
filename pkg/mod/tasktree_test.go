package mod

import (
	"fmt"
	"testing"

	"github.com/shiningrush/fastflow/pkg/entity"
	"github.com/stretchr/testify/assert"
)

func TestTaskNode_ComputeStatus(t *testing.T) {
	tests := []struct {
		caseDesc    string
		giveTaskIns []*entity.TaskInstance
		wantSrcId   string
		wantStatus  TreeStatus
	}{
		{
			caseDesc: "success",
			giveTaskIns: []*entity.TaskInstance{
				{
					BaseInfo: entity.BaseInfo{ID: "task1"},
					TaskID:   "task1",
					Status:   entity.TaskInstanceStatusSuccess,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "task2"},
					TaskID:   "task2",
					DependOn: []string{"task1"},
					Status:   entity.TaskInstanceStatusSkipped,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "task3"},
					TaskID:   "task3",
					DependOn: []string{"task1"},
					Status:   entity.TaskInstanceStatusSuccess,
				},
			},
			wantSrcId:  "",
			wantStatus: TreeStatusSuccess,
		},
		{
			caseDesc: "failed",
			giveTaskIns: []*entity.TaskInstance{
				{
					BaseInfo: entity.BaseInfo{ID: "task1"},
					TaskID:   "task1",
					Status:   entity.TaskInstanceStatusSuccess,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "task2"},
					TaskID:   "task2",
					DependOn: []string{"task1"},
					Status:   entity.TaskInstanceStatusSuccess,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "task3"},
					TaskID:   "task3",
					DependOn: []string{"task1"},
					Status:   entity.TaskInstanceStatusFailed,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "task4"},
					TaskID:   "task4",
					DependOn: []string{"task2", "task3"},
					Status:   entity.TaskInstanceStatusInit,
				},
			},
			wantSrcId:  "task3",
			wantStatus: TreeStatusFailed,
		},
		{
			caseDesc: "blocked",
			giveTaskIns: []*entity.TaskInstance{
				{
					BaseInfo: entity.BaseInfo{ID: "task1"},
					TaskID:   "task1",
					Status:   entity.TaskInstanceStatusBlocked,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "task2"},
					TaskID:   "task2",
					DependOn: []string{"task1"},
					Status:   entity.TaskInstanceStatusFailed,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "task3"},
					TaskID:   "task3",
					DependOn: []string{"task1"},
					Status:   entity.TaskInstanceStatusSuccess,
				},
			},
			wantSrcId:  "task1",
			wantStatus: TreeStatusBlocked,
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			root := MustBuildRootNode(MapTaskInsToGetter(tc.giveTaskIns))
			status, srcId := root.ComputeStatus()
			assert.Equal(t, tc.wantStatus, status)
			assert.Equal(t, tc.wantSrcId, srcId)
		})
	}
}

func TestBuildRootNode(t *testing.T) {
	tests := []struct {
		caseDesc   string
		giveDagIns *entity.DagInstance
		giveTasks  []*entity.TaskInstance
		wantRoot   *TaskNode
		wantErr    error
	}{
		{
			caseDesc: "normal",
			giveDagIns: &entity.DagInstance{
				BaseInfo: entity.BaseInfo{
					ID: "id",
				},
			},
			giveTasks: []*entity.TaskInstance{
				{
					BaseInfo: entity.BaseInfo{
						ID: "root1-ins",
					},
					TaskID: "root1",
				},
				{
					BaseInfo: entity.BaseInfo{
						ID: "r1-child1-ins",
					},
					TaskID:   "r1-child1",
					DependOn: []string{"root1"},
				},
				{
					BaseInfo: entity.BaseInfo{
						ID: "r1-child2-ins",
					},
					TaskID:   "r1-child2",
					DependOn: []string{"root1"},
				},
				{
					BaseInfo: entity.BaseInfo{
						ID: "c1-child1-ins",
					},
					TaskID:   "c1-child1",
					DependOn: []string{"r1-child2"},
				},
				{
					BaseInfo: entity.BaseInfo{
						ID: "root2-ins",
					},
					TaskID: "root2",
				},
				{
					BaseInfo: entity.BaseInfo{
						ID: "r2-child1-ins",
					},
					TaskID:   "r2-child1",
					DependOn: []string{"root2"},
					Status:   entity.TaskInstanceStatusInit,
				},
			},
			wantRoot: &TaskNode{
				TaskInsID: virtualTaskRootID,
				Status:    entity.TaskInstanceStatusSuccess,
				children: []*TaskNode{
					{
						TaskInsID: "root1-ins",
						children: []*TaskNode{
							{
								TaskInsID: "r1-child1-ins",
							},
							{
								TaskInsID: "r1-child2-ins",
								children: []*TaskNode{
									{TaskInsID: "c1-child1-ins"},
								},
							},
						},
					},
					{
						TaskInsID: "root2-ins",
						children: []*TaskNode{
							{TaskInsID: "r2-child1-ins", Status: entity.TaskInstanceStatusInit},
						},
					},
				},
			},
		},
		{
			caseDesc: "parent not existed",
			giveDagIns: &entity.DagInstance{
				BaseInfo: entity.BaseInfo{
					ID: "id",
				},
			},
			giveTasks: []*entity.TaskInstance{
				{
					TaskID: "root1",
				},
				{
					TaskID:   "r1-child1",
					DependOn: []string{"root2"},
				},
			},
			wantRoot: nil,
			wantErr:  fmt.Errorf("does not find task[r1-child1] depend: root2"),
		},
		{
			caseDesc: "no start nodes",
			giveDagIns: &entity.DagInstance{
				BaseInfo: entity.BaseInfo{
					ID: "id",
				},
			},
			giveTasks: []*entity.TaskInstance{
				{
					TaskID:   "child1",
					DependOn: []string{"child1"},
				},
				{
					TaskID:   "child2",
					DependOn: []string{"child1"},
				},
			},
			wantRoot: nil,
			wantErr:  fmt.Errorf("here is no start nodes"),
		},
		{
			caseDesc: "has cycle",
			giveDagIns: &entity.DagInstance{
				BaseInfo: entity.BaseInfo{
					ID: "id",
				},
			},
			giveTasks: []*entity.TaskInstance{
				{
					BaseInfo: entity.BaseInfo{
						ID: "root",
					},
					TaskID: "root",
				},
				{
					BaseInfo: entity.BaseInfo{
						ID: "child1",
					},
					TaskID:   "child1",
					DependOn: []string{"root", "child3"},
				},
				{
					BaseInfo: entity.BaseInfo{
						ID: "child2",
					},
					TaskID:   "child2",
					DependOn: []string{"child1"},
				},
				{
					BaseInfo: entity.BaseInfo{
						ID: "child3",
					},
					TaskID:   "child3",
					DependOn: []string{"child2"},
				},
			},
			wantRoot: nil,
			wantErr:  fmt.Errorf("dag has cycle at: child1"),
		},
		{
			caseDesc: "branch should not error",
			giveDagIns: &entity.DagInstance{
				BaseInfo: entity.BaseInfo{
					ID: "id",
				},
			},
			giveTasks: []*entity.TaskInstance{
				{
					BaseInfo: entity.BaseInfo{
						ID: "root",
					},
					TaskID: "root",
				},
				{
					BaseInfo: entity.BaseInfo{
						ID: "child1",
					},
					TaskID:   "child1",
					DependOn: []string{"root"},
				},
				{
					BaseInfo: entity.BaseInfo{
						ID: "child2",
					},
					TaskID:   "child2",
					DependOn: []string{"root"},
				},
				{
					BaseInfo: entity.BaseInfo{
						ID: "child3",
					},
					TaskID:   "child3",
					DependOn: []string{"child2"},
				},
				{
					BaseInfo: entity.BaseInfo{
						ID: "child4",
					},
					TaskID:   "child4",
					DependOn: []string{"child3", "child1"},
				},
			},
			wantRoot: &TaskNode{
				TaskInsID: virtualTaskRootID,
				Status:    entity.TaskInstanceStatusSuccess,
				children: []*TaskNode{
					{
						TaskInsID: "root",
						children: []*TaskNode{
							{
								TaskInsID: "child1",
								children: []*TaskNode{
									{TaskInsID: "child4"},
								},
							},
							{
								TaskInsID: "child2",
								children: []*TaskNode{
									{
										TaskInsID: "child3",
										children: []*TaskNode{
											{TaskInsID: "child4"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			ret, err := BuildRootNode(MapTaskInsToGetter(tc.giveTasks))
			assert.Equal(t, tc.wantErr, err)
			if err == nil {
				checkParentAndRemoveIt(t, ret, nil)
			}
			assert.Equal(t, tc.wantRoot, ret)
		})
	}
}

func checkParentAndRemoveIt(t *testing.T, node, pNode *TaskNode) {
	if pNode != nil {
		find := false
		var newParents []*TaskNode
		for _, n := range node.parents {
			if n == pNode {
				find = true
				continue
			}
			newParents = append(newParents, n)
		}
		node.parents = newParents
		if !find {
			assert.Fail(t, "parent node is not contain")
			return
		}
	}

	for _, c := range node.children {
		checkParentAndRemoveIt(t, c, node)
	}
}

func TestTaskNode_GetNextTaskIds(t *testing.T) {
	tests := []struct {
		caseDesc     string
		giveTasks    []*MockTaskInfoGetter
		giveTask     *entity.TaskInstance
		wantTaskNode *TaskNode
		wantRet      []string
		wantFind     bool
	}{
		{
			caseDesc: "root task succeed",
			giveTask: &entity.TaskInstance{
				BaseInfo: entity.BaseInfo{
					ID: "root",
				},
				Status: entity.TaskInstanceStatusSuccess,
			},
			giveTasks: []*MockTaskInfoGetter{
				{
					ID:     "root",
					Status: entity.TaskInstanceStatusRunning,
				},
				{
					ID:     "child1",
					Status: entity.TaskInstanceStatusRunning,
					Depend: []string{"root"},
				},
				{
					ID:     "child2",
					Status: entity.TaskInstanceStatusInit,
					Depend: []string{"root"},
				},
			},
			wantTaskNode: &TaskNode{
				TaskInsID: "root",
				Status:    entity.TaskInstanceStatusSuccess,
				children: []*TaskNode{
					{TaskInsID: "child1", Status: entity.TaskInstanceStatusRunning},
					{TaskInsID: "child2", Status: entity.TaskInstanceStatusInit},
				},
			},
			wantRet: []string{
				"child2",
			},
			wantFind: true,
		},
		{
			caseDesc: "parent task skipped",
			giveTask: &entity.TaskInstance{
				BaseInfo: entity.BaseInfo{
					ID: "child1",
				},
				Status: entity.TaskInstanceStatusSkipped,
			},
			giveTasks: []*MockTaskInfoGetter{
				{
					ID:     "root",
					Status: entity.TaskInstanceStatusSuccess,
				},
				{
					ID:     "child1",
					Status: entity.TaskInstanceStatusInit,
					Depend: []string{"root"},
				},
				{
					ID:     "child2",
					Status: entity.TaskInstanceStatusInit,
					Depend: []string{"child1"},
				},
			},
			wantTaskNode: &TaskNode{
				TaskInsID: "root",
				Status:    entity.TaskInstanceStatusSuccess,
				children: []*TaskNode{
					{TaskInsID: "child1", Status: entity.TaskInstanceStatusSkipped, children: []*TaskNode{
						{TaskInsID: "child2", Status: entity.TaskInstanceStatusInit},
					}},
				},
			},
			wantRet: []string{
				"child2",
			},
			wantFind: true,
		},
		{
			caseDesc: "root task failed",
			giveTask: &entity.TaskInstance{
				BaseInfo: entity.BaseInfo{
					ID: "root",
				},
				Status: entity.TaskInstanceStatusFailed,
			},
			giveTasks: []*MockTaskInfoGetter{
				{
					ID:     "root",
					Status: entity.TaskInstanceStatusRunning,
				},
				{
					ID:     "child1",
					Status: entity.TaskInstanceStatusInit,
					Depend: []string{"root"},
				},
				{
					ID:     "child2",
					Status: entity.TaskInstanceStatusInit,
					Depend: []string{"root"},
				},
			},
			wantTaskNode: &TaskNode{
				TaskInsID: "root",
				Status:    entity.TaskInstanceStatusFailed,
				children: []*TaskNode{
					{TaskInsID: "child1", Status: entity.TaskInstanceStatusInit},
					{TaskInsID: "child2", Status: entity.TaskInstanceStatusInit},
				},
			},
			wantFind: true,
		},
		{
			caseDesc: "root task blocked",
			giveTask: &entity.TaskInstance{
				BaseInfo: entity.BaseInfo{
					ID: "root",
				},
				Status: entity.TaskInstanceStatusBlocked,
			},
			giveTasks: []*MockTaskInfoGetter{
				{
					ID:     "root",
					Status: entity.TaskInstanceStatusRunning,
				},
				{
					ID:     "child1",
					Status: entity.TaskInstanceStatusInit,
					Depend: []string{"root"},
				},
				{
					ID:     "child2",
					Status: entity.TaskInstanceStatusInit,
					Depend: []string{"root"},
				},
			},
			wantTaskNode: &TaskNode{
				TaskInsID: "root",
				Status:    entity.TaskInstanceStatusBlocked,
				children: []*TaskNode{
					{TaskInsID: "child1", Status: entity.TaskInstanceStatusInit},
					{TaskInsID: "child2", Status: entity.TaskInstanceStatusInit},
				},
			},
			wantFind: true,
		},
		{
			caseDesc: "child task succeed",
			giveTask: &entity.TaskInstance{
				BaseInfo: entity.BaseInfo{
					ID: "child1",
				},
				Status: entity.TaskInstanceStatusSuccess,
			},
			giveTasks: []*MockTaskInfoGetter{
				{
					ID:     "root",
					Status: entity.TaskInstanceStatusSuccess,
				},
				{
					ID:     "child1",
					Status: entity.TaskInstanceStatusRunning,
					Depend: []string{"root"},
				},
				{
					ID:     "c1-child1",
					Status: entity.TaskInstanceStatusEnding,
					Depend: []string{"child1"},
				},
				{
					ID:     "c1-child2",
					Status: entity.TaskInstanceStatusEnding,
					Depend: []string{"child1"},
				},
				{
					ID:     "child2",
					Status: entity.TaskInstanceStatusInit,
					Depend: []string{"root"},
				},
				{
					ID:     "c2-child1",
					Status: entity.TaskInstanceStatusEnding,
					Depend: []string{"child2"},
				},
				{
					ID:     "c2-child2",
					Status: entity.TaskInstanceStatusEnding,
					Depend: []string{"child2"},
				},
			},
			wantTaskNode: &TaskNode{
				TaskInsID: "root",
				Status:    entity.TaskInstanceStatusSuccess,
				children: []*TaskNode{
					{TaskInsID: "child1", Status: entity.TaskInstanceStatusSuccess, children: []*TaskNode{
						{TaskInsID: "c1-child1", Status: entity.TaskInstanceStatusEnding},
						{TaskInsID: "c1-child2", Status: entity.TaskInstanceStatusEnding},
					}},
					{TaskInsID: "child2", Status: entity.TaskInstanceStatusInit, children: []*TaskNode{
						{TaskInsID: "c2-child1", Status: entity.TaskInstanceStatusEnding},
						{TaskInsID: "c2-child2", Status: entity.TaskInstanceStatusEnding},
					}},
				},
			},
			wantRet: []string{
				"c1-child1",
				"c1-child2",
			},
			wantFind: true,
		},
		{
			caseDesc: "child task succeed but root not success",
			giveTask: &entity.TaskInstance{
				BaseInfo: entity.BaseInfo{
					ID: "child1",
				},
				Status: entity.TaskInstanceStatusSuccess,
			},
			giveTasks: []*MockTaskInfoGetter{
				{
					ID:     "root",
					Status: entity.TaskInstanceStatusRunning,
				},
				{
					ID:     "child1",
					Status: entity.TaskInstanceStatusInit,
					Depend: []string{"root"},
				},
				{
					ID:     "c1-child1",
					Status: entity.TaskInstanceStatusEnding,
					Depend: []string{"child1"},
				},
				{
					ID:     "c1-child2",
					Status: entity.TaskInstanceStatusEnding,
					Depend: []string{"child1"},
				},
				{
					ID:     "child2",
					Status: entity.TaskInstanceStatusInit,
					Depend: []string{"root"},
				},
				{
					ID:     "c2-child1",
					Status: entity.TaskInstanceStatusEnding,
					Depend: []string{"child2"},
				},
				{
					ID:     "c2-child2",
					Status: entity.TaskInstanceStatusEnding,
					Depend: []string{"child2"},
				},
			},
			wantTaskNode: &TaskNode{
				TaskInsID: "root",
				Status:    entity.TaskInstanceStatusRunning,
				children: []*TaskNode{
					{TaskInsID: "child1", Status: entity.TaskInstanceStatusInit, children: []*TaskNode{
						{TaskInsID: "c1-child1", Status: entity.TaskInstanceStatusEnding},
						{TaskInsID: "c1-child2", Status: entity.TaskInstanceStatusEnding},
					}},
					{TaskInsID: "child2", Status: entity.TaskInstanceStatusInit, children: []*TaskNode{
						{TaskInsID: "c2-child1", Status: entity.TaskInstanceStatusEnding},
						{TaskInsID: "c2-child2", Status: entity.TaskInstanceStatusEnding},
					}},
				},
			},
			wantFind: false,
		},
		{
			caseDesc: "leaf node completed",
			giveTask: &entity.TaskInstance{
				BaseInfo: entity.BaseInfo{
					ID: "c1-child1",
				},
				Status: entity.TaskInstanceStatusSuccess,
			},
			giveTasks: []*MockTaskInfoGetter{
				{
					ID:     "root",
					Status: entity.TaskInstanceStatusSuccess,
				},
				{
					ID:     "child1",
					Status: entity.TaskInstanceStatusSuccess,
					Depend: []string{"root"},
				},
				{
					ID:     "c1-child1",
					Status: entity.TaskInstanceStatusEnding,
					Depend: []string{"child1"},
				},
				{
					ID:     "c1-child2",
					Status: entity.TaskInstanceStatusEnding,
					Depend: []string{"child1"},
				},
				{
					ID:     "child2",
					Status: entity.TaskInstanceStatusInit,
					Depend: []string{"root"},
				},
				{
					ID:     "c2-child1",
					Status: entity.TaskInstanceStatusEnding,
					Depend: []string{"child2"},
				},
				{
					ID:     "c2-child2",
					Status: entity.TaskInstanceStatusEnding,
					Depend: []string{"child2"},
				},
			},
			wantTaskNode: &TaskNode{
				TaskInsID: "root",
				Status:    entity.TaskInstanceStatusSuccess,
				children: []*TaskNode{
					{TaskInsID: "child1", Status: entity.TaskInstanceStatusSuccess, children: []*TaskNode{
						{TaskInsID: "c1-child1", Status: entity.TaskInstanceStatusSuccess},
						{TaskInsID: "c1-child2", Status: entity.TaskInstanceStatusEnding},
					}},
					{TaskInsID: "child2", Status: entity.TaskInstanceStatusInit, children: []*TaskNode{
						{TaskInsID: "c2-child1", Status: entity.TaskInstanceStatusEnding},
						{TaskInsID: "c2-child2", Status: entity.TaskInstanceStatusEnding},
					}},
				},
			},
			wantFind: true,
		},
		{
			caseDesc: "retry node",
			giveTask: &entity.TaskInstance{
				BaseInfo: entity.BaseInfo{
					ID: "child1",
				},
				Status: entity.TaskInstanceStatusInit,
			},
			giveTasks: []*MockTaskInfoGetter{
				{
					ID:     "root",
					Status: entity.TaskInstanceStatusSuccess,
				},
				{
					ID:     "child1",
					Status: entity.TaskInstanceStatusRetrying,
					Depend: []string{"root"},
				},
				{
					ID:     "c1-child1",
					Status: entity.TaskInstanceStatusInit,
					Depend: []string{"child1"},
				},
				{
					ID:     "c1-child2",
					Status: entity.TaskInstanceStatusInit,
					Depend: []string{"child1"},
				},
			},
			wantTaskNode: &TaskNode{
				TaskInsID: "root",
				Status:    entity.TaskInstanceStatusSuccess,
				children: []*TaskNode{
					{TaskInsID: "child1", Status: entity.TaskInstanceStatusInit, children: []*TaskNode{
						{TaskInsID: "c1-child1", Status: entity.TaskInstanceStatusInit},
						{TaskInsID: "c1-child2", Status: entity.TaskInstanceStatusInit},
					}},
				},
			},
			wantFind: true,
			wantRet:  []string{"child1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			root, err := BuildRootNode(MapMockTasksToGetter(tc.giveTasks))
			assert.NoError(t, err)
			ret, find := root.GetNextTaskIds(tc.giveTask)
			assert.Equal(t, tc.wantRet, ret)
			assert.Equal(t, tc.wantFind, find)
			checkParentAndRemoveIt(t, root, nil)
			if root.TaskInsID == virtualTaskRootID {
				root = root.children[0]
			}
			assert.Equal(t, tc.wantTaskNode, root)
		})
	}
}

func TestTaskNode_GetExecutableTaskIds(t *testing.T) {
	tests := []struct {
		caseDesc  string
		giveTasks []*entity.TaskInstance
		wantRet   []string
	}{
		{
			caseDesc: "root node executable",
			giveTasks: []*entity.TaskInstance{
				{
					BaseInfo: entity.BaseInfo{ID: "root"},
					TaskID:   "root",
					Status:   entity.TaskInstanceStatusEnding,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "child1"},
					TaskID:   "child1",
					DependOn: []string{"root"},
					Status:   entity.TaskInstanceStatusSuccess,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "c1-child1"},
					TaskID:   "c1-child1",
					DependOn: []string{"child1"},
					Status:   entity.TaskInstanceStatusEnding,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "c1-child2"},
					TaskID:   "c1-child2",
					DependOn: []string{"child1"},
					Status:   entity.TaskInstanceStatusEnding,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "child2"},
					TaskID:   "child2",
					DependOn: []string{"root"},
					Status:   entity.TaskInstanceStatusEnding,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "c2-child1"},
					TaskID:   "c2-child1",
					DependOn: []string{"child2"},
					Status:   entity.TaskInstanceStatusEnding,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "c2-child2"},
					TaskID:   "c2-child2",
					DependOn: []string{"child2"},
					Status:   entity.TaskInstanceStatusEnding,
				},
			},
			wantRet: []string{
				"root",
			},
		},
		{
			caseDesc: "root node failed",
			giveTasks: []*entity.TaskInstance{
				{
					BaseInfo: entity.BaseInfo{ID: "root"},
					TaskID:   "root",
					Status:   entity.TaskInstanceStatusFailed,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "child1"},
					TaskID:   "child1",
					DependOn: []string{"root"},
					Status:   entity.TaskInstanceStatusSuccess,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "c1-child1"},
					TaskID:   "c1-child1",
					DependOn: []string{"child1"},
					Status:   entity.TaskInstanceStatusEnding,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "c1-child2"},
					TaskID:   "c1-child2",
					DependOn: []string{"child1"},
					Status:   entity.TaskInstanceStatusEnding,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "child2"},
					TaskID:   "child2",
					DependOn: []string{"child1"},
					Status:   entity.TaskInstanceStatusEnding,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "c2-child1"},
					TaskID:   "c2-child1",
					DependOn: []string{"child2"},
					Status:   entity.TaskInstanceStatusEnding,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "c2-child2"},
					TaskID:   "c2-child2",
					DependOn: []string{"child2"},
					Status:   entity.TaskInstanceStatusEnding,
				},
			},
		},
		{
			caseDesc: "child task executable",
			giveTasks: []*entity.TaskInstance{
				{
					BaseInfo: entity.BaseInfo{ID: "root"},
					TaskID:   "root",
					Status:   entity.TaskInstanceStatusSuccess,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "child1"},
					TaskID:   "child1",
					DependOn: []string{"root"},
					Status:   entity.TaskInstanceStatusSkipped,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "c1-child1"},
					TaskID:   "c1-child1",
					DependOn: []string{"child1"},
					Status:   entity.TaskInstanceStatusEnding,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "c1-child2"},
					TaskID:   "c1-child2",
					DependOn: []string{"child1"},
					Status:   entity.TaskInstanceStatusEnding,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "child2"},
					TaskID:   "child2",
					DependOn: []string{"root"},
					Status:   entity.TaskInstanceStatusEnding,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "c2-child1"},
					TaskID:   "c2-child1",
					DependOn: []string{"child2"},
					Status:   entity.TaskInstanceStatusEnding,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "c2-child2"},
					TaskID:   "c2-child2",
					DependOn: []string{"child2"},
					Status:   entity.TaskInstanceStatusEnding,
				},
			},
			wantRet: []string{
				"c1-child1",
				"c1-child2",
				"child2",
			},
		},
		{
			caseDesc: "common child",
			giveTasks: []*entity.TaskInstance{
				{
					BaseInfo: entity.BaseInfo{ID: "root"},
					TaskID:   "root",
					Status:   entity.TaskInstanceStatusSuccess,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "child1"},
					TaskID:   "child1",
					DependOn: []string{"root"},
					Status:   entity.TaskInstanceStatusSuccess,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "child2"},
					TaskID:   "child2",
					Status:   entity.TaskInstanceStatusInit,
				},
				{
					BaseInfo: entity.BaseInfo{ID: "c2-child1"},
					TaskID:   "common -child",
					DependOn: []string{"child1", "child2"},
					Status:   entity.TaskInstanceStatusEnding,
				},
			},
			wantRet: []string{
				"child2",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {

			root, err := BuildRootNode(MapTaskInsToGetter(tc.giveTasks))
			assert.NoError(t, err)
			assert.Equal(t, tc.wantRet, root.GetExecutableTaskIds())
		})
	}
}

func TestTaskNode_Executable(t *testing.T) {
	tests := []struct {
		caseDesc     string
		giveTaskNode *TaskNode
		wantRet      bool
	}{
		{
			caseDesc: "running task",
			giveTaskNode: &TaskNode{
				Status: entity.TaskInstanceStatusRunning,
			},
			wantRet: false,
		},
		{
			caseDesc: "ending task",
			giveTaskNode: &TaskNode{
				Status: entity.TaskInstanceStatusEnding,
			},
			wantRet: true,
		},
		{
			caseDesc: "ending task has succeed parents",
			giveTaskNode: &TaskNode{
				Status: entity.TaskInstanceStatusEnding,
				parents: []*TaskNode{
					{Status: entity.TaskInstanceStatusSuccess},
					{Status: entity.TaskInstanceStatusSuccess},
				},
			},
			wantRet: true,
		},
		{
			caseDesc: "ending task has ending parents",
			giveTaskNode: &TaskNode{
				Status: entity.TaskInstanceStatusEnding,
				parents: []*TaskNode{
					{Status: entity.TaskInstanceStatusSuccess},
					{Status: entity.TaskInstanceStatusEnding},
				},
			},
			wantRet: false,
		},
		{
			caseDesc: "init task",
			giveTaskNode: &TaskNode{
				Status: entity.TaskInstanceStatusInit,
			},
			wantRet: true,
		},
		{
			caseDesc: "retrying task",
			giveTaskNode: &TaskNode{
				Status: entity.TaskInstanceStatusRetrying,
			},
			wantRet: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseDesc, func(t *testing.T) {
			assert.Equal(t, tc.wantRet, tc.giveTaskNode.Executable())
		})
	}
}
