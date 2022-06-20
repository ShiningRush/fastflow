package value

import "strings"

type MapValue map[string]interface{}

type MapValueCallback func(m MapValue, k string, v interface{}, e Extra) error

type Extra struct {
	MapValue MapValue
	path     []string
}

func (e *Extra) Path() string {
	return strings.Join(e.path, ".")
}

func (val MapValue) Walk(callback MapValueCallback) error {
	return val.walk(callback, Extra{})
}

func (val MapValue) walk(callback MapValueCallback, e Extra) error {
	for k, v := range val {
		if m, ok := v.(map[string]interface{}); ok {
			err := MapValue(m).walk(callback, Extra{
				path: append(e.path, k),
			})
			if err != nil {
				return err
			}
			continue
		}
		err := callback(val, k, v, Extra{
			path: append(e.path, k),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

type MapValueStringCallback func(m MapValue, k string, v string, e Extra) error

func (val MapValue) WalkString(callback MapValueStringCallback) error {
	return val.walk(func(m MapValue, k string, v interface{}, e Extra) error {
		if s, ok := v.(string); ok {
			return callback(m, k, s, e)
		}
		return nil
	}, Extra{})
}
