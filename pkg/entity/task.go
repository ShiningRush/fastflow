package entity

import (
	"fmt"
	"runtime"
	"time"

	"github.com/shiningrush/fastflow/pkg/entity/run"
	"github.com/shiningrush/fastflow/pkg/log"
	"github.com/shiningrush/fastflow/pkg/utils"
)

// Task
type Task struct {
	ID          string                 `yaml:"id,omitempty" json:"id,omitempty"  bson:"id,omitempty"`
	Name        string                 `yaml:"name,omitempty" json:"name,omitempty"  bson:"name,omitempty"`
	DependOn    []string               `yaml:"dependOn,omitempty" json:"dependOn,omitempty"  bson:"dependOn,omitempty"`
	ActionName  string                 `yaml:"actionName,omitempty" json:"actionName,omitempty"  bson:"actionName,omitempty"`
	TimeoutSecs int                    `yaml:"timeoutSecs,omitempty" json:"timeoutSecs,omitempty"  bson:"timeoutSecs,omitempty"`
	Params      map[string]interface{} `yaml:"params,omitempty" json:"params,omitempty"  bson:"params,omitempty"`
	PreChecks   PreChecks              `yaml:"preCheck,omitempty" json:"preCheck,omitempty"  bson:"preCheck,omitempty"`
}

// GetGraphID
func (t *Task) GetGraphID() string {
	return t.ID
}

// GetID
func (t *Task) GetID() string {
	return t.ID
}

// GetDepend
func (t *Task) GetDepend() []string {
	return t.DependOn
}

// GetStatus
func (t *Task) GetStatus() TaskInstanceStatus {
	return ""
}

type PreChecks map[string]*Check

// Check
type Check struct {
	Conditions []TaskCondition `yaml:"conditions,omitempty" json:"conditions,omitempty"  bson:"conditions,omitempty"`
	Act        ActiveAction    `yaml:"act,omitempty" json:"act,omitempty"  bson:"act,omitempty"`
}

// IsMeet return if check is meet
func (c *Check) IsMeet(dagIns *DagInstance) bool {
	for _, cd := range c.Conditions {
		if !cd.IsMeet(dagIns) {
			return false
		}
	}
	return true
}

type ActiveAction string

const (
	// skip action when all condition is meet, otherwise execute it
	ActiveActionSkip ActiveAction = "skip"
	// block action when all condition is meet, otherwise execute it
	ActiveActionBlock ActiveAction = "block"
)

type Operator string

const (
	OperatorIn    Operator = "in"
	OperatorNotIn Operator = "not-in"
)

type TaskConditionSource string

const (
	TaskConditionSourceVars      TaskConditionSource = "vars"
	TaskConditionSourceShareData TaskConditionSource = "share-data"
)

// BuildKvGetter
func (t TaskConditionSource) BuildKvGetter(dagIns *DagInstance) utils.KeyValueGetter {
	switch t {
	case TaskConditionSourceVars:
		return dagIns.VarsGetter()
	case TaskConditionSourceShareData:
		return dagIns.ShareData.Get
	default:
		panic(fmt.Sprintf("task condition source %s is not valid", t))
	}
}

// TaskCondition
type TaskCondition struct {
	Source TaskConditionSource `yaml:"source,omitempty" json:"source,omitempty"  bson:"source,omitempty"`
	Key    string              `yaml:"key,omitempty" json:"key,omitempty"  bson:"key,omitempty"`
	Values []string            `yaml:"values,omitempty" json:"values,omitempty"  bson:"values,omitempty"`
	Op     Operator            `yaml:"op,omitempty" json:"op,omitempty"  bson:"op,omitempty"`
}

// IsMeet return if check is meet
func (c *TaskCondition) IsMeet(dagIns *DagInstance) bool {
	kvGetter := c.Source.BuildKvGetter(dagIns)

	v, ok := kvGetter(c.Key)
	if !ok {
		return false
	}

	switch c.Op {
	case OperatorIn:
		return isStrInArray(v, c.Values)
	case OperatorNotIn:
		return !isStrInArray(v, c.Values)
	}
	return false
}

