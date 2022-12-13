package mongo

import (
	"context"
	"errors"
	"fmt"
	"github.com/realeyeeos/fastflow/pkg/entity"
	"github.com/realeyeeos/fastflow/pkg/event"
	"github.com/realeyeeos/fastflow/pkg/log"
	"github.com/realeyeeos/fastflow/pkg/mod"
	"github.com/realeyeeos/fastflow/pkg/utils"
	"github.com/realeyeeos/fastflow/pkg/utils/data"
	"github.com/shiningrush/goevent"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"sync"
	"time"
)

// StoreOption
type StoreOption struct {
	// mongo connection string
	ConnStr  string
	Database string
	// Timeout access mongo timeout.default 5s
	Timeout time.Duration
	// the prefix will append to the database
	Prefix string
}

// Store
type Store struct {
	opt            *StoreOption
	dagClsName     string
	dagInsClsName  string
	taskInsClsName string

	mongoClient *mongo.Client
	mongoDb     *mongo.Database
}

// NewStore
func NewStore(option *StoreOption) *Store {
	return &Store{
		opt: option,
	}
}

// Init store
func (s *Store) Init() error {
	if err := s.readOpt(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.opt.Timeout)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(s.opt.ConnStr))
	if err != nil {
		return fmt.Errorf("connect client failed: %w", err)
	}
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return fmt.Errorf("ping client failed: %w", err)
	}
	s.mongoClient = client
	s.mongoDb = s.mongoClient.Database(s.opt.Database)

	return nil
}

func (s *Store) readOpt() error {
	if s.opt.ConnStr == "" {
		return fmt.Errorf("connect string cannot be empty")
	}
	if s.opt.Database == "" {
		s.opt.Database = "fastflow"
	}
	if s.opt.Timeout == 0 {
		s.opt.Timeout = 5 * time.Second
	}
	s.dagClsName = "dag"
	s.dagInsClsName = "dag_instance"
	s.taskInsClsName = "task_instance"
	if s.opt.Prefix != "" {
		s.dagClsName = fmt.Sprintf("%s_%s", s.opt.Prefix, s.dagClsName)
		s.dagInsClsName = fmt.Sprintf("%s_%s", s.opt.Prefix, s.dagInsClsName)
		s.taskInsClsName = fmt.Sprintf("%s_%s", s.opt.Prefix, s.taskInsClsName)
	}

	return nil
}

// Close component when we not use it anymore
func (s *Store) Close() {
	ctx, cancel := context.WithTimeout(context.TODO(), s.opt.Timeout)
	defer cancel()

	if err := s.mongoClient.Disconnect(ctx); err != nil {
		log.Errorf("close store client failed: %s", err)
	}
}

// CreateDag
func (s *Store) CreateDag(dag *entity.Dag) error {
	// check task's connection
	_, err := mod.BuildRootNode(mod.MapTasksToGetter(dag.Tasks))
	if err != nil {
		return err
	}
	return s.genericCreate(dag, s.dagClsName)
}

// CreateDagIns
func (s *Store) CreateDagIns(dagIns *entity.DagInstance) error {
	err := s.genericCreate(dagIns, s.dagInsClsName)
	if err != nil {
		fmt.Errorf("genericCreate Dag instance failed: %w", err)
	}
	err = s.createIndexForDagIns()
	if err != nil {
		fmt.Errorf("createIndexForDagIns failed: %w", err)
	}
	return nil
}

// CreateTaskIns
func (s *Store) CreateTaskIns(taskIns *entity.TaskInstance) error {
	err := s.genericCreate(taskIns, s.taskInsClsName)
	if err != nil {
		fmt.Errorf("genericCreate task instance failed: %w", err)
	}
	err = s.createIndexForTaskIns()
	if err != nil {
		fmt.Errorf("createIndexForTaskIns failed: %w", err)
	}
	return nil
}

func (s *Store) genericCreate(input entity.BaseInfoGetter, clsName string) error {
	baseInfo := input.GetBaseInfo()
	baseInfo.Initial()

	ctx, cancel := context.WithTimeout(context.TODO(), s.opt.Timeout)
	defer cancel()

	if _, err := s.mongoDb.Collection(clsName).InsertOne(ctx, input); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("%s key[ %s ] already existed: %w", clsName, baseInfo.ID, data.ErrDataConflicted)
		}

		return fmt.Errorf("insert instance failed: %w", err)
	}
	return nil
}

