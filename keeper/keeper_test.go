package keeper

import (
	"errors"
	"testing"

	"github.com/shiningrush/fastflow/store"

	"github.com/stretchr/testify/assert"
)

type MockGenerator struct{}

func (m *MockGenerator) NextID() (uint64, error) {
	return 456, nil
}

func TestCheckWorkerKey(t *testing.T) {
	tests := []struct {
		name       string
		giveKey    string
		giveCusGen bool
		wantRet    int
		wantErr    error
	}{
		{
			name:    "check worker key min",
			giveKey: "fastflow-0",
			wantRet: 0,
		},
		{
			name:    "check worker key max",
			giveKey: "fastflow-65535",
			wantRet: 65535,
		},
		{
			name:    "check worker key overflow max",
			giveKey: "fastflow-65536",
			wantErr: errors.New("worker number must in range 0~65535"),
		},
		{
			name:    "wrong worker key",
			giveKey: "fastflow",
			wantErr: errors.New("worker key format is incorrect, must like 'xxx-1 or xxx-2'"),
		},
		{
			name:       "custom generator",
			giveCusGen: true,
			giveKey:    "fastflow",
			wantRet:    -1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.giveCusGen {
				store.InitCustomGenerator(&MockGenerator{})
			}

			ret, err := CheckWorkerKey(tc.giveKey)
			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantRet, ret)
		})
	}
}