func isStrInArray(str string, arr []string) bool {
	for _, v := range arr {
		if v == str {
			return true
		}
	}

	return false
}

// TaskInstance
type TaskInstance struct {
	BaseInfo `bson:"inline"`
	// Task's Id it should be unique in a dag instance
	TaskID      string                 `json:"taskId,omitempty" bson:"taskId,omitempty" gorm:"type:VARCHAR(256);not null"`
	DagInsID    string                 `json:"dagInsId,omitempty" bson:"dagInsId,omitempty" gorm:"type:VARCHAR(256);not null;index"`
	Name        string                 `json:"name,omitempty" bson:"name,omitempty" gorm:"-"`
	DependOn    []string               `json:"dependOn,omitempty" bson:"dependOn,omitempty" gorm:"type:JSON;serializer:json"`
	ActionName  string                 `json:"actionName,omitempty" bson:"actionName,omitempty" gorm:"type:VARCHAR(256);not null"`
	TimeoutSecs int                    `json:"timeoutSecs" bson:"timeoutSecs" gorm:"type:bigint(20) unsigned"`
	Params      map[string]interface{} `json:"params,omitempty" bson:"params,omitempty" gorm:"type:JSON;serializer:json"`
	Traces      []TraceInfo            `json:"traces,omitempty" bson:"traces,omitempty" gorm:"type:JSON;serializer:json"`
	Status      TaskInstanceStatus     `json:"status,omitempty" bson:"status,omitempty" gorm:"type:enum('init', 'canceled', 'running', 'ending', 'failed', 'retrying', 'success', 'blocked', 'skipped');index;not null;"`
	Reason      string                 `json:"reason,omitempty" bson:"reason,omitempty" gorm:"type:TEXT"`
	PreChecks   PreChecks              `json:"preChecks,omitempty"  bson:"preChecks,omitempty" gorm:"type:JSON;serializer:json"`
	// used to save changes
	Patch              func(*TaskInstance) error `json:"-" bson:"-" gorm:"-"`
	Context            run.ExecuteContext        `json:"-" bson:"-" gorm:"-"`
	RelatedDagInstance *DagInstance              `json:"-" bson:"-" gorm:"-"`

	// it used to buffer traces, and persist when status changed
	bufTraces []TraceInfo `gorm:"-"`
}

// TraceInfo
type TraceInfo struct {
	Time    int64  `json:"time,omitempty" bson:"time,omitempty"`
	Message string `json:"message,omitempty" bson:"message,omitempty"`
}

// NewTaskInstance
func NewTaskInstance(dagInsId string, t Task) *TaskInstance {
	return &TaskInstance{
		TaskID:      t.ID,
		DagInsID:    dagInsId,
		Name:        t.Name,
		DependOn:    t.DependOn,
		ActionName:  t.ActionName,
		TimeoutSecs: t.TimeoutSecs,
		Params:      t.Params,
		Status:      TaskInstanceStatusInit,
		PreChecks:   t.PreChecks,
	}
}

// GetGraphID
func (t *TaskInstance) GetGraphID() string {
	return t.TaskID
}

// GetID
func (t *TaskInstance) GetID() string {
	return t.ID
}

// GetDepend
func (t *TaskInstance) GetDepend() []string {
	return t.DependOn
}

// GetStatus
func (t *TaskInstance) GetStatus() TaskInstanceStatus {
	return t.Status
}

// InitialDep
func (t *TaskInstance) InitialDep(ctx run.ExecuteContext, patch func(*TaskInstance) error, dagIns *DagInstance) {
	t.Patch = patch
	t.Context = ctx
	t.RelatedDagInstance = dagIns
}

// SetStatus will persist task instance
func (t *TaskInstance) SetStatus(s TaskInstanceStatus) error {
	t.Status = s
	patch := &TaskInstance{BaseInfo: BaseInfo{ID: t.ID}, Status: t.Status, Reason: t.Reason}
	if len(t.bufTraces) != 0 {
		patch.Traces = append(t.Traces, t.bufTraces...)
	}
	return t.Patch(patch)
}

