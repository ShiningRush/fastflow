package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/shiningrush/fastflow"
	mongoKeeper "github.com/shiningrush/fastflow/keeper/mongo"
	"github.com/shiningrush/fastflow/pkg/entity"
	"github.com/shiningrush/fastflow/pkg/entity/run"
	"github.com/shiningrush/fastflow/pkg/exporter"
	"github.com/shiningrush/fastflow/pkg/mod"
	"github.com/shiningrush/fastflow/pkg/utils/data"
	mongoStore "github.com/shiningrush/fastflow/store/mongo"
)

type ActionParam struct {
	Name string
	Desc string
}

type ActionA struct {
	code string
}

func (a *ActionA) Name() string {
	return fmt.Sprintf("Action-%s", a.code)
}
func (a *ActionA) RunBefore(ctx run.ExecuteContext, params interface{}) error {
	input := params.(*ActionParam)
	log.Println(fmt.Sprintf("%s run before, p.Name: %s, p.Desc: %s", a.Name(), input.Name, input.Desc))
	time.Sleep(time.Second)
	if a.code != "B" && a.code != "C" {
		ctx.ShareData().Set(fmt.Sprintf("%s-key", a.code), fmt.Sprintf("%s value", a.code))
	}
	return nil
}
func (a *ActionA) Run(ctx run.ExecuteContext, params interface{}) error {
	input := params.(*ActionParam)
	log.Println(fmt.Sprintf("%s run, p.Name: %s, p.Desc: %s", a.Name(), input.Name, input.Desc))
	ctx.Trace("run start", run.TraceOpPersistAfterAction)
	time.Sleep(2 * time.Second)
	ctx.Trace("run end")
	return nil
}
func (a *ActionA) RunAfter(ctx run.ExecuteContext, params interface{}) error {
	input := params.(*ActionParam)
	log.Println(fmt.Sprintf("%s run after, p.Name: %s, p.Desc: %s", a.Name(), input.Name, input.Desc))
	time.Sleep(time.Second)
	return nil
}
func (a *ActionA) ParameterNew() interface{} {
	return &ActionParam{}
}

func ensureDagCreated() error {
	dag := &entity.Dag{
		BaseInfo: entity.BaseInfo{
			ID: "test-dag",
		},
		Name: "test",
		Vars: entity.DagVars{
			"var": {DefaultValue: "default-var"},
		},
		Status: entity.DagStatusNormal,
		Tasks: []entity.Task{
			{ID: "task1", ActionName: "Action-A", Params: map[string]interface{}{
				"Name": "task-p1",
				"Desc": "{{var}}",
			}, TimeoutSecs: 5},
			{ID: "task2", ActionName: "Action-B", DependOn: []string{"task1"}, Params: map[string]interface{}{
				"Name": "task-p1",
				"Desc": "{{var}}",
			}},
			{ID: "task3", ActionName: "Action-C", DependOn: []string{"task1"}, Params: map[string]interface{}{
				"Name": "task-p1",
				"Desc": "{{var}}",
			}},
			{ID: "task4", ActionName: "Action-D", DependOn: []string{"task2", "task3"}, Params: map[string]interface{}{
				"Name": "task-p1",
				"Desc": "{{var}}",
			}},
		},
	}
	oldDag, err := mod.GetStore().GetDag(dag.ID)
	if errors.Is(err, data.ErrDataNotFound) {
		if err := mod.GetStore().CreateDag(dag); err != nil {
			return err
		}
	}
	if oldDag != nil {
		if err := mod.GetStore().UpdateDag(dag); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	// init action
	fastflow.RegisterAction([]run.Action{
		&ActionA{code: "A"},
		&ActionA{code: "B"},
		&ActionA{code: "C"},
		&ActionA{code: "D"},
	})
	// init keeper
	keeper := mongoKeeper.NewKeeper(&mongoKeeper.KeeperOption{
		Key: "worker-1",
		// if your mongo does not set user/pwd, you should remove it
		ConnStr:  "mongodb://root:pwd@127.0.0.1:27017/fastflow?authSource=admin",
		Database: "mongo-demo",
		Prefix:   "test",
	})
	if err := keeper.Init(); err != nil {
		log.Fatal(fmt.Errorf("init keeper failed: %w", err))
	}

	// init store
	st := mongoStore.NewStore(&mongoStore.StoreOption{
		// if your mongo does not set user/pwd, you should remove it
		ConnStr:  "mongodb://root:pwd@127.0.0.1:27017/fastflow?authSource=admin",
		Database: "mongo-demo",
		Prefix:   "test",
	})
	if err := st.Init(); err != nil {
		log.Fatal(fmt.Errorf("init store failed: %w", err))
	}

	// init fastflow
	if err := fastflow.Init(&fastflow.InitialOption{
		Keeper:            keeper,
		Store:             st,
		ParserWorkersCnt:  10,
		ExecutorWorkerCnt: 50,
	}); err != nil {
		panic(fmt.Sprintf("init fastflow failed: %s", err))
	}

	// create a dag as template
	if err := ensureDagCreated(); err != nil {
		log.Fatalf(err.Error())
	}
	// run dag interval
	go runInstance()

	// listen a http endpoint to serve metrics
	if err := http.ListenAndServe(":9090", exporter.HttpHandler()); err != nil {
		panic(fmt.Sprintf("metrics serve failed: %s", err))
	}
}

func runInstance() {
	// wait init completed
	time.Sleep(2 * time.Second)
	dag, err := mod.GetStore().GetDag("test-dag")
	if err != nil {
		panic(err)
	}

	count := uint64(0)
	for {
		runVar := map[string]string{
			"var": "run-var",
		}
		if count%2 == 0 {
			runVar = nil
		}
		dagIns, err := dag.Run(entity.TriggerManually, runVar)
		if err != nil {
			panic(err)
		}

		err = mod.GetStore().CreateDagIns(dagIns)
		if err != nil {
			panic(err)
		}

		count++
		time.Sleep(1 * time.Second)
	}
}
