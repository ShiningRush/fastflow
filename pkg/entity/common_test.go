package entity

import (
	"context"
	"testing"

	"github.com/shiningrush/fastflow/store"
	"github.com/stretchr/testify/assert"
)

func TestBaseInfo_Initial(t *testing.T) {
	store.InitFlakeGenerator(100)
	bi := &BaseInfo{}
	bi.Initial()
	assert.NotEmpty(t, bi.ID)
	assert.NotZero(t, bi.CreatedAt)
	assert.NotZero(t, bi.UpdatedAt)

	bi = &BaseInfo{ID: "test"}
	bi.Initial()
	assert.Equal(t, "test", bi.ID)
	assert.NotZero(t, bi.CreatedAt)
	assert.NotZero(t, bi.UpdatedAt)
}

func TestBaseInfo_Update(t *testing.T) {
	bi := &BaseInfo{}
	bi.Update()
	assert.Zero(t, bi.CreatedAt)
	assert.NotZero(t, bi.UpdatedAt)
}

func TestCtxVar(t *testing.T) {
	ctx := context.TODO()
	taskIns, ok := CtxRunningTaskIns(ctx)
	assert.False(t, ok)
	assert.Zero(t, taskIns)

	storeTaskIns := &TaskInstance{BaseInfo: BaseInfo{ID: "2"}}

	ctx = CtxWithRunningTaskIns(ctx, storeTaskIns)
	taskIns, ok = CtxRunningTaskIns(ctx)
	assert.True(t, ok)
	assert.Equal(t, storeTaskIns, taskIns)
}
