// +build integration

package mongo

import (
	"context"
	"github.com/shiningrush/fastflow/pkg/mod"
	"github.com/shiningrush/fastflow/pkg/utils/data"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"time"
)

func TestMongoMutex_Sanity(t *testing.T) {
	wk := initWorker(t, "worker-1")
	mux := wk.NewMutex("key")
	mux2 := wk.NewMutex("key")
	err := mux.Lock(context.TODO())
	assert.NoError(t, err)
	log.Println("m1 locked")
	go func() {
		time.Sleep(2 * time.Second)
		err := mux.Unlock(context.TODO())
		assert.NoError(t, err)
		log.Println("m1 unlocked")
	}()

	err = mux2.Lock(context.TODO())
	assert.NoError(t, err)
	log.Println("m2 locked")

	err = mux2.Unlock(context.TODO())
	assert.NoError(t, err)
	log.Println("m2 unlocked")

	// race lock
	err = mux.Lock(context.TODO(), mod.LockTTL(time.Second*2))
	assert.NoError(t, err)
	log.Println("m1 race locked")

	err = mux2.Lock(context.TODO())
	assert.NoError(t, err)
	log.Println("m2 race locked")

	err = mux.Unlock(context.TODO())
	assert.Equal(t, data.ErrMutexAlreadyUnlock, err)

	err = mux2.Unlock(context.TODO())
	assert.NoError(t, err)

	// reentrant
	err = mux.Lock(context.TODO(), mod.Reentrant("m1"))
	assert.NoError(t, err)
	log.Println("m1 locked")

	err = mux.Lock(context.TODO(), mod.Reentrant("m1"))
	assert.NoError(t, err)
	log.Println("m1 reentrant locked")

	err = mux.Unlock(context.TODO())
	assert.NoError(t, err)
}
