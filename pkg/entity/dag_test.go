package entity

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDagInstance_VarsIterator(t *testing.T) {
	dagIns := &DagInstance{
		Vars: DagInstanceVars{
			"key1": DagInstanceVar{Value: "value1"},
			"key2": DagInstanceVar{Value: "value2"},
		},
	}

	ret := []struct {
		key   string
		value string
	}{}
	wantRet := []struct {
		key   string
		value string
	}{
		{key: "key1", value: "value1"},
		{key: "key2", value: "value2"},
	}

	dagIns.VarsIterator()(func(key, val string) (stop bool) {
		ret = append(ret, struct {
			key   string
			value string
		}{key: key, value: val})
		return false
	})

	assert.ElementsMatch(t, wantRet, ret)

	dagIns.VarsIterator()(func(key, val string) (stop bool) {
		ret = append(ret, struct {
			key   string
			value string
		}{key: key, value: val})
		return true
	})
	assert.Equal(t, len(ret), 3)
}

func TestDagInstance_VarsGetter(t *testing.T) {
	dagIns := &DagInstance{
		Vars: DagInstanceVars{
			"key1": DagInstanceVar{Value: "value1"},
			"key2": DagInstanceVar{Value: "value2"},
		},
	}

	keyMap := map[string]struct {
		wantFind  bool
		wantValue string
	}{
		"key1": {wantFind: true, wantValue: "value1"},
		"key2": {wantFind: true, wantValue: "value2"},
		"key3": {wantFind: false, wantValue: ""},
	}

	getter := dagIns.VarsGetter()
	for k, val := range keyMap {
		ret, ok := getter(k)
		assert.Equal(t, val.wantValue, ret)
		assert.Equal(t, val.wantFind, ok)
	}
}

func TestDagInstance_Success(t *testing.T) {
	dagIns := &DagInstance{}
	testHook(t, dagIns, string(DagInstanceStatusSuccess), DagInstanceStatusSuccess, func() {
		dagIns.Success()
	})
}

func TestDagInstance_Fail(t *testing.T) {
	dagIns := &DagInstance{}
	testHook(t, dagIns, string(DagInstanceStatusFailed), DagInstanceStatusFailed, func() {
		dagIns.Fail("")
	})
}

func TestDagInstance_Run(t *testing.T) {
	dagIns := &DagInstance{}
	testHook(t, dagIns, string(DagInstanceStatusRunning), DagInstanceStatusRunning, func() {
		dagIns.Run()
	})
}

func TestDagInstance_Retry(t *testing.T) {
	dagIns := &DagInstance{
		Status: DagInstanceStatusFailed,
	}
	testHook(t, dagIns, "retry", DagInstanceStatusFailed, func() {
		dagIns.Retry([]string{"testId"})
	})
}

func TestDagInstance_Block(t *testing.T) {
	dagIns := &DagInstance{}
	testHook(t, dagIns, string(DagInstanceStatusBlocked), DagInstanceStatusBlocked, func() {
		dagIns.Block("")
	})
}

func TestDagInstance_Continue(t *testing.T) {
	dagIns := &DagInstance{
		Status: DagInstanceStatusBlocked,
	}
	testHook(t, dagIns, "continue", DagInstanceStatusBlocked, func() {
		err := dagIns.Continue([]string{"testId"})
		assert.NoError(t, err)
	})

	incompleteDagIns := &DagInstance{
		Status: DagInstanceStatusBlocked,
		Cmd:    &Command{},
	}
	err := incompleteDagIns.Continue([]string{"testId"})
	assert.Equal(t, fmt.Errorf("dag instance have a incomplete command"), err)
}

func testHook(t *testing.T, dagIns *DagInstance, wantRet string, wantStatus DagInstanceStatus, call func()) {
	ret := ""
	HookDagInstance = DagInstanceLifecycleHook{
		BeforeRun: func(dagIns *DagInstance) {
			assert.NotNil(t, dagIns)
			ret = string(DagInstanceStatusRunning)
		},
		BeforeSuccess: func(dagIns *DagInstance) {
			assert.NotNil(t, dagIns)
			ret = string(DagInstanceStatusSuccess)
		},
		BeforeFail: func(dagIns *DagInstance) {
			assert.NotNil(t, dagIns)
			ret = string(DagInstanceStatusFailed)
		},
		BeforeBlock: func(dagIns *DagInstance) {
			assert.NotNil(t, dagIns)
			ret = string(DagInstanceStatusBlocked)
		},
		BeforeContinue: func(dagIns *DagInstance) {
			assert.NotNil(t, dagIns)
			ret = "continue"
		},
		BeforeRetry: func(dagIns *DagInstance) {
			assert.NotNil(t, dagIns)
			ret = "retry"
		},
	}

	call()
	assert.Equal(t, wantStatus, dagIns.Status)
	assert.Equal(t, wantRet, ret)
}

func TestDagInstanceVars_Render(t *testing.T) {
	tests := []struct {
		name       string
		giveVar    DagInstanceVars
		giveParams map[string]interface{}
		wantParams map[string]interface{}
	}{
		{
			name: "simple val",
			giveVar: DagInstanceVars{
				"test1": {
					Value: "test1-v",
				},
				"test2": {
					Value: "test2-v",
				},
			},
			giveParams: map[string]interface{}{
				"test1": "{{test1}}",
				"test2": "{{test2}}",
				"test3": map[string]interface{}{
					"test4": "{{test1}}",
				},
			},
			wantParams: map[string]interface{}{
				"test1": "test1-v",
				"test2": "test2-v",
				"test3": map[string]interface{}{
					"test4": "test1-v",
				},
			},
		},
		{
			name: "json string",
			giveVar: DagInstanceVars{
				"jsonString": {
					Value: `{"cluster_id":"tenc-6h27vfr2"}`,
				},
			},
			giveParams: map[string]interface{}{
				"jsonString": "{{jsonString}}",
			},
			wantParams: map[string]interface{}{
				"jsonString": `{"cluster_id":"tenc-6h27vfr2"}`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ret, err := tc.giveVar.Render(tc.giveParams)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantParams, ret)
		})
	}
}

func TestShareData_Get(t *testing.T) {
	tests := []struct {
		giveData *ShareData
		giveKey  string
		wantRet  string
	}{
		{
			giveData: &ShareData{
				Dict: map[string]string{
					"key": "value",
				},
			},
			giveKey: "key",
			wantRet: "value",
		},
		{
			giveData: &ShareData{},
			wantRet:  "",
		},
	}

	for _, tc := range tests {
		ret, _ := tc.giveData.Get(tc.giveKey)
		assert.Equal(t, tc.wantRet, ret)
	}
}

func TestShareData_Set(t *testing.T) {
	tests := []struct {
		giveData  *ShareData
		giveKey   string
		giveValue string
		wantRet   map[string]string
		wantErr   error
	}{
		{
			giveData: &ShareData{
				Dict: map[string]string{},
			},
			giveKey:   "key",
			giveValue: "value",
			wantRet: map[string]string{
				"key": "value",
			},
		},
		{
			giveData: &ShareData{
				Dict: map[string]string{},
				Save: func(data *ShareData) error {
					return fmt.Errorf("save failed")
				},
			},
			wantErr: fmt.Errorf("save failed"),
			wantRet: map[string]string{},
		},
	}

	for _, tc := range tests {
		tc.giveData.Set(tc.giveKey, tc.giveValue)
		assert.Equal(t, tc.wantRet, tc.giveData.Dict)
	}
}
