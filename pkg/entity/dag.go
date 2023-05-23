package entity

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/shiningrush/fastflow/pkg/log"
	"github.com/shiningrush/fastflow/pkg/utils"
	"github.com/shiningrush/fastflow/pkg/utils/value"
)

// NewDag new a dag
func NewDag() *Dag {
	return &Dag{
		Status: DagStatusNormal,
	}
}

// Dag
type Dag struct {
	BaseInfo `yaml:",inline" json:",inline" bson:"inline"`
	Name     string    `yaml:"name,omitempty" json:"name,omitempty" bson:"name,omitempty" gorm:"type:VARCHAR(128);not null"`
	Desc     string    `yaml:"desc,omitempty" json:"desc,omitempty" bson:"desc,omitempty" gorm:"type:VARCHAR(256);not null"`
	Cron     string    `yaml:"cron,omitempty" json:"cron,omitempty" bson:"cron,omitempty" gorm:"-"`
	Vars     DagVars   `yaml:"vars,omitempty" json:"vars,omitempty" bson:"vars,omitempty" gorm:"type:JSON;serializer:json"`
	Status   DagStatus `yaml:"status,omitempty" json:"status,omitempty" bson:"status,omitempty" gorm:"type:enum('normal', 'stopped');not null;"`
	Tasks    DagTasks  `yaml:"tasks,omitempty" json:"tasks,omitempty" bson:"tasks,omitempty" gorm:"type:JSON;not null;serializer:json"`
}

type DagTasks []Task

// SpecifiedVar
type SpecifiedVar struct {
	Name  string
	Value string
}

// Run used to build a new DagInstance, then you also need save it to Store
func (d *Dag) Run(trigger Trigger, specVars map[string]string) (*DagInstance, error) {
	if d.Status != DagStatusNormal {
		return nil, fmt.Errorf("you cannot run a stopeed dag")
	}

	dagInsVars := DagInstanceVars{}
	for key, value := range d.Vars {
		v := value.DefaultValue
		if specVars != nil && specVars[key] != "" {
			v = specVars[key]
		}
		dagInsVars[key] = DagInstanceVar{
			Value: v,
		}
	}

	return &DagInstance{
		DagID:     d.ID,
		Trigger:   trigger,
		Vars:      dagInsVars,
		ShareData: &ShareData{},
		Status:    DagInstanceStatusInit,
	}, nil
}

type DagVars map[string]DagVar

// DagVar
type DagVar struct {
	Desc         string `yaml:"desc,omitempty" json:"desc,omitempty" bson:"desc,omitempty"`
	DefaultValue string `yaml:"defaultValue,omitempty" json:"defaultValue,omitempty" bson:"defaultValue,omitempty"`
}

// DagInstanceVar
type DagInstanceVar struct {
	Value string `json:"value,omitempty" bson:"value,omitempty"`
}

// DagStatus
type DagStatus string

const (
	DagStatusNormal  DagStatus = "normal"
	DagStatusStopped DagStatus = "stopped"
)

// DagInstance
type DagInstance struct {
	BaseInfo  `bson:"inline"`
	DagID     string            `json:"dagId,omitempty" bson:"dagId,omitempty" gorm:"type:VARCHAR(256);not null"`
	Trigger   Trigger           `json:"trigger,omitempty" bson:"trigger,omitempty" gorm:"type:enum('manually', 'cron');not null;"`
	Worker    string            `json:"worker,omitempty" bson:"worker,omitempty" gorm:"type:VARCHAR(256)"`
	Vars      DagInstanceVars   `json:"vars,omitempty" bson:"vars,omitempty" gorm:"type:JSON;serializer:json"`
	ShareData *ShareData        `json:"shareData,omitempty" bson:"shareData,omitempty" gorm:"type:JSON;serializer:json"`
	Status    DagInstanceStatus `json:"status,omitempty" bson:"status,omitempty" gorm:"type:enum('init', 'scheduled', 'running', 'blocked', 'failed', 'success');index;not null;"`
	Reason    string            `json:"reason,omitempty" bson:"reason,omitempty" gorm:"type:TEXT"`
	Cmd       *Command          `json:"cmd,omitempty" bson:"cmd,omitempty" gorm:"type:JSON;serializer:json"`
}

var (
	StoreMarshal   func(interface{}) ([]byte, error)
	StoreUnmarshal func([]byte, interface{}) error
)

// ShareData can read/write within all tasks and will persist it
// if you want a high performance just within same task, you can use
// ExecuteContext's Context
type ShareData struct {
	Dict map[string]string
	Save func(data *ShareData) error

	mutex sync.Mutex
}

// MarshalBSON used by mongo
func (d *ShareData) MarshalBSON() ([]byte, error) {
	return StoreMarshal(d.Dict)
}

// UnmarshalBSON used by mongo
func (d *ShareData) UnmarshalBSON(data []byte) error {
	if d.Dict == nil {
		d.Dict = make(map[string]string)
	}
	return StoreUnmarshal(data, &d.Dict)
}

// MarshalJSON used by json
func (d *ShareData) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Dict)
}

// UnmarshalJSON used by json
func (d *ShareData) UnmarshalJSON(data []byte) error {
	if d.Dict == nil {
		d.Dict = make(map[string]string)
	}
	return json.Unmarshal(data, &d.Dict)
}

