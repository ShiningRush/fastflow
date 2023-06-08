package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/shiningrush/fastflow/pkg/entity"
	"github.com/shiningrush/fastflow/pkg/event"
	"github.com/shiningrush/fastflow/pkg/log"
	"github.com/shiningrush/fastflow/pkg/mod"
	"github.com/shiningrush/fastflow/pkg/utils"
	"github.com/shiningrush/fastflow/pkg/utils/data"
	"github.com/shiningrush/goevent"
	gormDriver "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// StoreOption
type StoreOption struct {
	MySQLConfig *mysql.Config
	GormConfig  *gorm.Config
	PoolConfig  *ConnectionPoolOption

	// business timeout
	Timeout           time.Duration
	MigrationSwitch   bool
	BatchUpdateConfig *BatchUpdateOption
}

type ConnectionPoolOption struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

type BatchUpdateOption struct {
	ConcurrencyCount int
	Timeout          time.Duration
}

// Store
type Store struct {
	opt *StoreOption

	gormDB *gorm.DB
}

func (s *Store) Close() {
	sqlDB, err := s.gormDB.DB()
	if err != nil {
		log.Errorf("get store client failed: %s", err)
	}

	if err = sqlDB.Close(); err != nil {
		log.Errorf("close store client failed: %s", err)
	}
}

func (s *Store) CreateDag(dag *entity.Dag) error {
	// check task's connection
	_, err := mod.BuildRootNode(mod.MapTasksToGetter(dag.Tasks))
	if err != nil {
		return err
	}

	err = s.transaction(func(tx *gorm.DB) error {
		dag.BaseInfo.Initial()
		return tx.Create(dag).Error
	})
	if err != nil {
		return fmt.Errorf("insert dag failed: %w", err)
	}
	return nil
}

func (s *Store) transaction(cb func(tx *gorm.DB) error) error {
	ctx, cancel := context.WithTimeout(context.TODO(), s.opt.Timeout)
	defer cancel()

	db := s.gormDB.WithContext(ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		return cb(tx)
	})
}

