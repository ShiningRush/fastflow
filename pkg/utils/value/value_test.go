package value

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const beginFlag = "beginFlag"

var (
	val = map[string]interface{}{
		beginFlag: "flag",
		"string":  "b",
		"int":     1,
		"float64": 1.2,
		"map": map[string]interface{}{
			"internal": "internal",
		},
	}
)

func TestValue_Walk(t *testing.T) {
	var logs []interface{}
	type args struct {
		callback func(m MapValue, k string, v interface{}, e Extra) error
	}
	type TestCase struct {
		name    string
		val     MapValue
		args    args
		wantErr assert.ErrorAssertionFunc
		assert  func(t *testing.T, tt TestCase)
	}

	tests := []TestCase{
		{
			name: "walk",
			val:  val,
			args: args{
				callback: func(m MapValue, k string, v interface{}, e Extra) error {
					logs = append(logs, k)
					logs = append(logs, v)
					return nil
				},
			},
			wantErr: assert.NoError,

			assert: func(t *testing.T, tt TestCase) {
				// fix map order
				logs = fixMapOrder(logs)
				wantLogs := []interface{}{beginFlag, "flag", "string", "b", "int", 1,
					"float64", 1.2, "internal", "internal"}
				assert.Equal(t, wantLogs, logs)
			},
		},
		{
			name: "break",
			val:  val,
			args: args{
				callback: func(m MapValue, k string, v interface{}, e Extra) error {
					logs = append(logs, k)
					logs = append(logs, v)
					if k == "string" {
						return errors.New("break")
					}
					return nil
				},
			},
			wantErr: assert.Error,
			assert: func(t *testing.T, tt TestCase) {
				assert.Equal(t, "string", logs[len(logs)-2])
				assert.Equal(t, "b", logs[len(logs)-1])
			},
		},
		{
			name: "",
			val: map[string]interface{}{
				"a": map[string]interface{}{
					"b": "vb",
					"c": map[string]interface{}{
						"d": map[string]interface{}{
							"e": "f",
						},
					},
				},
			},
			args: args{
				func(m MapValue, k string, v interface{}, e Extra) error {
					logs = append(logs, e.Path())
					return nil
				},
			},
			wantErr: assert.NoError,
			assert: func(t *testing.T, tt TestCase) {
				expected := []interface{}{"a.b", "a.c.d.e"}
				assert.Equal(t, expected, logs)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs = nil
			err := tt.val.Walk(tt.args.callback)
			tt.wantErr(t, err, fmt.Sprintf("Walk()"))
			tt.assert(t, tt)

		})
	}
}

func fixMapOrder(logs []interface{}) []interface{} {
	for {
		if len(logs) == 0 || logs[0] == beginFlag {
			break
		}
		logs = append(logs, logs[0])[1:]
	}
	return logs
}

func TestValue_WalkString(t *testing.T) {
	var logs []interface{}
	type args struct {
		callback MapValueStringCallback
	}
	type TestCase struct {
		name    string
		val     MapValue
		args    args
		wantErr assert.ErrorAssertionFunc
		assert  func(t *testing.T, tt TestCase)
	}
	tests := []TestCase{
		{
			name: "walk string",
			val:  val,
			args: args{
				func(m MapValue, k string, v string, e Extra) error {
					logs = append(logs, k)
					logs = append(logs, v)
					return nil
				},
			},
			wantErr: assert.NoError,
			assert: func(t *testing.T, tt TestCase) {
				logs = fixMapOrder(logs)
				wantLogs := []interface{}{beginFlag, "flag", "string", "b", "internal", "internal"}
				assert.Equal(t, wantLogs, logs)
			},
		},
		{
			name: "break",
			val:  val,
			args: args{
				callback: func(m MapValue, k string, v string, e Extra) error {
					logs = append(logs, k)
					logs = append(logs, v)
					if k == "string" {
						return errors.New("aa")
					}
					return nil
				},
			},
			wantErr: assert.Error,
			assert: func(t *testing.T, tt TestCase) {
				assert.Equal(t, "string", logs[len(logs)-2])
				assert.Equal(t, "b", logs[len(logs)-1])
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs = nil
			tt.wantErr(t, tt.val.WalkString(tt.args.callback), fmt.Sprintf("WalkString()"))
			tt.assert(t, tt)
		})

	}
}
