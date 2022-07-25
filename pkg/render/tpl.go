package render

import (
	"sync"
	"text/template"

	"github.com/golang/groupcache/lru"
)

//go:generate mockgen -source=tpl.go -destination=tpl_mock.go -package=render   TplProvider
type TplProvider interface {
	GetTpl(tplText string) (*template.Template, error)
}

var _ TplProvider = &CachedTplProvider{}

type CachedTplProvider struct {
	cache   *lru.Cache
	rwMutex sync.RWMutex
}

func NewCachedTplProvider(maxSize int) *CachedTplProvider {
	cache := lru.New(maxSize)
	return &CachedTplProvider{
		cache:   cache,
		rwMutex: sync.RWMutex{},
	}
}

func (c *CachedTplProvider) cacheGetTpl(tplText string) (*template.Template, bool) {
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()
	v, ok := c.cache.Get(tplText)
	if !ok {
		return nil, false
	}
	return v.(*template.Template), true
}

func (c *CachedTplProvider) cacheSetTpl(tplText string, template *template.Template) {
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()
	c.cache.Add(tplText, template)
}

func (c *CachedTplProvider) GetTpl(tplText string) (*template.Template, error) {
	tpl, ok := c.cacheGetTpl(tplText)
	if ok {
		return tpl, nil
	}
	tpl, err := c.ParseTpl(tplText)
	if err != nil {
		return nil, err
	}
	c.cacheSetTpl(tplText, tpl)
	return tpl, err
}

func (c *CachedTplProvider) ParseTpl(tplText string) (*template.Template, error) {
	tpl, err := template.New(tplText).Parse(tplText)
	if err != nil {
		return nil, err
	}
	tpl.Option("missingkey=error")
	return tpl, err
}
