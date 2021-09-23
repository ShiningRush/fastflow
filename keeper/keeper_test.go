package keeper

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCheckWorkerKey(t *testing.T) {
	tests := []struct {
		giveKey string
		wantRet int
		wantErr error
	}{
		{
			giveKey: "fastflow-15",
			wantRet: 15,
		},
		{
			giveKey: "fastflow-256",
			wantErr: errors.New("worker number must in range 0~255"),
		},
	}

	for _, tc := range tests {
		ret, err := CheckWorkerKey(tc.giveKey)
		assert.Equal(t, tc.wantErr, err)
		assert.Equal(t, tc.wantRet, ret)
	}
}