// BatchCreatTaskIns
func (s *Store) BatchCreatTaskIns(taskIns []*entity.TaskInstance) error {
	ctx, cancel := context.WithTimeout(context.TODO(), s.opt.Timeout)
	defer cancel()

	for i := range taskIns {
		taskIns[i].Initial()
		if _, err := s.mongoDb.Collection(s.taskInsClsName).InsertOne(ctx, taskIns[i]); err != nil {
			return fmt.Errorf("insert task instance failed: %w", err)
		}
	}
	return nil
}

// PatchTaskIns
func (s *Store) PatchTaskIns(taskIns *entity.TaskInstance) error {
	if taskIns.ID == "" {
		return fmt.Errorf("id cannot be empty")
	}
	update := bson.M{
		"updatedAt": time.Now().Unix(),
	}
	if taskIns.Status != "" {
		update["status"] = taskIns.Status
	}
	if taskIns.Reason != "" {
		update["reason"] = taskIns.Reason
	}
	if len(taskIns.Traces) > 0 {
		update["traces"] = taskIns.Traces
	}
	update = bson.M{
		"$set": update,
	}

	ctx, cancel := context.WithTimeout(context.TODO(), s.opt.Timeout)
	defer cancel()
	if _, err := s.mongoDb.Collection(s.taskInsClsName).UpdateOne(ctx, bson.M{"_id": taskIns.ID}, update); err != nil {
		return fmt.Errorf("patch task instance failed: %w", err)
	}
	return nil
}

// PatchDagIns
func (s *Store) PatchDagIns(dagIns *entity.DagInstance, mustsPatchFields ...string) error {
	update := bson.M{
		"updatedAt": time.Now().Unix(),
	}

	if dagIns.ShareData != nil {
		update["shareData"] = dagIns.ShareData
	}
	if dagIns.Status != "" {
		update["status"] = dagIns.Status
	}
	if utils.StringsContain(mustsPatchFields, "Cmd") || dagIns.Cmd != nil {
		update["cmd"] = dagIns.Cmd
	}
	if dagIns.Worker != "" {
		update["worker"] = dagIns.Worker
	}
	if utils.StringsContain(mustsPatchFields, "Reason") || dagIns.Reason != "" {
		update["reason"] = dagIns.Reason
	}

	update = bson.M{
		"$set": update,
	}

	ctx, cancel := context.WithTimeout(context.TODO(), s.opt.Timeout)
	defer cancel()
	if _, err := s.mongoDb.Collection(s.dagInsClsName).UpdateOne(ctx, bson.M{"_id": dagIns.ID}, update); err != nil {
		return fmt.Errorf("patch dag instance failed: %w", err)
	}

	goevent.Publish(&event.DagInstancePatched{
		Payload:         dagIns,
		MustPatchFields: mustsPatchFields,
	})
	return nil
}

// UpdateDag
func (s *Store) UpdateDag(dag *entity.Dag) error {
	// check task's connection
	_, err := mod.BuildRootNode(mod.MapTasksToGetter(dag.Tasks))
	if err != nil {
		return err
	}
	return s.genericUpdate(dag, s.dagClsName)
}

// UpdateDagIns
func (s *Store) UpdateDagIns(dagIns *entity.DagInstance) error {
	if err := s.genericUpdate(dagIns, s.dagInsClsName); err != nil {
		return err
	}

	goevent.Publish(&event.DagInstanceUpdated{Payload: dagIns})
	return nil
}

// UpdateTaskIns
func (s *Store) UpdateTaskIns(taskIns *entity.TaskInstance) error {
	return s.genericUpdate(taskIns, s.taskInsClsName)
}

// genericUpdate
func (s *Store) genericUpdate(input entity.BaseInfoGetter, clsName string) error {
	baseInfo := input.GetBaseInfo()
	baseInfo.Update()

	ctx, cancel := context.WithTimeout(context.TODO(), s.opt.Timeout)
	defer cancel()
	ret, err := s.mongoDb.Collection(clsName).ReplaceOne(ctx, bson.M{"_id": baseInfo.ID}, input)
	if err != nil {
		return fmt.Errorf("update dag instance failed: %w", err)
	}
	if ret.MatchedCount == 0 {
		return fmt.Errorf("%s has no key[ %s ] to update: %w", clsName, baseInfo.ID, data.ErrDataNotFound)
	}
	return nil
}

