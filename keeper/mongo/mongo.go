package mongo

import (
	"context"
	"errors"
	"fmt"
	"github.com/shiningrush/fastflow/keeper"
	"github.com/shiningrush/fastflow/pkg/event"
	"github.com/shiningrush/fastflow/pkg/log"
	"github.com/shiningrush/fastflow/pkg/mod"
	"github.com/shiningrush/fastflow/store"
	"github.com/shiningrush/goevent"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"sync"
	"sync/atomic"
	"time"
)

const LeaderKey = "leader"

// Keeper mongo implement
type Keeper struct {
	opt              *KeeperOption
	leaderClsName    string
	heartbeatClsName string
	mutexClsName     string

	leaderFlag  atomic.Value
	keyNumber   int
	mongoClient *mongo.Client
	mongoDb     *mongo.Database

	wg            sync.WaitGroup
	firstInitWg   sync.WaitGroup
	initCompleted bool
	closeCh       chan struct{}
}

// KeeperOption
type KeeperOption struct {
	// Key the work key, must be the format like "xxxx-{{number}}", number is the code of worker
	Key string
	// mongo connection string
	ConnStr  string
	Database string
	// the prefix will append to the database
	Prefix string
	// UnhealthyTime default 5s, campaign and heartbeat time will be half of it
	UnhealthyTime time.Duration
	// Timeout default 2s
	Timeout time.Duration
}

// NewKeeper
func NewKeeper(opt *KeeperOption) *Keeper {
	k := &Keeper{
		opt:     opt,
		closeCh: make(chan struct{}),
	}
	k.leaderFlag.Store(false)
	return k
}

// Init
func (k *Keeper) Init() error {
	if err := k.readOpt(); err != nil {
		return err
	}
	store.InitFlakeGenerator(uint16(k.WorkerNumber()))

	ctx, cancel := context.WithTimeout(context.Background(), k.opt.Timeout)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(k.opt.ConnStr))
	if err != nil {
		return fmt.Errorf("connect client failed: %w", err)
	}
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return fmt.Errorf("ping client failed: %w", err)
	}
	k.mongoClient = client
	k.mongoDb = k.mongoClient.Database(k.opt.Database)
	if err := k.ensureTtlIndex(ctx, k.leaderClsName, "updatedAt", int32(k.opt.UnhealthyTime.Seconds())); err != nil {
		return err
	}
	if err := k.ensureTtlIndex(ctx, k.heartbeatClsName, "updatedAt", int32(k.opt.UnhealthyTime.Seconds())); err != nil {
		return err
	}
	if err := k.ensureTtlIndex(ctx, k.mutexClsName, "expiredAt", 1); err != nil {
		return err
	}

	k.firstInitWg.Add(2)

	k.wg.Add(1)
	go k.goElect()
	k.wg.Add(1)
	go k.goHeartBeat()

	k.firstInitWg.Wait()
	k.initCompleted = true
	return nil
}

func (k *Keeper) setLeaderFlag(isLeader bool) {
	k.leaderFlag.Store(isLeader)
	goevent.Publish(&event.LeaderChanged{
		IsLeader:  isLeader,
		WorkerKey: k.WorkerKey(),
	})
}

func (k *Keeper) ensureTtlIndex(ctx context.Context, clsName, field string, ttl int32) error {
	if _, err := k.mongoDb.Collection(clsName).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{
			field: 1,
		},
		Options: &options.IndexOptions{
			ExpireAfterSeconds: &ttl,
		},
	}); err != nil {
		isDiffOptErr := false
		if driverErr, ok := err.(driver.Error); ok {
			// index already existed and option is different
			if driverErr.Code == 85 {
				isDiffOptErr = true
			}
		}
		if !isDiffOptErr {
			return fmt.Errorf("create index failed: %w", err)
		}

		_, err := k.mongoDb.Collection(clsName).Indexes().DropAll(ctx)
		if err != nil {
			return fmt.Errorf("drop all index failed: %w", err)
		}
		return k.ensureTtlIndex(ctx, clsName, field, ttl)
	}
	return nil
}

func (k *Keeper) readOpt() error {
	if k.opt.Key == "" || k.opt.ConnStr == "" {
		return fmt.Errorf("worker key or connection string can not be empty")
	}

	if k.opt.Database == "" {
		k.opt.Database = "fastflow"
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

	k.leaderClsName = "election"
	k.heartbeatClsName = "heartbeat"
	k.mutexClsName = "mutex"
	if k.opt.Prefix != "" {
		k.leaderClsName = fmt.Sprintf("%s_%s", k.opt.Prefix, k.leaderClsName)
		k.heartbeatClsName = fmt.Sprintf("%s_%s", k.opt.Prefix, k.heartbeatClsName)
		k.mutexClsName = fmt.Sprintf("%s_%s", k.opt.Prefix, k.mutexClsName)
	}

	return nil
}

// IsLeader indicate the component if is leader node
func (k *Keeper) IsLeader() bool {
	return k.leaderFlag.Load().(bool)
}

// AliveNodes get all alive nodes
func (k *Keeper) AliveNodes() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), k.opt.Timeout)
	defer cancel()
	// mongodb background worker delete expired date every 60s, so can not believe it
	cur, err := k.mongoDb.Collection(k.heartbeatClsName).Find(ctx, bson.M{
		"updatedAt": bson.M{
			"$gt": time.Now().Add(-1 * k.opt.UnhealthyTime),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("find result failed: %w", err)
	}

	var ret []Payload
	if err := cur.All(ctx, &ret); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	var aliveNodes []string
	for i := range ret {
		aliveNodes = append(aliveNodes, ret[i].WorkerKey)
	}
	return aliveNodes, nil
}

// IsAlive check if a worker still alive
func (k *Keeper) IsAlive(workerKey string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), k.opt.Timeout)
	defer cancel()

	var p Payload
	// mongodb background worker delete expired date every 60s, so can not believe it
	err := k.mongoDb.Collection(k.heartbeatClsName).FindOne(ctx, bson.M{
		"_id": workerKey,
		"updatedAt": bson.M{
			"$gt": time.Now().Add(-1 * k.opt.UnhealthyTime),
		},
	}).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query mongo failed: %w", err)
	}
	return true, nil
}

