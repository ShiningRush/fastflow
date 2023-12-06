package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/shiningrush/fastflow/keeper"
	"github.com/shiningrush/fastflow/pkg/event"
	"github.com/shiningrush/fastflow/pkg/log"
	"github.com/shiningrush/fastflow/pkg/mod"
	"github.com/shiningrush/fastflow/store"
	"github.com/shiningrush/goevent"
	gormDriver "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const LeaderKey = "leader"

// Keeper mysql implement
type Keeper struct {
	opt    *KeeperOption
	gormDB *gorm.DB

	leaderFlag atomic.Value
	keyNumber  int

	wg            sync.WaitGroup
	firstInitWg   sync.WaitGroup
	initCompleted atomic.Value
	closeCh       chan struct{}
}

// KeeperOption
type KeeperOption struct {
	// Key the work key, must be the format like "xxxx-{{number}}", number is the code of worker
	Key string
	// mongo connection string
	MySQLConfig *mysql.Config
	GormConfig  *gorm.Config
	PoolConfig  *ConnectionPoolOption

	// UnhealthyTime default 5s, campaign and heartbeat time will be half of it
	UnhealthyTime time.Duration
	// Timeout default 2s
	Timeout time.Duration

	MigrationSwitch bool
	WatcherFlag     bool
}

type ConnectionPoolOption struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

// NewKeeper
func NewKeeper(opt *KeeperOption) *Keeper {
	k := &Keeper{
		opt:     opt,
		closeCh: make(chan struct{}),
	}
	k.leaderFlag.Store(false)
	k.initCompleted.Store(false)
	return k
}

// Init
func (k *Keeper) Init() error {
	if err := k.readOpt(); err != nil {
		return err
	}
	store.InitFlakeGenerator(uint16(k.WorkerNumber()))

	db, err := gorm.Open(gormDriver.Open(k.opt.MySQLConfig.FormatDSN()), k.opt.GormConfig)
	if err != nil {
		return fmt.Errorf("connect to mysql occur error: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sqlDB error: %w", err)
	}

	sqlDB.SetConnMaxLifetime(k.opt.PoolConfig.ConnMaxLifetime)
	sqlDB.SetMaxIdleConns(k.opt.PoolConfig.MaxIdleConns)
	sqlDB.SetMaxOpenConns(k.opt.PoolConfig.MaxOpenConns)

	if k.opt.MigrationSwitch {
		err = db.AutoMigrate(&Heartbeat{}, &Election{})
		if err != nil {
			return err
		}
	}
	k.gormDB = db
	if k.opt.WatcherFlag {
		return nil
	}

	k.firstInitWg.Add(2)

	k.wg.Add(1)
	go k.goElect()

	if err := k.initHeartBeat(); err != nil {
		return err
	}
	k.wg.Add(1)
	go k.goHeartBeat()

	k.firstInitWg.Wait()
	k.initCompleted.Store(true)
	return nil
}

func (k *Keeper) setLeaderFlag(isLeader bool) {
	k.leaderFlag.Store(isLeader)
	goevent.Publish(&event.LeaderChanged{
		IsLeader:  isLeader,
		WorkerKey: k.WorkerKey(),
	})
}

// IsLeader indicate the component if is leader node
func (k *Keeper) IsLeader() bool {
	return k.leaderFlag.Load().(bool)
}

// AliveNodes get all alive nodes
func (k *Keeper) AliveNodes() ([]string, error) {
	var heartbeats []Heartbeat
	err := k.transaction(func(tx *gorm.DB) error {
		log.Info("%v", time.Now().Add(-1*k.opt.UnhealthyTime))
		return tx.Where("updated_at > ?", time.Now().Add(-1*k.opt.UnhealthyTime)).Find(&heartbeats).Error
	})
	if err != nil {
		return nil, fmt.Errorf("find heartbeats failed: %w", err)
	}

	var aliveNodes []string
	for i := range heartbeats {
		aliveNodes = append(aliveNodes, heartbeats[i].WorkerKey)
	}
	return aliveNodes, nil
}

func (k *Keeper) transaction(cb func(tx *gorm.DB) error) error {
	ctx, cancel := context.WithTimeout(context.TODO(), k.opt.Timeout)
	defer cancel()

	db := k.gormDB.WithContext(ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		return cb(tx)
	})
}

// IsAlive check if a worker still alive
func (k *Keeper) IsAlive(workerKey string) (bool, error) {
	heartbeat := &Heartbeat{}
	err := k.transaction(func(tx *gorm.DB) error {
		return tx.Where("worker_key", workerKey).
			Where("updated_at > ?", time.Now().Add(-1*k.opt.UnhealthyTime)).
			Find(heartbeat).Error
	})

	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query mysql failed: %w", err)
	}
	return true, nil
}

