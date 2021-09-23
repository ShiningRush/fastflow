package run

import (
	"context"
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
type ExecuteContext interface {
	Context() context.Context
	// WithValue can attach value to context,so can share data between action
	// however it is base on memory, it is possible to lose changes such as application crash
	WithValue(key, value interface{})
	ShareData() ShareDataOperator
	Trace(msg string, opt ...TraceOp)
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
