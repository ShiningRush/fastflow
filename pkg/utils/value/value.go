package value

import (
	"fmt"
	"strings"
)

type MapValue map[string]interface{}

type Setter func(v interface{})

type MapValueCallback func(walkContext *WalkContext, v interface{}) error

var NoneSetter = func(v interface{}) {}

type WalkContext struct {
	path   []string
	Setter Setter
}

func NewWalkContext() *WalkContext {
	return &WalkContext{
		path:   nil,
		Setter: NoneSetter,
	}
}

func (c *WalkContext) push(path string) {
	c.path = append(c.path, path)
}

func (c *WalkContext) pop() {
	c.path = c.path[:len(c.path)-1]
}

func (c *WalkContext) Path() string {
	var b strings.Builder
	for i, s := range c.path {
		if !strings.HasPrefix(s, "[") && i != 0 {
			b.WriteString(".")
		}
		b.WriteString(s)
	}
	return b.String()
}

func (c *WalkContext) resetSetter() {
	c.Setter = NoneSetter
}

func (val MapValue) Walk(callback MapValueCallback) error {
	return val.walk(NewWalkContext(), callback)
}

func (val MapValue) walk(walkContext *WalkContext, callback MapValueCallback) error {
	for k, v := range val {
		err := func() error {
			walkContext.push(k)
			defer walkContext.pop()
			if m, ok := v.(map[string]interface{}); ok {
				// 遍历 map
				err := MapValue(m).walk(walkContext, callback)
				if err != nil {
					return err
				}
				return nil
			}
			if s, ok := v.([]interface{}); ok {
				// 遍历 slice
				err := val.walkSlice(walkContext, s, callback)
				if err != nil {
					return err
				}
				return nil
			}
			// 遍历 map 中的 value
			walkContext.Setter = func(v interface{}) {
				val[k] = v
			}
			defer walkContext.resetSetter()
			return callback(walkContext, v)
		}()
		if err != nil {
			return err
		}
	}
	return nil
}

func (val MapValue) walkSlice(walkContext *WalkContext, slice []interface{}, callback MapValueCallback) error {
	for i, item := range slice {
		err := func() error {
			path := fmt.Sprintf("[%d]", i)
			walkContext.push(path)
			defer walkContext.pop()
			// 遍历 slice 中的 map
			if m, ok := item.(map[string]interface{}); ok {
				err := MapValue(m).walk(walkContext, callback)
				if err != nil {
					return err
				}
				return nil
			}
			if s, ok := item.([]interface{}); ok {
				err := val.walkSlice(walkContext, s, callback)
				if err != nil {
					return err
				}
				return nil
			}

			// 遍历 slice 中的 value
			walkContext.Setter = func(v interface{}) {
				slice[i] = v
			}
			defer walkContext.resetSetter()
			return callback(walkContext, item)
		}()
		if err != nil {
			return err
		}
	}
	return nil
}

type StringSetter func(s string)

type MapValueStringCallback func(walkContext *WalkContext, v string) error

func (val MapValue) WalkString(callback MapValueStringCallback) error {
	return val.walk(NewWalkContext(), func(walkContext *WalkContext, v interface{}) error {
		if s, ok := v.(string); ok {
			return callback(walkContext, s)
		}
		return nil
	})
}
