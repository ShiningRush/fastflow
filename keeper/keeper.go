package keeper

import (
	"fmt"
	"regexp"
	"strconv"
)

var reg = regexp.MustCompile(`^.+-(\d+)$`)

// CheckWorkerKey check if worker key is correct, the format must be "xxxxx-{{number}}"
// number must in range 0~255
// if key is correct, worker number
func CheckWorkerKey(key string) (int, error) {
	ret := reg.FindStringSubmatch(key)
	if ret == nil {
		return 0, fmt.Errorf("worker key format is incorrect, must like 'xxx-1 or xxx-2'")
	}

	number, err := strconv.Atoi(ret[1])
	if err != nil {
		return 0, fmt.Errorf("convert number failed: %w", err)
	}

	if number < 0 || number > 255 {
		return 0, fmt.Errorf("worker number must in range 0~255")
	}

	return number, nil
}