// WorkerKey must match `xxxx-1` format
func (k *Keeper) WorkerKey() string {
	return k.opt.Key
}

// WorkerNumber get the key number of Worker key, if here is a WorkKey like `worker-1`, then it will return "1"
func (k *Keeper) WorkerNumber() int {
	return k.keyNumber
}

func (k *Keeper) NewMutex(key string) mod.DistributedMutex {
	panic("implement me")
}

// close component
func (k *Keeper) Close() {
	close(k.closeCh)
	k.wg.Wait()

	if k.leaderFlag.Load().(bool) {
		err := k.transaction(func(tx *gorm.DB) error {
			return tx.Delete(&Election{}, "id = ?", LeaderKey).Error
		})
		if err != nil {
			log.Errorf("deregister leader failed: %s", err)
		}
	}

	err := k.transaction(func(tx *gorm.DB) error {
		return tx.Delete(&Heartbeat{}, "worker_key = ?", k.WorkerKey()).Error
	})
	if err != nil {
		log.Errorf("deregister heart beat failed: %s", err)
	}

	sqlDB, err := k.gormDB.DB()
	if err != nil {
		log.Errorf("get store client failed: %s", err)
	}

	if err = sqlDB.Close(); err != nil {
		log.Errorf("close store client failed: %s", err)
	}
}

// this function is just for testing
func (k *Keeper) forceClose() {
	close(k.closeCh)
	k.wg.Wait()
}

func (k *Keeper) goElect() {
	timerCh := time.Tick(k.opt.UnhealthyTime / 2)
	closed := false
	for !closed {
		select {
		case <-k.closeCh:
			closed = true
		case <-timerCh:
			k.elect()
		}
	}
	k.wg.Done()
}

func (k *Keeper) elect() {
	if k.leaderFlag.Load().(bool) {
		if err := k.continueLeader(); err != nil {
			log.Errorf("continue leader failed: %s", err)
			k.setLeaderFlag(false)
			return
		}
	} else {
		if err := k.campaign(); err != nil {
			log.Errorf("campaign failed: %s", err)
			return
		}
	}

	if !k.initCompleted.Load().(bool) {
		k.firstInitWg.Done()
	}
}

func (k *Keeper) campaign() error {
	election := &Election{}
	err := k.transaction(func(tx *gorm.DB) error {
		return tx.Where("id = ?", LeaderKey).First(election).Error
	})
	if err == nil {
		if election.WorkerKey == k.WorkerKey() {
			k.setLeaderFlag(true)
			return nil
		}
		if election.UpdatedAt.Before(time.Now().Add(-1 * k.opt.UnhealthyTime)) {
			return k.transaction(func(tx *gorm.DB) error {
				update := tx.Model(&Election{}).
					Where("id = ?", LeaderKey).
					Where("worker_key = ?", election.WorkerKey).
					Updates(map[string]interface{}{
						"worker_key": k.WorkerKey(),
						"updated_at": time.Now(),
					})
				if update.Error != nil {
					log.Errorf("update failed: %s", update.Error)
					return fmt.Errorf("update failed: %w", update.Error)
				}
				if update.RowsAffected > 0 {
					k.setLeaderFlag(true)
				}
				return nil
			})
		}
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		err := k.transaction(func(tx *gorm.DB) error {
			election := &Election{
				ID:        LeaderKey,
				WorkerKey: k.WorkerKey(),
				UpdatedAt: time.Now(),
			}
			return tx.Create(election).Error
		})
		if err != nil {
			if err == gorm.ErrDuplicatedKey {
				log.Infof("campaign failed")
				return nil
			}
			log.Errorf("insert campaign rec failed: %s", err)
			return fmt.Errorf("insert failed: %w", err)
		}
		k.setLeaderFlag(true)
		return nil
	}

	return fmt.Errorf("query leader failed: %w", err)
}

func (k *Keeper) continueLeader() error {
	return k.transaction(func(tx *gorm.DB) error {
		update := tx.Model(&Election{}).Where("id = ?", LeaderKey).Update("updated_at", time.Now())
		if update.Error != nil {
			log.Errorf("update failed: %s", update.Error)
			return fmt.Errorf("update failed: %w", update.Error)
		}
		if update.RowsAffected == 0 {
			log.Errorf("re-elected failed")
			return fmt.Errorf("re-elected failed")
		}
		return nil
	})
}

func (k *Keeper) goHeartBeat() {
	timerCh := time.Tick(k.opt.UnhealthyTime / 2)
	closed := false
	for !closed {
		select {
		case <-k.closeCh:
			closed = true
		case <-timerCh:
			if err := k.heartBeat(); err != nil {
				k.queryGormStats()
				log.Errorf("heart beat failed: %s", err)
				continue
			}
		}
		if !k.initCompleted.Load().(bool) {
			k.firstInitWg.Done()
		}
	}
	k.wg.Done()
}

