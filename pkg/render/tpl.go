package render

import (
	"sync"
	"text/template"

	"github.com/golang/groupcache/lru"
)

type TplProvider struct {
	cache   *lru.Cache
	rwMutex sync.RWMutex
}

func NewCachedTplProvider(maxSize int) *TplProvider {
	cache := lru.New(maxSize)
	return &TplProvider{
		cache:   cache,
		rwMutex: sync.RWMutex{},
	}
}

func (c *TplProvider) cacheGetTpl(tplText string) (*template.Template, bool) {
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()
	v, ok := c.cache.Get(tplText)
	if !ok {
		return nil, false
	}
	return v.(*template.Template), true
}

func (c *TplProvider) cacheSetTpl(tplText string, template *template.Template) {
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()
	c.cache.Add(tplText, template)
}

func (c *TplProvider) GetTpl(tplText string) (*template.Template, error) {
	tpl, ok := c.cacheGetTpl(tplText)
	if ok {
		return tpl, nil
	}
	tpl, err := c.parseTpl(tplText)
	if err != nil {
		return nil, err
	}
	c.cacheSetTpl(tplText, tpl)
	return tpl, err
}

func (c *TplProvider) parseTpl(tplText string) (*template.Template, error) {
	tpl, err := template.New(tplText).Parse(tplText)
	if err != nil {
		return nil, err
	}
	tpl.Option("missingkey=error")
	return tpl, err
}
