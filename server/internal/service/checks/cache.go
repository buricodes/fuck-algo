package checks

import (
	"container/list"
	"sync"
	"time"
)

type verdict struct {
	spoilerOf string
	topic     string
}

type entry struct {
	key       string
	value     verdict
	expiresAt time.Time
}

type cache struct {
	mu      sync.Mutex
	ttl     time.Duration
	size    int
	order   *list.List
	entries map[string]*list.Element
}

func newCache(ttl time.Duration, size int) *cache {
	return &cache{ttl: ttl, size: size, order: list.New(), entries: map[string]*list.Element{}}
}

func (c *cache) get(key string, now time.Time) (verdict, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.entries[key]
	if !ok {
		return verdict{}, false
	}
	e := el.Value.(*entry)
	if now.After(e.expiresAt) {
		c.remove(el)
		return verdict{}, false
	}
	c.order.MoveToFront(el)
	return e.value, true
}

func (c *cache) put(key string, value verdict, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.entries[key]; ok {
		c.remove(el)
	}
	c.entries[key] = c.order.PushFront(&entry{key: key, value: value, expiresAt: now.Add(c.ttl)})
	for c.order.Len() > c.size {
		c.remove(c.order.Back())
	}
}

func (c *cache) remove(el *list.Element) {
	c.order.Remove(el)
	delete(c.entries, el.Value.(*entry).key)
}