// BatchUpdateDagIns
func (s *Store) BatchUpdateDagIns(dagIns []*entity.DagInstance) error {
	ctx, cancel := context.WithTimeout(context.TODO(), s.opt.Timeout)
	defer cancel()

	errChan := make(chan error)
	defer close(errChan)

	errs := &data.Errors{}
	go func() {
		for err := range errChan {
			errs.Append(err)
		}
	}()

	wg := sync.WaitGroup{}
	for i := range dagIns {
		wg.Add(1)
		go func(dag *entity.DagInstance, ch chan error) {
			dag.Update()
			if _, err := s.mongoDb.Collection(s.dagInsClsName).ReplaceOne(
				ctx,
				bson.M{"_id": dag.ID}, dag); err != nil {
				errChan <- fmt.Errorf("batch update dag instance failed: %w", err)
			}
			wg.Done()
		}(dagIns[i], errChan)
	}
	wg.Wait()
	return nil
}

// BatchUpdateTaskIns
func (s *Store) BatchUpdateTaskIns(taskIns []*entity.TaskInstance) error {
	ctx, cancel := context.WithTimeout(context.TODO(), s.opt.Timeout)
	defer cancel()
	for i := range taskIns {
		taskIns[i].Update()
		if _, err := s.mongoDb.Collection(s.taskInsClsName).ReplaceOne(
			ctx,
			bson.M{"_id": taskIns[i].ID}, taskIns[i]); err != nil {
			return fmt.Errorf("batch update task instance failed: %w", err)
		}
	}
	return nil
}

// GetTaskIns
func (s *Store) GetTaskIns(taskInsId string) (*entity.TaskInstance, error) {
	ret := new(entity.TaskInstance)
	if err := s.genericGet(s.taskInsClsName, taskInsId, ret); err != nil {
		return nil, err
	}

	return ret, nil
}

// GetDag
func (s *Store) GetDag(dagId string) (*entity.Dag, error) {
	ret := new(entity.Dag)
	if err := s.genericGet(s.dagClsName, dagId, ret); err != nil {
		return nil, err
	}

	return ret, nil
}

// GetDagInstance
func (s *Store) GetDagInstance(dagInsId string) (*entity.DagInstance, error) {
	ret := new(entity.DagInstance)
	if err := s.genericGet(s.dagInsClsName, dagInsId, ret); err != nil {
		return nil, err
	}

	return ret, nil
}

func (s *Store) genericGet(clsName, id string, ret interface{}) error {
	ctx, cancel := context.WithTimeout(context.TODO(), s.opt.Timeout)
	defer cancel()

	if err := s.mongoDb.Collection(clsName).FindOne(ctx, bson.M{"_id": id}).Decode(ret); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("%s key[ %s ] not found: %w", clsName, id, data.ErrDataNotFound)
		}
		return fmt.Errorf("get dag instance failed: %w", err)
	}

	return nil
}