func (s *Store) CreateDagIns(dagIns *entity.DagInstance) error {
	err := s.transaction(func(tx *gorm.DB) error {
		dagIns.BaseInfo.Initial()
		if dagIns.ShareData == nil {
			dagIns.ShareData = &entity.ShareData{}
		}
		if dagIns.ShareData.Dict == nil {
			dagIns.ShareData.Dict = map[string]string{}
		}
		err := tx.Create(dagIns).Error
		if err != nil {
			return err
		}
		if len(dagIns.Tags) == 0 {
			return nil
		}
		for _, tag := range dagIns.Tags {
			tag.BaseInfo.Initial()
			tag.DagInsId = dagIns.ID
			err := tx.Create(tag).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("insert dagIns failed: %w", err)
	}
	return nil
}

func (s *Store) BatchCreatTaskIns(taskIns []*entity.TaskInstance) error {
	err := s.transaction(func(tx *gorm.DB) error {
		for _, task := range taskIns {
			task.BaseInfo.Initial()
		}
		return tx.CreateInBatches(taskIns, 100).Error
	})
	if err != nil {
		return fmt.Errorf("batch insert taskIns failed: %w", err)
	}
	return nil
}

func (s *Store) PatchTaskIns(taskIns *entity.TaskInstance) error {
	if taskIns.ID == "" {
		return fmt.Errorf("id cannot be empty")
	}

	updateIns := &entity.TaskInstance{}
	updateIns.BaseInfo = taskIns.BaseInfo
	updateIns.BaseInfo.Update()
	if taskIns.Status != "" {
		updateIns.Status = taskIns.Status
	}
	if taskIns.Reason != "" {
		updateIns.Reason = taskIns.Reason
	}
	if len(taskIns.Traces) > 0 {
		updateIns.Traces = taskIns.Traces
	}

	err := s.transaction(func(tx *gorm.DB) error {
		return tx.Model(taskIns).Updates(updateIns).Error
	})
	if err != nil {
		return fmt.Errorf("patch taskIns failed: %w", err)
	}
	return nil
}

func (s *Store) PatchDagIns(dagIns *entity.DagInstance, mustsPatchFields ...string) error {
	if dagIns.ID == "" {
		return fmt.Errorf("id cannot be empty")
	}

	updateIns := &entity.DagInstance{}
	updateIns.BaseInfo.Update()
	updateFields := []string{"UpdatedAt"}
	if dagIns.ShareData != nil {
		updateFields = append(updateFields, "ShareData")
		updateIns.ShareData = dagIns.ShareData
	}
	if dagIns.Status != "" {
		updateFields = append(updateFields, "Status")
		updateIns.Status = dagIns.Status
	}
	if utils.StringsContain(mustsPatchFields, "Cmd") || dagIns.Cmd != nil {
		updateFields = append(updateFields, "Cmd")
		updateIns.Cmd = dagIns.Cmd
	}
	if dagIns.Worker != "" {
		updateFields = append(updateFields, "Worker")
		updateIns.Worker = dagIns.Worker
	}
	if utils.StringsContain(mustsPatchFields, "Reason") || dagIns.Reason != "" {
		updateFields = append(updateFields, "Reason")
		updateIns.Reason = dagIns.Reason
	}

	err := s.transaction(func(tx *gorm.DB) error {
		return tx.Model(dagIns).Select(updateFields).Updates(updateIns).Error
	})
	if err != nil {
		return fmt.Errorf("patch dagIns failed: %w", err)
	}
	goevent.Publish(&event.DagInstancePatched{
		Payload:         dagIns,
		MustPatchFields: mustsPatchFields,
	})
	return nil
}

func (s *Store) UpdateDag(dag *entity.Dag) error {
	// check task's connection
	_, err := mod.BuildRootNode(mod.MapTasksToGetter(dag.Tasks))
	if err != nil {
		return err
	}

	err = s.transaction(func(tx *gorm.DB) error {
		dag.BaseInfo.Update()
		return tx.Model(dag).Select("*").Updates(dag).Error
	})
	if err != nil {
		return fmt.Errorf("update dag failed: %w", err)
	}
	return nil
}

func (s *Store) UpdateDagIns(dagIns *entity.DagInstance) error {
	err := s.transaction(func(tx *gorm.DB) error {
		dagIns.BaseInfo.Update()
		return tx.Updates(dagIns).Error
	})
	if err != nil {
		return fmt.Errorf("update dagIns failed: %w", err)
	}
	goevent.Publish(&event.DagInstanceUpdated{Payload: dagIns})
	return nil
}

func (s *Store) UpdateTaskIns(taskIns *entity.TaskInstance) error {
	err := s.transaction(func(tx *gorm.DB) error {
		taskIns.BaseInfo.Update()
		return tx.Updates(taskIns).Error
	})
	if err != nil {
		return fmt.Errorf("update taskIns failed: %w", err)
	}
	return nil
}

func (s *Store) BatchUpdateDagIns(dagIns []*entity.DagInstance) (err error) {
	var anySlice []any
	for _, dag := range dagIns {
		anySlice = append(anySlice, dag)
	}

	return s.batchUpdate(anySlice, func(tx *gorm.DB, en any) error {
		dagInstance, ok := en.(*entity.DagInstance)
		if !ok {
			return fmt.Errorf("invalid entity type: %T", en)
		}
		dagInstance.BaseInfo.Update()
		return tx.Updates(dagInstance).Error
	})
}

func (s *Store) batchUpdate(entitys []any, cb func(tx *gorm.DB, en any) error) (err error) {
	ctx, cancel := context.WithTimeout(context.TODO(), s.opt.BatchUpdateConfig.Timeout)
	defer cancel()

	errs := &data.Errors{}
	errChan := make(chan error)
	defer func() {
		close(errChan)
		if errs.Len() > 0 {
			err = errs
		}
	}()

	go func() {
		for err := range errChan {
			errs.Append(err)
		}
	}()

	entityChunks := Chunk(entitys, s.opt.BatchUpdateConfig.ConcurrencyCount)
	var wg sync.WaitGroup
	for _, entityChunk := range entityChunks {
		wg.Add(len(entityChunk))
		for i := range entityChunk {
			go func(ctx context.Context, en any, ch chan error) {
				db := s.gormDB.WithContext(ctx)
				err := db.Transaction(func(tx *gorm.DB) error {
					return cb(tx, en)
				})
				if err != nil {
					errChan <- fmt.Errorf("batch update entity failed: %w", err)
				}
				wg.Done()
			}(ctx, entityChunk[i], errChan)
		}
		wg.Wait()
	}
	return
}

func (s *Store) BatchUpdateTaskIns(taskIns []*entity.TaskInstance) error {
	var anySlice []any
	for _, task := range taskIns {
		anySlice = append(anySlice, task)
	}

	return s.batchUpdate(anySlice, func(tx *gorm.DB, en any) error {
		taskInstance, ok := en.(*entity.TaskInstance)
		if !ok {
			return fmt.Errorf("invalid entity type: %T", en)
		}
		taskInstance.BaseInfo.Update()
		return tx.Updates(taskInstance).Error
	})
}

func (s *Store) GetTaskIns(taskInsId string) (*entity.TaskInstance, error) {
	taskIns := &entity.TaskInstance{}
	err := s.transaction(func(tx *gorm.DB) error {
		return tx.Where("id = ?", taskInsId).First(taskIns).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("key[ %s ] not found: %w", taskInsId, data.ErrDataNotFound)
		}
		log.Errorf("get task instance %s failed: %s", taskInsId, err)
		return nil, fmt.Errorf("get task instance failed: %w", err)
	}
	return taskIns, nil
}

func (s *Store) GetDag(dagId string) (*entity.Dag, error) {
	dag := &entity.Dag{}
	err := s.transaction(func(tx *gorm.DB) error {
		return tx.Where("id = ?", dagId).First(dag).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("key[ %s ] not found: %w", dagId, data.ErrDataNotFound)
		}
		log.Errorf("get dag %s failed: %s", dagId, err)
		return nil, fmt.Errorf("get dag failed: %w", err)
	}
	return dag, nil
}

func (s *Store) GetDagInstance(dagInsId string) (*entity.DagInstance, error) {
	dagIns := &entity.DagInstance{}
	err := s.transaction(func(tx *gorm.DB) error {
		return tx.Where("id = ?", dagInsId).First(dagIns).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("key[ %s ] not found: %w", dagInsId, data.ErrDataNotFound)
		}
		log.Errorf("get dag instance %s failed: %s", dagInsId, err)
		return nil, fmt.Errorf("get dag instance failed: %w", err)
	}
	return dagIns, nil
}

func (s *Store) ListDagInstance(input *mod.ListDagInstanceInput) ([]*entity.DagInstance, error) {
	if len(input.Tags) > 0 {
		return s.ListDagInstanceWithFilterTags(input)
	}
	return s.ListDagInstanceWithoutFilterTags(input)
}

func (s *Store) ListDagInstanceWithFilterTags(input *mod.ListDagInstanceInput) ([]*entity.DagInstance, error) {
	var ret []*entity.DagInstanceTag
	err := s.transaction(func(tx *gorm.DB) error {
		var queryParams [][]interface{}
		for k, v := range input.Tags {
			queryParams = append(queryParams, []interface{}{k, v})
		}
		return tx.Where("(`key`, `value`) IN ?", queryParams).
			Select("dag_ins_id, count(*) as total").
			Group("dag_ins_id").
			Having("total = ?", len(input.Tags)).
			Find(&ret).Error
	})
	if err != nil {
		log.Errorf("list dag instance tags input: %v, failed: %s", input, err)
		return nil, err
	}
	var dagInsIds []string
	for _, v := range ret {
		dagInsIds = append(dagInsIds, v.DagInsId)
	}
	if len(input.Ids) == 0 {
		input.Ids = dagInsIds
	} else {
		input.Ids = IntersectStringSlice(input.Ids, dagInsIds)
	}
	if len(input.Ids) == 0 {
		return nil, nil
	}

	return s.ListDagInstanceWithoutFilterTags(input)
}

func (s *Store) ListDagInstanceWithoutFilterTags(input *mod.ListDagInstanceInput) ([]*entity.DagInstance, error) {
	var ret []*entity.DagInstance
	err := s.transaction(func(tx *gorm.DB) error {
		if len(input.Status) > 0 {
			tx = tx.Where("status in (?)", input.Status)
		}
		if len(input.Ids) > 0 {
			tx = tx.Where("id in (?)", input.Ids)
		}
		if input.Worker != "" {
			tx = tx.Where("worker = ?", input.Worker)
		}
		if input.UpdatedEnd > 0 {
			tx = tx.Where("updated_at <= ?", input.UpdatedEnd)
		}
		if input.HasCmd {
			tx = tx.Where("cmd is not null")
		}
		if input.Limit > 0 {
			tx = tx.Limit(int(input.Limit))
		}
		return tx.Find(&ret).Error
	})
	if err != nil {
		log.Errorf("list dag instance input: %v, failed: %s", input, err)
		return nil, err
	}
	return ret, nil
}

func (s *Store) ListTaskInstance(input *mod.ListTaskInstanceInput) ([]*entity.TaskInstance, error) {
	var ret []*entity.TaskInstance
	err := s.transaction(func(tx *gorm.DB) error {
		if len(input.IDs) > 0 {
			tx = tx.Where("id in (?)", input.IDs)
		}
		if len(input.Status) > 0 {
			tx = tx.Where("status in (?)", input.Status)
		}
		if input.Expired {
			tx = tx.Where("(?) >= updated_at + timeout_secs", time.Now().Unix()-5)
		}
		if input.DagInsID != "" {
			tx = tx.Where("dag_ins_id = ?", input.DagInsID)
		}
		if len(input.SelectField) > 0 {
			tx = tx.Select(input.SelectField)
		}
		return tx.Find(&ret).Error
	})
	if err != nil {
		log.Errorf("list task instance input: %v, failed: %s", input, err)
		return nil, err
	}
	return ret, nil
}

// ListDag
func (s *Store) ListDag(input *mod.ListDagInput) ([]*entity.Dag, error) {
	var ret []*entity.Dag
	err := s.transaction(func(tx *gorm.DB) error {
		return tx.Find(&ret).Error
	})
	if err != nil {
		log.Errorf("list dag input: %v, failed: %s", input, err)
		return nil, err
	}
	return ret, nil
}

// BatchDeleteDag
func (s *Store) BatchDeleteDag(ids []string) error {
	err := s.transaction(func(tx *gorm.DB) error {
		return tx.Delete(&entity.Dag{}, "id in (?)", ids).Error
	})
	if err != nil {
		log.Errorf("delete dag input: %v, failed: %s", ids, err)
		return err
	}
	return nil
}

// BatchDeleteDagIns
func (s *Store) BatchDeleteDagIns(ids []string) error {
	err := s.transaction(func(tx *gorm.DB) error {
		return tx.Delete(&entity.DagInstance{}, "id in (?)", ids).Error
	})
	if err != nil {
		log.Errorf("delete dag instance input: %v, failed: %s", ids, err)
		return err
	}
	return nil
}

// BatchDeleteTaskIns
func (s *Store) BatchDeleteTaskIns(ids []string) error {
	err := s.transaction(func(tx *gorm.DB) error {
		return tx.Delete(&entity.TaskInstance{}, "id in (?)", ids).Error
	})
	if err != nil {
		log.Errorf("delete task instance input: %v, failed: %s", ids, err)
		return err
	}
	return nil
}

func (s *Store) Marshal(obj interface{}) ([]byte, error) {
	return json.Marshal(obj)
}

func (s *Store) Unmarshal(bytes []byte, ptr interface{}) error {
	return json.Unmarshal(bytes, ptr)
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

	db, err := gorm.Open(gormDriver.Open(s.opt.MySQLConfig.FormatDSN()), s.opt.GormConfig)
	if err != nil {
		return fmt.Errorf("connect to mysql occur error: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sqlDB error: %w", err)
	}

	sqlDB.SetConnMaxLifetime(s.opt.PoolConfig.ConnMaxLifetime)
	sqlDB.SetMaxIdleConns(s.opt.PoolConfig.MaxIdleConns)
	sqlDB.SetMaxOpenConns(s.opt.PoolConfig.MaxOpenConns)

	if s.opt.MigrationSwitch {
		err = db.AutoMigrate(&entity.Dag{}, &entity.DagInstance{}, &entity.DagInstanceTag{}, &entity.TaskInstance{})
		if err != nil {
			return err
		}
	}
	s.gormDB = db
	return nil
}

func (s *Store) readOpt() error {
	err := s.readMySQLConfigOpt()
	if err != nil {
		return err
	}

	s.readGormConfigOpt()
	s.readPoolConfigOpt()
	s.readBatchUpdateConfigOpt()
	if s.opt.Timeout == 0 {
		s.opt.Timeout = time.Second * 5
	}

	return nil
}

func (s *Store) readBatchUpdateConfigOpt() {
	if s.opt.BatchUpdateConfig == nil {
		s.opt.BatchUpdateConfig = &BatchUpdateOption{}
	}
	if s.opt.BatchUpdateConfig.ConcurrencyCount == 0 {
		s.opt.BatchUpdateConfig.ConcurrencyCount = 5
	}
	if s.opt.BatchUpdateConfig.Timeout == 0 {
		s.opt.BatchUpdateConfig.Timeout = time.Second * 40
	}
}

func (s *Store) readGormConfigOpt() {
	if s.opt.GormConfig == nil {
		s.opt.GormConfig = &gorm.Config{}
	}
}

func (s *Store) readPoolConfigOpt() {
	if s.opt.PoolConfig == nil {
		s.opt.PoolConfig = &ConnectionPoolOption{}
	}
	if s.opt.PoolConfig.MaxOpenConns == 0 {
		s.opt.PoolConfig.MaxOpenConns = 100
	}
	if s.opt.PoolConfig.MaxIdleConns == 0 {
		s.opt.PoolConfig.MaxIdleConns = 100
	}
	if s.opt.PoolConfig.ConnMaxLifetime == 0 {
		s.opt.PoolConfig.ConnMaxLifetime = time.Minute * 3
	}
}

func (s *Store) readMySQLConfigOpt() error {
	if s.opt.MySQLConfig == nil {
		return fmt.Errorf("mysql config cannot be empty")
	}

	if s.opt.MySQLConfig.Addr == "" {
		return fmt.Errorf("addr cannot be empty")
	}

	if s.opt.MySQLConfig.User == "" {
		return fmt.Errorf("user cannot be empty")
	}

	if s.opt.MySQLConfig.Passwd == "" {
		return fmt.Errorf("passwd cannot be empty")
	}

	if s.opt.MySQLConfig.DBName == "" {
		return fmt.Errorf("dbName cannot be empty")
	}

	if s.opt.MySQLConfig.Collation == "" {
		s.opt.MySQLConfig.Collation = "utf8mb4_unicode_ci"
	}

	if s.opt.MySQLConfig.Loc == nil {
		s.opt.MySQLConfig.Loc = time.UTC
	}

	if s.opt.MySQLConfig.MaxAllowedPacket == 0 {
		s.opt.MySQLConfig.MaxAllowedPacket = mysql.NewConfig().MaxAllowedPacket
	}

	s.opt.MySQLConfig.Net = "tcp"
	s.opt.MySQLConfig.AllowNativePasswords = true
	s.opt.MySQLConfig.CheckConnLiveness = true
	s.opt.MySQLConfig.ParseTime = true

	if s.opt.MySQLConfig.Timeout == 0 {
		s.opt.MySQLConfig.Timeout = 5 * time.Second
	}

	if s.opt.MySQLConfig.ReadTimeout == 0 {
		s.opt.MySQLConfig.ReadTimeout = 30 * time.Second
	}

	if s.opt.MySQLConfig.WriteTimeout == 0 {
		s.opt.MySQLConfig.WriteTimeout = 30 * time.Second
	}

	if s.opt.MySQLConfig.Params == nil {
		s.opt.MySQLConfig.Params = map[string]string{}
	}
	if _, ok := s.opt.MySQLConfig.Params["charset"]; !ok {
		s.opt.MySQLConfig.Params["charset"] = "utf8mb4"
	}
	return nil
}
