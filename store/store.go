package store

import (
	"strconv"
	"sync"

	"github.com/sony/sonyflake"
)

var (
	generator *sonyflake.Sonyflake
	mutex     sync.Mutex
)

// InitFlakeGenerator
func InitFlakeGenerator(machineId uint16) {
	mutex.Lock()
	defer mutex.Unlock()

	if generator != nil {
		return
	}

	generator = sonyflake.NewSonyflake(sonyflake.Settings{
		MachineID: func() (uint16, error) {
			return machineId, nil
		},
	})
}

// NextID
func NextID() uint64 {
	id, err := generator.NextID()
	if err != nil {
		panic(err)
	}
	return id
}

// NextStringID
func NextStringID() string {
	return strconv.FormatUint(NextID(), 10)
}
