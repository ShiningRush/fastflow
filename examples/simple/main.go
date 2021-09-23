package main

import (
	"errors"
	"fmt"
	"github.com/shiningrush/fastflow"
	mongoKeeper "github.com/shiningrush/fastflow/keeper/mongo"
	"github.com/shiningrush/fastflow/pkg/entity"
	"github.com/shiningrush/fastflow/pkg/entity/run"
	"github.com/shiningrush/fastflow/pkg/mod"
	"github.com/shiningrush/fastflow/pkg/utils/data"
	mongoStore "github.com/shiningrush/fastflow/store/mongo"
	"log"
	"time"
)

type PrintAction struct {
}

// Name define the unique action identity, it will used by Task
func (a *PrintAction) Name() string {
	return "PrintAction"
}
func (a *PrintAction) Run(ctx run.ExecuteContext, params interface{}) error {
	fmt.Println("action start: ", time.Now())
	return nil
}

func main() {
	// Register action
	fastflow.RegisterAction([]run.Action{
		&PrintAction{},
	})

	// init keeper, it used to e
	keeper := mongoKeeper.NewKeeper(&mongoKeeper.KeeperOption{
		Key:      "worker-1",
		ConnStr:  "mongodb://root:pwd@127.0.0.1:27017/fastflow?authSource=admin",
		Database: "mongo-demo",
		Prefix:   "test",
	})
	if err := keeper.Init(); err != nil {
		log.Fatal(fmt.Errorf("init keeper failed: %w", err))
	}

	// init store
	st := mongoStore.NewStore(&mongoStore.StoreOption{
		ConnStr:  "mongodb://root:pwd@127.0.0.1:27017/fastflow?authSource=admin",
		Database: "mongo-demo",
		Prefix:   "test",
	})
	if err := st.Init(); err != nil {
		log.Fatal(fmt.Errorf("init store failed: %w", err))
	}

	go createDagAndInstance()

	// start fastflow
	if err := fastflow.Start(&fastflow.InitialOption{
		Keeper: keeper,
		Store:  st,
	}); err != nil {
		panic(fmt.Sprintf("init fastflow failed: %s", err))
	}
}

func createDagAndInstance() {
	// wait fast start completed
	time.Sleep(time.Second)

	// create a dag as template
	dag := &entity.Dag{
		BaseInfo: entity.BaseInfo{
			ID: "test-dag",
		},
		Name: "test",
		Tasks: []entity.Task{
			{ID: "task1", ActionName: "PrintAction"},
			{ID: "task2", ActionName: "PrintAction", DependOn: []string{"task1"}},
			{ID: "task3", ActionName: "PrintAction", DependOn: []string{"task2"}},
		},
	}
	if err := ensureDagCreated(dag); err != nil {
		log.Fatalf(err.Error())
	}

	// run some dag instance
	for i := 0; i < 10; i++ {
		dagInstance, err := dag.Run(entity.TriggerManually, nil)
		if err != nil {
			log.Fatal(err)
		}
		if err := mod.GetStore().CreateDagIns(dagInstance); err != nil {
			log.Fatal(err)
		}
		time.Sleep(time.Second * 10)
	}
}

func ensureDagCreated(dag *entity.Dag) error {
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
