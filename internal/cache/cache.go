package cache

import (
	"sync"
)

type Cache struct {sync.Map}

func (c *Cache) Get(k any) any {
	v, _ := c.Load(k)
	return v
}

func (c *Cache) Set(k, v any) {
	c.Store(k, v)
}

func (c *Cache) Clear(k any){
	c.Delete(k)
}
