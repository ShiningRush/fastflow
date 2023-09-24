package mod

import (
	"time"

	"github.com/shiningrush/fastflow/pkg/entity"
	"github.com/shiningrush/fastflow/pkg/entity/run"
)

var (
	ActionMap = map[string]run.Action{}

	defExc       Executor
	defStore     Store
	defKeeper    Keeper
	defParser    Parser
	defCommander Commander
)

// Commander used to execute command
type Commander interface {
	RunDag(dagId string, specVar map[string]string) (*entity.DagInstance, error)
	RetryDagIns(dagInsId string, ops ...CommandOptSetter) error
	RetryTask(taskInsIds []string, ops ...CommandOptSetter) error
	CancelTask(taskInsIds []string, ops ...CommandOptSetter) error
	ContinueDagIns(dagInsId string, ops ...CommandOptSetter) error
	ContinueTask(taskInsIds []string, ops ...CommandOptSetter) error
}

// CommandOption
type CommandOption struct {
	// isSync means commander will watch dag instance's cmd executing situation until it's command is executed
	// usually command executing time is very short, so async mode is enough,
	// but if you want a sync call, you set it to true
	isSync bool
	// syncTimeout is just work at sync mode, it is the timeout of watch dag instance
	// default is 5s
	syncTimeout time.Duration
	// syncInterval is just work at sync mode, it is the interval of watch dag instance
	// default is 500ms
	syncInterval time.Duration
}
type CommandOptSetter func(opt *CommandOption)

var (
	// CommSync means commander will watch dag instance's cmd executing situation until it's command is executed
	// usually command executing time is very short, so async mode is enough,
	// but if you want a sync call, you set it to true
	CommSync = func() CommandOptSetter {
		return func(opt *CommandOption) {
			opt.isSync = true
		}
	}
	// CommSync means commander will watch dag instance's cmd executing situation until it's command is executed
	// usually command executing time is very short, so async mode is enough,
	// but if you want a sync call, you set it to true
	CommSyncTimeout = func(duration time.Duration) CommandOptSetter {
		return func(opt *CommandOption) {
			if duration > 0 {
				opt.syncTimeout = duration
			}
		}
	}
	// CommSyncInterval is just work at sync mode, it is the interval of watch dag instance
	// default is 500ms
	CommSyncInterval = func(duration time.Duration) CommandOptSetter {
		return func(opt *CommandOption) {
			if duration > 0 {
				opt.syncInterval = duration
			}
		}
	}
)

// SetCommander
func SetCommander(c Commander) {
	defCommander = c
}

// GetCommander
func GetCommander() Commander {
	return defCommander
}

// Executor is used to execute task
type Executor interface {
	Push(dagIns *entity.DagInstance, taskIns *entity.TaskInstance)
	CancelTaskIns(taskInsIds []string) error
}

// SetExecutor
func SetExecutor(e Executor) {
	defExc = e
}

// GetExecutor
func GetExecutor() Executor {
	return defExc
}

// Closer means the component need be closeFunc
type Closer interface {
	Close()
}

// Store used to persist obj
type Store interface {
	Closer
	CreateDag(dag *entity.Dag) error
	CreateDagIns(dagIns *entity.DagInstance) error
	BatchCreatTaskIns(taskIns []*entity.TaskInstance) error
	PatchTaskIns(taskIns *entity.TaskInstance) error
	PatchDagIns(dagIns *entity.DagInstance, mustsPatchFields ...string) error
	UpdateDag(dagIns *entity.Dag) error
	UpdateDagIns(dagIns *entity.DagInstance) error
	UpdateTaskIns(taskIns *entity.TaskInstance) error
	BatchUpdateDagIns(dagIns []*entity.DagInstance) error
	BatchUpdateTaskIns(taskIns []*entity.TaskInstance) error
	GetTaskIns(taskIns string) (*entity.TaskInstance, error)
	GetDag(dagId string) (*entity.Dag, error)
	GetDagInstance(dagInsId string) (*entity.DagInstance, error)
	ListDagInstance(input *ListDagInstanceInput) ([]*entity.DagInstance, error)
	ListTaskInstance(input *ListTaskInstanceInput) ([]*entity.TaskInstance, error)
	Marshal(obj interface{}) ([]byte, error)
	Unmarshal(bytes []byte, ptr interface{}) error
}

// ListDagInput
type ListDagInput struct {
}

// ListDagInstanceInput
type ListDagInstanceInput struct {
	Worker     string
	DagID      string
	UpdatedEnd int64
	Status     []entity.DagInstanceStatus
	HasCmd     bool
	Limit      int64
	Offset     int64
}

// ListTaskInstanceInput
type ListTaskInstanceInput struct {
	IDs      []string
	DagInsID string
	Status   []entity.TaskInstanceStatus
	// query expired tasks(it will calculate task's timeout)
	Expired     bool
	SelectField []string
}

// SetStore
func SetStore(e Store) {
	defStore = e
}

// GetStore
func GetStore() Store {
	return defStore
}

// Keeper
type Keeper interface {
	Closer
	IsLeader() bool
	IsAlive(workerKey string) (bool, error)
	AliveNodes() ([]string, error)
	WorkerKey() string
	WorkerNumber() int
	NewMutex(key string) DistributedMutex
}

// SetKeeper
func SetKeeper(e Keeper) {
	defKeeper = e
}

// GetKeeper
func GetKeeper() Keeper {
	return defKeeper
}

// Parser used to execute command, init dag instance and push task instance
type Parser interface {
	InitialDagIns(dagIns *entity.DagInstance)
	EntryTaskIns(taskIns *entity.TaskInstance)
}

// SetParser
func SetParser(e Parser) {
	defParser = e
}

// GetParser
func GetParser() Parser {
	return defParser
}
