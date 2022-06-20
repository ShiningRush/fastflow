package render

import (
	"fmt"
	"strings"
)

//go:generate mockgen -source=render.go -destination=render_mock.go -package=render   Render
type Render interface {
	Render(tplText string, data interface{}) (string, error)
}

var _ Render = &TplRender{}

type TplRender struct {
	tplProvider TplProvider
}

func NewTplRender(tplProvider TplProvider) *TplRender {
	return &TplRender{
		tplProvider: tplProvider,
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
