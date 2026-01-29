package keeper

import (
	"fmt"
	"math"
	"regexp"
	"strconv"

	"github.com/shiningrush/fastflow/store"
)

var reg = regexp.MustCompile(`^.+-(\d+)$`)

// CheckWorkerKey check if worker key is correct, the format must be "xxxxx-{{number}}"
// number must in range 0~65535(sonyflake machine id is uint16)
// if you give a custom generator, we will not check worker key format
func CheckWorkerKey(key string) (int, error) {
	if store.IsCustomGenerator() {
		return 0, nil
	}

	ret := reg.FindStringSubmatch(key)
	if ret == nil {
		return 0, fmt.Errorf("worker key format is incorrect, must like 'xxx-1 or xxx-2'")
	}

	number, err := strconv.Atoi(ret[1])
	if err != nil {
		return 0, fmt.Errorf("convert number failed: %w", err)
	}

	if number < 0 || number > math.MaxUint16 {
		return 0, fmt.Errorf("worker number must in range 0~%d", math.MaxUint16)
	}

	return number, nil
}
