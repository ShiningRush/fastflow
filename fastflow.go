package fastflow

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/shiningrush/fastflow/pkg/actions"
	"github.com/shiningrush/fastflow/pkg/entity"
	"github.com/shiningrush/fastflow/pkg/entity/run"
	"github.com/shiningrush/fastflow/pkg/event"
	"github.com/shiningrush/fastflow/pkg/mod"
	"github.com/shiningrush/fastflow/pkg/utils"
	"github.com/shiningrush/fastflow/pkg/utils/data"
	"github.com/shiningrush/goevent"
	"gopkg.in/yaml.v3"
)

var closers []mod.Closer

// RegisterAction you need register all used action to it
func RegisterAction(acts []run.Action) {
	for i := range acts {
		mod.ActionMap[acts[i].Name()] = acts[i]
	}
}

// GetAction
func GetAction(name string) (run.Action, bool) {
	act, ok := mod.ActionMap[name]
	return act, ok
}

// InitialOption
type InitialOption struct {
	Keeper mod.Keeper
	Store  mod.Store

	// ParserWorkersCnt default 100
	ParserWorkersCnt int
	// ExecutorWorkerCnt default 1000
	ExecutorWorkerCnt int
	// ExecutorTimeout default 30s
	ExecutorTimeout time.Duration
	// ExecutorTimeout default 15s
	DagScheduleTimeout time.Duration

	// Read dag define from directory
	// each file will be pared to a dag, so you CAN'T define all dag in one file
	ReadDagFromDir string
}

// Start will block until accept system signal, if you don't want block, plz check "Init"
func Start(opt *InitialOption, afterInit ...func() error) error {
	if err := Init(opt); err != nil {
		return err
	}
	for i := range afterInit {
		if err := afterInit[i](); err != nil {
			return err
		}
	}

	log.Println("fastflow start success")
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-c
	log.Println(fmt.Sprintf("get sig: %s, ready to close component", sig))
	Close()
	log.Println("close completed")
	return nil
}

// Init will not block, but you need to close fastflow after application closing
func Init(opt *InitialOption) error {
	if err := checkOption(opt); err != nil {
		return err
	}

	initCommonComponent(opt)
	initLeaderChangedHandler(opt)

	RegisterAction([]run.Action{
		&actions.Waiting{},
	})

	if opt.ReadDagFromDir != "" {
		return readDagFromDir(opt.ReadDagFromDir)
	}
	return nil
}

func initLeaderChangedHandler(opt *InitialOption) {
	h := &LeaderChangedHandler{
		opt: opt,
	}
	if err := goevent.Subscribe(h); err != nil {
		log.Fatalln(err)
	}
	closers = append(closers, h)

	// when application init, leader election should completed, so we need trigger it
	if opt.Keeper.IsLeader() {
		h.Handle(context.Background(), &event.LeaderChanged{
			IsLeader:  true,
			WorkerKey: opt.Keeper.WorkerKey(),
		})
	}
	return
}

// SetDagInstanceLifecycleHook set hook handler for fastflow
// IMPORTANT: you MUST set hook before you call Init or Start to avoid lost changes.
// (because component will work immediately after you call Init or Start)
func SetDagInstanceLifecycleHook(hook entity.DagInstanceLifecycleHook) {
	entity.HookDagInstance = hook
}

// LeaderChangedHandler used to handle leader chaged event
type LeaderChangedHandler struct {
	opt *InitialOption

	leaderCloser []mod.Closer
	mutex        sync.Mutex
}

// Topic
func (l *LeaderChangedHandler) Topic() []string {
	return []string{event.KeyLeaderChanged}
}

// Handle
func (l *LeaderChangedHandler) Handle(cxt context.Context, e goevent.Event) {
	lcEvent := e.(*event.LeaderChanged)
	// changed to leader
	if lcEvent.IsLeader && len(l.leaderCloser) == 0 {
		wg := mod.NewDefWatchDog(l.opt.DagScheduleTimeout)
		wg.Init()
		l.leaderCloser = append(l.leaderCloser, wg)

		dis := mod.NewDefDispatcher()
		dis.Init()
		l.leaderCloser = append(l.leaderCloser, dis)
		log.Println("leader initial")
	}
	// continue leader failed
	if !lcEvent.IsLeader && len(l.leaderCloser) == 0 {
		l.Close()
	}
}

// Close leader component
func (l *LeaderChangedHandler) Close() {
	for i := range l.leaderCloser {
		l.leaderCloser[i].Close()
	}
	l.leaderCloser = []mod.Closer{}
}

// Close all closer
func Close() {
	for i := range closers {
		closers[i].Close()
	}
	goevent.Close()
}

func checkOption(opt *InitialOption) error {
	if opt.Keeper == nil {
		return fmt.Errorf("keeper cannot be nil")
	}
	if opt.Store == nil {
		return fmt.Errorf("store cannot be nil")
	}

	if opt.ExecutorTimeout == 0 {
		opt.ExecutorTimeout = 30 * time.Second
	}
	if opt.DagScheduleTimeout == 0 {
		opt.DagScheduleTimeout = 15 * time.Second
	}
	if opt.ExecutorWorkerCnt == 0 {
		opt.ExecutorWorkerCnt = 1000
	}
	if opt.ParserWorkersCnt == 0 {
		opt.ParserWorkersCnt = 100
	}
	return nil
}

func initCommonComponent(opt *InitialOption) {
	mod.SetKeeper(opt.Keeper)
	mod.SetStore(opt.Store)
	entity.StoreMarshal = opt.Store.Marshal
	entity.StoreUnmarshal = opt.Store.Unmarshal

	// Executor must init before parse otherwise will cause a error
	exe := mod.NewDefExecutor(opt.ExecutorTimeout, opt.ExecutorWorkerCnt)
	mod.SetExecutor(exe)
	p := mod.NewDefParser(opt.ParserWorkersCnt, opt.ExecutorTimeout)
	mod.SetParser(p)

	exe.Init()
	closers = append(closers, exe)
	p.Init()
	closers = append(closers, p)

	comm := &mod.DefCommander{}
	mod.SetCommander(comm)

	// keeper and store must close latest
	closers = append(closers, opt.Store)
	closers = append(closers, opt.Keeper)
}

func readDagFromDir(dir string) error {
	paths, err := utils.DefaultReader.ReadPathsFromDir(dir)
	if err != nil {
		return err
	}

	for _, path := range paths {
		bs, err := utils.DefaultReader.ReadDag(path)
		if err != nil {
			return fmt.Errorf("read %s failed: %w", path, err)
		}

		dag := entity.Dag{
			Status: entity.DagStatusNormal,
		}
		err = yaml.Unmarshal(bs, &dag)
		if err != nil {
			return fmt.Errorf("unmarshal %s failed: %w", path, err)
		}

		if dag.ID == "" {
			dag.ID = strings.TrimSuffix(strings.TrimSuffix(filepath.Base(path), ".yaml"), ".yml")
		}

		if err := ensureDagLatest(&dag); err != nil {
			return err
		}
	}
	return nil
}

func ensureDagLatest(dag *entity.Dag) error {
	oDag, err := mod.GetStore().GetDag(dag.ID)
	if err != nil && !errors.Is(err, data.ErrDataNotFound) {
		return err
	}
	if oDag != nil {
		return mod.GetStore().UpdateDag(dag)
	}

	return mod.GetStore().CreateDag(dag)
}
