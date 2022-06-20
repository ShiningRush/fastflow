package render

import "fmt"

type TplOptions struct {
	MissKeyStrategy MissKeyStrategy
}

func NewOptions(opts ...TplOption) *TplOptions {
	o := &TplOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

type TplOption func(options *TplOptions)

func WithMissKeyStrategy(st MissKeyStrategy) TplOption {
	return func(options *TplOptions) {
		options.MissKeyStrategy = st
	}
}

type MissKeyStrategy string

const (
	// MissKeyStrategyDefault The default behavior: Do nothing and continue execution.
	//		If printed, the result of the index operation is the string
	//		"<no value>".
	MissKeyStrategyDefault MissKeyStrategy = "default"
	// MissKeyStrategyInvalid = MissKeyStrategyDefault
	MissKeyStrategyInvalid MissKeyStrategy = "invalid"
	// MissKeyStrategyZero  returns the zero value for the map type's element.
	// returns <no value> if use map[string]interface{}
	MissKeyStrategyZero MissKeyStrategy = "zero"
	// MissKeyStrategyError Execution stops immediately with an error.
	MissKeyStrategyError MissKeyStrategy = "error"
)

func (m MissKeyStrategy) OptionString() string {
	if m == "" {
		m = MissKeyStrategyInvalid
	}
	return fmt.Sprintf("missingkey=%s", m)
}

func (m MissKeyStrategy) Effective() bool {
	return m != ""
}
