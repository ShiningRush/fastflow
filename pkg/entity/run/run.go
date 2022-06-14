package run

import (
	"context"
	"fmt"

	"github.com/shiningrush/fastflow/pkg/utils"
)

// NewDefExecuteContext
func NewDefExecuteContext(
	ctx context.Context,
	op ShareDataOperator,
	trace func(msg string, opt ...TraceOp),
	dagVars utils.KeyValueGetter,
	varsIterator utils.KeyValueIterator,
) *DefExecuteContext {
	return &DefExecuteContext{
		ctx:          ctx,
		op:           op,
		trace:        trace,
		varsGetter:   dagVars,
		varsIterator: varsIterator,
	}
}

// ExecuteContext is a context using by action
//go:generate mockery --name=ExecuteContext --output=. --inpackage  --filename=run_mock.go
type ExecuteContext interface {
	Context() context.Context
	// WithValue can attach value to context,so can share data between action
	// however it is base on memory, it is possible to lose changes such as application crash
	WithValue(key, value interface{})
	ShareData() ShareDataOperator
	// Trace print msg to the TaskInstance.Traces.
	Trace(msg string, opt ...TraceOp)
	// Tracef print msg to the TaskInstance.Traces.
	// Arguments are handled in the manner of fmt.Printf.
	// Opt can only be placed at the end of args.
	// Tracef("{format_str}",{format_val},{opts})
	// e.g. Tracef("%d", 1, TraceOpPersistAfterAction)
	// wrong case: Tracef("%d", TraceOpPersistAfterAction, 1)
	Tracef(msg string, a ...interface{})
	GetVar(varName string) (string, bool)
	IterateVars(iterateFunc utils.KeyValueIterateFunc)
}

// ShareDataOperator used to operate share data
type ShareDataOperator interface {
	Get(key string) (string, bool)
	Set(key string, val string)
}

var _ ExecuteContext = &DefExecuteContext{}

// Default Executor context
type DefExecuteContext struct {
	ctx          context.Context
	op           ShareDataOperator
	trace        func(msg string, opt ...TraceOp)
	varsGetter   func(string) (string, bool)
	varsIterator utils.KeyValueIterator
}

// Context
func (e *DefExecuteContext) Context() context.Context {
	return e.ctx
}

// WithValue is wrapper of "context.WithValue"
func (e *DefExecuteContext) WithValue(key, value interface{}) {
	e.ctx = context.WithValue(e.ctx, key, value)
}

// ShareData
func (e *DefExecuteContext) ShareData() ShareDataOperator {
	return e.op
}

// Trace
func (e *DefExecuteContext) Trace(msg string, opt ...TraceOp) {
	e.trace(msg, opt...)
}

// Tracef
func (e *DefExecuteContext) Tracef(msg string, a ...interface{}) {
	args, ops := splitArgsAndOpt(a...)
	e.trace(fmt.Sprintf(msg, args...), ops...)
}

// splitArgsAndOpt split args and opt, opt must be placed at the end of args
func splitArgsAndOpt(a ...interface{}) ([]interface{}, []TraceOp) {
	optStartIndex := len(a)
	for i := len(a) - 1; i >= 0; i -= 1 {
		if _, ok := a[i].(TraceOp); !ok {
			optStartIndex = i + 1
			break
		}
		if i == 0 {
			optStartIndex = 0
		}
	}

	var traceOps []TraceOp
	for i := optStartIndex; i < len(a); i++ {
		traceOps = append(traceOps, a[i].(TraceOp))
	}

	return a[:optStartIndex], traceOps
}

// GetVar used to get key from ShareData
func (e *DefExecuteContext) GetVar(varName string) (string, bool) {
	return e.varsGetter(varName)
}

// IterateVars used to iterate ShareData
func (e *DefExecuteContext) IterateVars(iterateFunc utils.KeyValueIterateFunc) {
	e.varsIterator(iterateFunc)
}

// TraceOption
type TraceOption struct {
	Priority PersistPriority
}
type TraceOp func(opt *TraceOption)

// NewTraceOption
func NewTraceOption(ops ...TraceOp) *TraceOption {
	opt := &TraceOption{}
	for i := range ops {
		if ops[i] != nil {
			ops[i](opt)
		}
	}
	return opt
}

var (
	// TraceOpPersistAfterAction
	// Patch change when after execute each action("RunBefore", "Run" or "RunAfter")
	// this will be high performance, but here is a risk to lost trace when application crashed
	TraceOpPersistAfterAction TraceOp = func(opt *TraceOption) {
		opt.Priority = PersistPriorityAfterAction
	}
)

// PersistPriority
type PersistPriority string

const (
	// Patch change immediately, this will increase the burden of storage
	// the default behavior
	PersistPriorityImmediately = "Immediately"
	// Patch change when after execute each action("RunBefore", "Run" or "RunAfter")
	// this will be high performance, but here is a risk to lost trace when application crashed
	PersistPriorityAfterAction = "AfterAction"
)
