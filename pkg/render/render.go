package render

import (
	"fmt"
	"strings"
)

var (
	// CacheSize tpl cache size
	CacheSize = 1000
)

type TplRender struct {
	tplProvider *TplProvider
}

func NewTplRender() *TplRender {
	return &TplRender{
		tplProvider: NewCachedTplProvider(CacheSize),
	}
}

func (t *TplRender) Render(tplText string, data interface{}) (string, error) {
	tpl, err := t.tplProvider.GetTpl(tplText)
	if err != nil {
		return "", fmt.Errorf("get tpl failed: %w", err)
	}

	var buf strings.Builder
	err = tpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("execute tpl failed: %w", err)
	}
	return buf.String(), nil

}