// Get value from share data, it is thread-safe.
func (d *ShareData) Get(key string) (string, bool) {
	if d.Dict == nil {
		return "", false
	}
	d.mutex.Lock()
	defer d.mutex.Unlock()

	v, ok := d.Dict[key]
	return v, ok
}

// Set value to share data, it is thread-safe.
func (d *ShareData) Set(key string, val string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.Dict[key] = val
	if d.Save != nil {
		if err := d.Save(d); err != nil {
			delete(d.Dict, key)
			log.Error("save share data failed",
				"err", err,
				"key", key,
				"value", val)
		}
	}
}

// DagInstanceVars
type DagInstanceVars map[string]DagInstanceVar

// Cancel a task, it is just set a command, command will execute by Parser
func (dagIns *DagInstance) Cancel(taskInsIds []string) error {
	if dagIns.Status != DagInstanceStatusRunning {
		return fmt.Errorf("you can only cancel a running dag instance")
	}
	if dagIns.Cmd != nil {
		return fmt.Errorf("dag instance have a incomplete command")
	}
	dagIns.Cmd = &Command{
		Name:             CommandNameCancel,
		TargetTaskInsIDs: taskInsIds,
	}
	return nil
}

var (
	HookDagInstance DagInstanceLifecycleHook
)

type DagInstanceHookFunc func(dagIns *DagInstance)

// DagInstanceLifecycleHook
type DagInstanceLifecycleHook struct {
	BeforeRun     DagInstanceHookFunc
	BeforeSuccess DagInstanceHookFunc
	BeforeFail    DagInstanceHookFunc
	BeforeBlock   DagInstanceHookFunc
	BeforeRetry   DagInstanceHookFunc
}

// VarsGetter
func (dagIns *DagInstance) VarsGetter() utils.KeyValueGetter {
	return func(key string) (string, bool) {
		val, ok := dagIns.Vars[key]
		return val.Value, ok
	}
}

// VarsIterator
func (dagIns *DagInstance) VarsIterator() utils.KeyValueIterator {
	return func(iterateFunc utils.KeyValueIterateFunc) {
		for k, v := range dagIns.Vars {
			if iterateFunc(k, v.Value) {
				break
			}
		}
	}
}

// Success the dag instance
func (dagIns *DagInstance) Run() {
	dagIns.executeHook(HookDagInstance.BeforeRun)
	dagIns.Status = DagInstanceStatusRunning
	dagIns.Reason = ""
}

// Success the dag instance
func (dagIns *DagInstance) Success() {
	dagIns.executeHook(HookDagInstance.BeforeSuccess)
	dagIns.Status = DagInstanceStatusSuccess
	dagIns.Reason = ""
}

// Fail the dag instance
func (dagIns *DagInstance) Fail(reason string) {
	dagIns.Reason = reason
	dagIns.executeHook(HookDagInstance.BeforeFail)
	dagIns.Status = DagInstanceStatusFailed
}

// Block the dag instance
func (dagIns *DagInstance) Block(reason string) {
	dagIns.executeHook(HookDagInstance.BeforeBlock)
	dagIns.Status = DagInstanceStatusBlocked
}

// Retry a task, it is just set a command, command will execute by Parser
func (dagIns *DagInstance) Retry(taskInsIds []string) error {
	if dagIns.Cmd != nil {
		return fmt.Errorf("dag instance have a incomplete command")
	}

	dagIns.executeHook(HookDagInstance.BeforeRetry)
	dagIns.Cmd = &Command{
		Name:             CommandNameRetry,
		TargetTaskInsIDs: taskInsIds,
	}
	return nil
}

func (dagIns *DagInstance) executeHook(hookFunc DagInstanceHookFunc) {
	if hookFunc != nil {
		hookFunc(dagIns)
	}
}

// CanChange indicate if the dag instance can modify status
func (dagIns *DagInstance) CanModifyStatus() bool {
	return dagIns.Status != DagInstanceStatusFailed
}

// Render variables
func (vars DagInstanceVars) Render(p map[string]interface{}) (map[string]interface{}, error) {
	err := value.MapValue(p).WalkString(func(walkContext *value.WalkContext, s string) error {
		for varKey, varValue := range vars {
			s = strings.ReplaceAll(s, fmt.Sprintf("{{%s}}", varKey), varValue.Value)
		}
		walkContext.Setter(s)
		return nil
	})
	return p, err
}

// Command
type Command struct {
	Name             CommandName
	TargetTaskInsIDs []string
}

// CommandName
type CommandName string

const (
	CommandNameRetry  = "retry"
	CommandNameCancel = "cancel"
)

// DagInstanceStatus
type DagInstanceStatus string

const (
	DagInstanceStatusInit      DagInstanceStatus = "init"
	DagInstanceStatusScheduled DagInstanceStatus = "scheduled"
	DagInstanceStatusRunning   DagInstanceStatus = "running"
	DagInstanceStatusBlocked   DagInstanceStatus = "blocked"
	DagInstanceStatusFailed    DagInstanceStatus = "failed"
	DagInstanceStatusSuccess   DagInstanceStatus = "success"
)

// Trigger
type Trigger string

const (
	TriggerManually Trigger = "manually"
	TriggerCron     Trigger = "cron"
)
