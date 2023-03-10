package keeper

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
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
