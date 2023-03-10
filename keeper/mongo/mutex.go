package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/shiningrush/fastflow/pkg/mod"
	"github.com/shiningrush/fastflow/pkg/utils/data"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type LockDetail struct {
	Key       string    `bson:"_id"`
	ExpiredAt time.Time `bson:"expiredAt"`
	Identity  string    `bson:"identity"`
}

type MongoMutex struct {
	key string

	clsName    string
	mongoDb    *mongo.Database
	lockDetail *LockDetail
}

func (m *MongoMutex) Lock(ctx context.Context, ops ...mod.LockOptionOp) error {
	opt := mod.NewLockOption(ops)
	if err := m.spinLock(ctx, opt); err != nil {
		return err
	}
	// already keep lock
	if m.lockDetail != nil {
		return nil
	}

	// when get lock failed, loop to get it
	ticker := time.NewTicker(opt.SpinInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := m.spinLock(ctx, opt); err != nil {
				return nil
			}
			// already keep lock
			if m.lockDetail != nil {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (m *MongoMutex) spinLock(ctx context.Context, opt *mod.LockOption) error {
	detail := LockDetail{}
	err := m.mongoDb.Collection(m.clsName).FindOne(ctx, bson.M{"_id": m.key}).Decode(&detail)
	if err != nil && err != mongo.ErrNoDocuments {
		return fmt.Errorf("get lock detail failed: %w", err)
	}

	// no lock
	if err == mongo.ErrNoDocuments {
		d := &LockDetail{
			Key:       m.key,
			ExpiredAt: time.Now().Add(opt.TTL),
			Identity:  opt.ReentrantIdentity,
		}
		_, err := m.mongoDb.Collection(m.clsName).InsertOne(ctx, d)
		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				// race lock failed, ready to get lock next time
				return nil
			}
			return fmt.Errorf("insert lock detail failed: %w", err)
		}
		m.lockDetail = d
		return nil
	}

	// lock existed, we should check it is expired
	if detail.ExpiredAt.Before(time.Now()) {
		exp := time.Now().Add(opt.TTL)
		ret, err := m.mongoDb.Collection(m.clsName).UpdateOne(ctx, bson.M{"_id": m.key, "expiredAt": detail.ExpiredAt}, bson.M{
			"$set": bson.M{
				"expiredAt": time.Now().Add(opt.TTL),
				"identity":  opt.ReentrantIdentity,
			},
		})
		if err != nil {
			return fmt.Errorf("get lock failed: %w", err)
		}
		// lock is keep by others
		if ret.ModifiedCount == 0 {
			return nil
		}
		detail.ExpiredAt = exp
		m.lockDetail = &detail
		return nil
	}

	// lock existed, we should check it is reentrant
	if opt.ReentrantIdentity != "" && detail.Identity == opt.ReentrantIdentity {
		m.lockDetail = &detail
		return nil
	}

	// lock is keep by others, return to loop
	return nil
}

//
func (m *MongoMutex) Unlock(ctx context.Context) error {
	if m.lockDetail == nil {
		return fmt.Errorf("the mutex is not locked")
	}

	ret, err := m.mongoDb.Collection(m.clsName).DeleteOne(ctx, bson.M{"_id": m.key, "expiredAt": m.lockDetail.ExpiredAt})
	if err != nil {
		return fmt.Errorf("delete lock detail failed: %w", err)
	}

	if ret.DeletedCount == 0 {
		return data.ErrMutexAlreadyUnlock
	}
	m.lockDetail = nil
	return nil
}
