package value

import (
	"fmt"
	"strings"
)

type MapValue map[string]interface{}

type Setter func(v interface{})

type MapValueCallback func(setter Setter, v interface{}, e Extra) error

type Extra struct {
	MapValue MapValue
	path     []string
}

func (e *Extra) Path() string {
	var b strings.Builder
	for i, s := range e.path {
		if !strings.HasPrefix(s, "[") && i != 0 {
			b.WriteString(".")
		}
		b.WriteString(s)
	}
	return b.String()
}

func (val MapValue) Walk(callback MapValueCallback) error {
	return val.walk(callback, Extra{})
}

func (val MapValue) walk(callback MapValueCallback, e Extra) error {
	for k, v := range val {
		extra := Extra{
			path: append(e.path, k),
		}
		if m, ok := v.(map[string]interface{}); ok {
			// 遍历 map
			err := MapValue(m).walk(callback, extra)
			if err != nil {
				return err
			}
			continue
		}
		if s, ok := v.([]interface{}); ok {
			// 遍历 slice
			err := val.walkSlice(s, callback, extra)
			if err != nil {
				return err
			}
			continue
		}
		// 遍历 map 中的 value
		err := callback(func(v interface{}) {
			val[k] = v
		}, v, extra)
		if err != nil {
			return err
		}
	}
	return nil
}

func (val MapValue) walkSlice(slice []interface{}, callback MapValueCallback, e Extra) error {
	for i, item := range slice {
		path := fmt.Sprintf("[%d]", i)
		// 遍历 slice 中的 map
		extra := Extra{
			path: append(e.path, path),
		}
		if m, ok := item.(map[string]interface{}); ok {
			err := MapValue(m).walk(callback, extra)
			if err != nil {
				return err
			}
			continue
		}
		if s, ok := item.([]interface{}); ok {
			err := val.walkSlice(s, callback, extra)
			if err != nil {
				return err
			}
			continue
		}

		// 遍历 slice 中的 value
		err := callback(func(v interface{}) {
			slice[i] = v
		}, item, extra)
		if err != nil {
			return err
		}
	}
	return nil
}

type StringSetter func(s string)

type MapValueStringCallback func(setter StringSetter, v string, e Extra) error

func (val MapValue) WalkString(callback MapValueStringCallback) error {
	return val.walk(func(setter Setter, v interface{}, e Extra) error {
		if s, ok := v.(string); ok {
			return callback(func(s string) {
				setter(s)
			}, s, e)
		}
		return nil
	}, Extra{})
}
