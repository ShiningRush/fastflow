package store

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/sony/sonyflake"
)

var (
	generator           Generator
	customGeneratorFlag bool
	mutex               sync.Mutex
)

type Generator interface {
	NextID() (uint64, error)
}

// InitCustomGenerator
// if you give a custom generator, it will be used to generate id and not check worker number format anymore
func InitCustomGenerator(cusGen Generator) {
	// check custom generator is valid
	_, err := cusGen.NextID()
	if err != nil {
		panic(fmt.Errorf("failed to generate custom generator id: %v", err))
	}

	generator = cusGen
	customGeneratorFlag = true
}

// IsCustomGenerator returns whether the custom generator is initialized.
func IsCustomGenerator() bool {
	return customGeneratorFlag
}

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
