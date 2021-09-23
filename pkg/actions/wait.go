package actions

import (
	"fmt"
	"github.com/shiningrush/fastflow/pkg/entity/run"
	"regexp"
	"strconv"
	"time"
)

const (
	ActionKeyWait = "ff-waiting"
)

// WaitingParams
type WaitingParams struct {
	// Support "d|h|m|s|ms" such as 1h mean 1hours
	WaitingTime string `json:"waitingTime"`
}

// Waiting action
type Waiting struct {
}

// Name
func (s *Waiting) Name() string {
	return ActionKeyWait
}

// ParameterNew
func (s *Waiting) ParameterNew() interface{} {
	return &WaitingParams{}
}

// Run
func (s *Waiting) Run(ctx run.ExecuteContext, params interface{}) error {
	p := params.(*WaitingParams)
	d, err := ParseDuration(p.WaitingTime)
	if err != nil {
		return err
	}

	tc := time.NewTimer(d).C

	select {
	case <-tc:
	case <-ctx.Context().Done():
		return fmt.Errorf("context deadlined")
	}
	return nil
}

var durationRE = regexp.MustCompile("^([0-9]+)(d|h|m|s|ms)$")

// ParseDuration
func ParseDuration(durationStr string) (time.Duration, error) {
	matches := durationRE.FindStringSubmatch(durationStr)
	if len(matches) != 3 {
		return 0, fmt.Errorf("not a valid duration string: %q", durationStr)
	}
	var (
		n, _ = strconv.Atoi(matches[1])
		dur  = time.Duration(n) * time.Millisecond
	)
	switch unit := matches[2]; unit {
	case "d":
		dur *= 1000 * 60 * 60 * 24
	case "h":
		dur *= 1000 * 60 * 60
	case "m":
		dur *= 1000 * 60
	case "s":
		dur *= 1000
	case "ms":
	default:
		return 0, fmt.Errorf("invalid time unit in duration string: %q", unit)
	}
	return dur, nil
}