// WorkerKey must match `xxxx-1` format
func (k *Keeper) WorkerKey() string {
	return k.opt.Key
}

// WorkerNumber get the the key number of Worker key, if here is a WorkKey like `worker-1`, then it will return "1"
func (k *Keeper) WorkerNumber() int {
	return k.keyNumber
}

// NewMutex(key string) create a new distributed mutex
func (k *Keeper) NewMutex(key string) mod.DistributedMutex {
	return &MongoMutex{
		key:     key,
		clsName: k.mutexClsName,
		mongoDb: k.mongoDb,
	}
}

// close component
func (k *Keeper) Close() {
	close(k.closeCh)
	k.wg.Wait()

	ctx, cancel := context.WithTimeout(context.TODO(), k.opt.Timeout)
	defer cancel()
	if k.leaderFlag.Load().(bool) {
		_, err := k.mongoDb.Collection(k.leaderClsName).DeleteOne(ctx, bson.M{
			"_id": LeaderKey,
		})
		if err != nil {
			log.Errorf("deregister leader failed: %s", err)
		}
	}

	_, err := k.mongoDb.Collection(k.heartbeatClsName).DeleteOne(ctx, bson.M{
		"_id": k.opt.Key,
	})
	if err != nil {
		log.Errorf("deregister heart beat failed: %s", err)
	}

	err = k.mongoClient.Disconnect(ctx)
	if err != nil {
		log.Errorf("close keeper client failed: %s", err)
	}
}

// this function is just for testing
func (k *Keeper) forceClose() {
	close(k.closeCh)
	k.wg.Wait()
}

// Payload header beat dto
type Payload struct {
	WorkerKey string    `bson:"_id"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

// LeaderPayload leader election dto
type LeaderPayload struct {
	ID        string    `bson:"_id"`
	WorkerKey string    `bson:"workerKey"`
	UpdatedAt time.Time `bson:"updatedAt"`
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

	if !k.initCompleted {
		k.firstInitWg.Done()
	}
}

func (k *Keeper) campaign() error {
	ctx, cancel := context.WithTimeout(context.TODO(), k.opt.Timeout)
	defer cancel()
	cur, err := k.mongoDb.Collection(k.leaderClsName).Find(ctx, bson.D{})
	if err != nil {
		return fmt.Errorf("find data failed: %w", err)
	}
	var ret []LeaderPayload
	if err := cur.All(ctx, &ret); err != nil {
		return fmt.Errorf("decode data failed: %w", err)
	}

	if len(ret) > 0 {
		if ret[0].WorkerKey == k.opt.Key {
			k.setLeaderFlag(true)
			return nil
		}
		if ret[0].UpdatedAt.Before(time.Now().Add(-1 * k.opt.UnhealthyTime)) {
			ret, err := k.mongoDb.Collection(k.leaderClsName).UpdateOne(ctx,
				bson.M{
					"_id":       LeaderKey,
					"workerKey": ret[0].WorkerKey,
				},
				bson.M{
					"$set": bson.M{
						"workerKey": k.opt.Key,
						"updatedAt": time.Now(),
					},
				})
			if err != nil {
				return fmt.Errorf("update failed: %w", err)

			}
			if ret.ModifiedCount > 0 {
				k.setLeaderFlag(true)
			}
		}
	}
	if len(ret) == 0 {
		_, err := k.mongoDb.Collection(k.leaderClsName).InsertOne(ctx,
			LeaderPayload{
				ID:        LeaderKey,
				WorkerKey: k.opt.Key,
				UpdatedAt: time.Now(),
			})
		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				log.Infof("campaign failed")
				return nil
			}
			log.Errorf("insert campaign rec failed: %s", err)
			return fmt.Errorf("insert failed: %w", err)
		}
		k.setLeaderFlag(true)
	}
	return nil
}

func (k *Keeper) continueLeader() error {
	ctx, cancel := context.WithTimeout(context.TODO(), k.opt.Timeout)
	defer cancel()
	ret, err := k.mongoDb.Collection(k.leaderClsName).UpdateOne(ctx, bson.M{
		"_id":       LeaderKey,
		"workerKey": k.opt.Key,
	},
		bson.M{
			"$set": bson.M{
				"updatedAt": time.Now(),
			},
		})
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	if ret.MatchedCount == 0 {
		return fmt.Errorf("re-elected failed")
	}
	return nil
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
				log.Errorf("heart beat failed: %s", err)
				continue
			}
		}
		if !k.initCompleted {
			k.firstInitWg.Done()
		}
	}
	k.wg.Done()
}

func (k *Keeper) heartBeat() error {
	ctx, cancel := context.WithTimeout(context.TODO(), k.opt.Timeout)
	defer cancel()
	_, err := k.mongoDb.Collection(k.heartbeatClsName).UpdateOne(ctx,
		bson.M{
			"_id": k.opt.Key,
		},
		bson.M{
			"$set": bson.M{
				"updatedAt": time.Now(),
			},
		},
		&options.UpdateOptions{
			Upsert: boolPtr(true),
		})
	if err != nil {
		return fmt.Errorf("update mongo failed: %w", err)
	}
	return nil
}

func boolPtr(b bool) *bool {
	return &b
}