// ListDag
func (s *Store) ListDag(input *mod.ListDagInput) ([]*entity.Dag, error) {
	query := bson.M{}

	var ret []*entity.Dag
	err := s.genericList(&ret, s.dagClsName, query)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// ListDagInstance
func (s *Store) ListDagInstance(input *mod.ListDagInstanceInput) ([]*entity.DagInstance, error) {
	query := bson.M{}
	if len(input.Status) > 0 {
		query["status"] = bson.M{
			"$in": input.Status,
		}
	}
	if input.Worker != "" {
		query["worker"] = input.Worker
	}
	if input.UpdatedEnd > 0 {
		query["updatedAt"] = bson.M{
			"$lte": input.UpdatedEnd,
		}
	}
	if input.HasCmd {
		query["cmd"] = bson.M{
			"$ne": nil,
		}
	}
	opt := &options.FindOptions{}
	if input.Limit > 0 {
		opt.Limit = &input.Limit
	}

	var ret []*entity.DagInstance
	err := s.genericList(&ret, s.dagInsClsName, query, opt)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// ListTaskInstance
func (s *Store) ListTaskInstance(input *mod.ListTaskInstanceInput) ([]*entity.TaskInstance, error) {
	query := bson.M{}
	if len(input.IDs) > 0 {
		query["_id"] = bson.M{
			"$in": input.IDs,
		}
	}
	if len(input.Status) > 0 {
		query["status"] = bson.M{
			"$in": input.Status,
		}
	}
	if input.Expired {
		query["$expr"] = bson.M{
			"$lte": bson.A{
				"$updatedAt",
				bson.M{
					"$subtract": bson.A{
						// delay is prevent watch dog conflicted with task's context timeout
						time.Now().Unix() - 5,
						"$timeoutSecs",
					},
				},
			},
		}
	}
	if input.DagInsID != "" {
		query["dagInsId"] = input.DagInsID
	}
	opt := &options.FindOptions{}
	if len(input.SelectField) > 0 {
		fields := bson.M{}
		for _, f := range input.SelectField {
			fields[f] = 1
		}
		opt.Projection = fields
	}

	var ret []*entity.TaskInstance
	err := s.genericList(&ret, s.taskInsClsName, query, opt)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *Store) genericList(ret interface{}, clsName string, query bson.M, opts ...*options.FindOptions) error {
	ctx, cancel := context.WithTimeout(context.TODO(), s.opt.Timeout)
	defer cancel()

	cur, err := s.mongoDb.Collection(clsName).Find(ctx, query, opts...)
	if err != nil {
		return fmt.Errorf("find %s failed: %w", clsName, err)
	}
	if err := cur.All(ctx, ret); err != nil {
		return fmt.Errorf("decode failed: %w", err)
	}
	return nil
}

// BatchDeleteDag
func (s *Store) BatchDeleteDag(ids []string) error {
	return s.genericBatchDelete(ids, s.dagClsName)
}

// BatchDeleteDagIns
func (s *Store) BatchDeleteDagIns(ids []string) error {
	return s.genericBatchDelete(ids, s.dagInsClsName)
}

// BatchDeleteTaskIns
func (s *Store) BatchDeleteTaskIns(ids []string) error {
	return s.genericBatchDelete(ids, s.taskInsClsName)
}

// DelDagInsBytime
func (s *Store) DelDagInsBytime(hours int) error {
	return s.batchDeleteBytime(hours, s.dagInsClsName)
}

// DelTaskInsBytime
func (s *Store) DelTaskInsBytime(hours int) error {
	return s.batchDeleteBytime(hours, s.taskInsClsName)
}

func (s *Store) batchDeleteBytime(hours int, clsName string) error {
	ctx, cancel := context.WithTimeout(context.TODO(), s.opt.Timeout)
	defer cancel()

	timeUnix := time.Now().Unix() - int64(hours)*3600
	_, err := s.mongoDb.Collection(clsName).DeleteMany(ctx, bson.M{
		"createdAt": bson.M{
			"$lte": timeUnix,
		},
	})
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	return nil
}

func (s *Store) genericBatchDelete(ids []string, clsName string) error {
	ctx, cancel := context.WithTimeout(context.TODO(), s.opt.Timeout)
	defer cancel()

	_, err := s.mongoDb.Collection(clsName).DeleteMany(ctx, bson.M{
		"_id": bson.M{
			"$in": ids,
		},
	})
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	return nil
}

// Marshal
func (s *Store) Marshal(obj interface{}) ([]byte, error) {
	return bson.Marshal(obj)
}

// Unmarshal
func (s *Store) Unmarshal(bytes []byte, ptr interface{}) error {
	return bson.Unmarshal(bytes, ptr)
}

// create index for DagIns
func (s *Store) createIndexForDagIns() error {
	models := []mongo.IndexModel{
		{Keys: bson.D{{"cmd", 1}}, Options: options.Index().SetName("cmd_index")},
		{Keys: bson.D{{"status", 1}}, Options: options.Index().SetName("status_index")},
		{Keys: bson.D{{"updated_at", 1}}, Options: options.Index().SetName("updated_at_index")},
	}
	return s.createIndex(s.dagInsClsName, models)
}

// create index for TaskIns
func (s *Store) createIndexForTaskIns() error {
	models := []mongo.IndexModel{
		{Keys: bson.D{{"status", 1}}, Options: options.Index().SetName("status_index")},
		{Keys: bson.D{{"dagInsId", 1}}, Options: options.Index().SetName("dag_ins_id_index")},
		{Keys: bson.D{{"updated_at", 1}}, Options: options.Index().SetName("updated_at_index")},
	}
	return s.createIndex(s.taskInsClsName, models)
}

// create index
func (s *Store) createIndex(clsName string, models []mongo.IndexModel) error {
	ctx, cancel := context.WithTimeout(context.TODO(), s.opt.Timeout)
	defer cancel()

	opts := options.CreateIndexes().SetMaxTime(2 * time.Second)
	_, err := s.mongoDb.Collection(clsName).Indexes().CreateMany(ctx, models, opts)
	if err != nil {
		return err
	}
	return nil
}
