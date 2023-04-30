package store

import (
	"strconv"

	"github.com/sony/sonyflake"
)

var generator *sonyflake.Sonyflake

// InitFlakeGenerator
func InitFlakeGenerator(machineId uint16) {
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
