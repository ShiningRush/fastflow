package keeper

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckWorkerKey(t *testing.T) {
	tests := []struct {
		name    string
		giveKey string
		wantRet int
		wantErr error
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ret, err := CheckWorkerKey(tc.giveKey)
			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantRet, ret)
		})
	}
}