// Trace info
func (t *TaskInstance) Trace(msg string, ops ...run.TraceOp) {
	opt := run.NewTraceOption(ops...)
	if opt.Priority == run.PersistPriorityAfterAction {
		t.bufTraces = append(t.bufTraces, TraceInfo{
			Time:    time.Now().Unix(),
			Message: msg,
		})
		return
	}

	t.Traces = append(t.Traces, TraceInfo{
		Time:    time.Now().Unix(),
		Message: msg,
	})

	if err := t.Patch(&TaskInstance{BaseInfo: BaseInfo{ID: t.ID}, Traces: t.Traces}); err != nil {
		log.Error("save trace failed",
			"err", err,
			"trace", t.Traces)
	}
}

// Run action
func (t *TaskInstance) Run(params interface{}, act run.Action) (err error) {
	defer func() {
		if rErr := recover(); rErr != nil {
			var stacktrace string
			for i := 2; ; i++ {
				_, f, l, got := runtime.Caller(i)
				if !got {
					break
				}
				stacktrace += fmt.Sprintf("%s:%d\n", f, l)
			}
			err = fmt.Errorf("get panic when running action: %s, err: %s, stack: %s", act.Name(), rErr, stacktrace)
		}

	}()

	if t.Status == TaskInstanceStatusInit {
		beforeAct, ok := act.(run.BeforeAction)
		if ok {
			if err := beforeAct.RunBefore(t.Context, params); err != nil {
				return fmt.Errorf("run before failed: %w", err)
			}
		}
		if err := t.SetStatus(TaskInstanceStatusRunning); err != nil {
			return err
		}

		if err := act.Run(t.Context, params); err != nil {
			return fmt.Errorf("run failed: %w", err)
		}

		if err := t.SetStatus(TaskInstanceStatusEnding); err != nil {
			return err
		}
	}

	if t.Status == TaskInstanceStatusEnding {
		afterAct, ok := act.(run.AfterAction)
		if ok {
			if err := afterAct.RunAfter(t.Context, params); err != nil {
				return fmt.Errorf("run after failed: %w", err)
			}
		}
		if err := t.SetStatus(TaskInstanceStatusSuccess); err != nil {
			return err
		}
	}

	if t.Status == TaskInstanceStatusRetrying {
		retryAct, ok := act.(run.RetryBeforeAction)
		if ok {
			if err := retryAct.RetryBefore(t.Context, params); err != nil {
				return fmt.Errorf("run retryBefore failed: %w", err)
			}
		}
		if err := t.SetStatus(TaskInstanceStatusInit); err != nil {
			return err
		}
	}
	return nil
}

// DoPreCheck
func (t *TaskInstance) DoPreCheck(dagIns *DagInstance) (isActive bool, err error) {
	if t.PreChecks == nil {
		return
	}

	for k, c := range t.PreChecks {
		if c.IsMeet(dagIns) {
			switch c.Act {
			case ActiveActionSkip:
				t.Status = TaskInstanceStatusSkipped
			case ActiveActionBlock:
				t.Status = TaskInstanceStatusBlocked
			default:
				return false, fmt.Errorf("pre-check[%s] act is invalid: %s", k, c.Act)
			}
			isActive = true
			return
		}
	}

	return
}

// TaskInstanceStatus
type TaskInstanceStatus string

const (
	TaskInstanceStatusInit     TaskInstanceStatus = "init"
	TaskInstanceStatusCanceled TaskInstanceStatus = "canceled"
	TaskInstanceStatusRunning  TaskInstanceStatus = "running"
	TaskInstanceStatusEnding   TaskInstanceStatus = "ending"
	TaskInstanceStatusFailed   TaskInstanceStatus = "failed"
	TaskInstanceStatusRetrying TaskInstanceStatus = "retrying"
	TaskInstanceStatusSuccess  TaskInstanceStatus = "success"
	TaskInstanceStatusBlocked  TaskInstanceStatus = "blocked"
	TaskInstanceStatusSkipped  TaskInstanceStatus = "skipped"
)