func (k *Keeper) heartBeat() error {
	err := k.transaction(func(tx *gorm.DB) error {
		return tx.Where("worker_key = ?", k.WorkerKey()).Update("updated_at", time.Now()).Error
	})
	if err != nil {
		return fmt.Errorf("update hearbeat failed: %w", err)
	}
	return nil
}

func (k *Keeper) queryGormStats() {
	tx, err := k.gormDB.DB()
	if err != nil {
		log.Errorf("get store client failed: %s", err)
	} else {
		bytes, err := json.Marshal(tx.Stats())
		if err != nil {
			log.Errorf("marshal stats failed: %s", err)
		}
		log.Info("stats: %s", string(bytes))
	}
}

func (k *Keeper) initHeartBeat() error {
	err := k.transaction(func(tx *gorm.DB) error {
		err := tx.Delete(&Heartbeat{}, "worker_key = ?", k.WorkerKey()).Error
		if err != nil {
			return err
		}
		heartbeat := Heartbeat{
			WorkerKey: k.WorkerKey(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		return tx.Create(&heartbeat).Error
	})
	if err != nil {
		return fmt.Errorf("init hearbeat failed: %w", err)
	}
	return nil
}

func (k *Keeper) readOpt() error {
	if k.opt.Key == "" {
		return fmt.Errorf("worker key  can not be empty")
	}
	if k.opt.UnhealthyTime == 0 {
		k.opt.UnhealthyTime = time.Second * 5
	}
	if k.opt.Timeout == 0 {
		k.opt.Timeout = time.Second * 2
	}
	number, err := keeper.CheckWorkerKey(k.opt.Key)
	if err != nil {
		return err
	}
	k.keyNumber = number

	err = k.readMySQLConfigOpt()
	if err != nil {
		return err
	}
	k.readGormConfigOpt()
	k.readPoolConfigOpt()
	return nil
}

func (k *Keeper) readGormConfigOpt() {
	if k.opt.GormConfig == nil {
		k.opt.GormConfig = &gorm.Config{}
	}
}

func (k *Keeper) readPoolConfigOpt() {
	if k.opt.PoolConfig == nil {
		k.opt.PoolConfig = &ConnectionPoolOption{}
	}
	if k.opt.PoolConfig.MaxOpenConns == 0 {
		k.opt.PoolConfig.MaxOpenConns = 10
	}
	if k.opt.PoolConfig.MaxIdleConns == 0 {
		k.opt.PoolConfig.MaxIdleConns = 15
	}
	if k.opt.PoolConfig.ConnMaxLifetime == 0 {
		k.opt.PoolConfig.ConnMaxLifetime = time.Minute * 3
	}
}

func (k *Keeper) readMySQLConfigOpt() error {
	if k.opt.MySQLConfig == nil {
		return fmt.Errorf("mysql config cannot be empty")
	}

	if k.opt.MySQLConfig.Addr == "" {
		return fmt.Errorf("addr cannot be empty")
	}

	if k.opt.MySQLConfig.User == "" {
		return fmt.Errorf("user cannot be empty")
	}

	if k.opt.MySQLConfig.Passwd == "" {
		return fmt.Errorf("passwd cannot be empty")
	}

	if k.opt.MySQLConfig.DBName == "" {
		return fmt.Errorf("dbName cannot be empty")
	}

	if k.opt.MySQLConfig.Collation == "" {
		k.opt.MySQLConfig.Collation = "utf8mb4_unicode_ci"
	}

	if k.opt.MySQLConfig.Loc == nil {
		k.opt.MySQLConfig.Loc = time.UTC
	}

	if k.opt.MySQLConfig.MaxAllowedPacket == 0 {
		k.opt.MySQLConfig.MaxAllowedPacket = mysql.NewConfig().MaxAllowedPacket
	}

	k.opt.MySQLConfig.Net = "tcp"
	k.opt.MySQLConfig.AllowNativePasswords = true
	k.opt.MySQLConfig.CheckConnLiveness = true
	k.opt.MySQLConfig.ParseTime = true

	if k.opt.MySQLConfig.Timeout == 0 {
		k.opt.MySQLConfig.Timeout = 5 * time.Second
	}

	if k.opt.MySQLConfig.ReadTimeout == 0 {
		k.opt.MySQLConfig.ReadTimeout = 30 * time.Second
	}

	if k.opt.MySQLConfig.WriteTimeout == 0 {
		k.opt.MySQLConfig.WriteTimeout = 30 * time.Second
	}

	if k.opt.MySQLConfig.Params == nil {
		k.opt.MySQLConfig.Params = map[string]string{}
	}
	if _, ok := k.opt.MySQLConfig.Params["charset"]; !ok {
		k.opt.MySQLConfig.Params["charset"] = "utf8mb4"
	}
	return nil
}
