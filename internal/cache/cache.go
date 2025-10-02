package cache

import (
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type Entry struct {
	Balance uint64
	Err     error
	Expiry  time.Time
}

type Cache struct {
	mu   sync.RWMutex
	data map[string]Entry
	ttl  time.Duration
	sfg  singleflight.Group
}

func New(ttl time.Duration) *Cache { return &Cache{data: make(map[string]Entry), ttl: ttl} }

func (c *Cache) Get(key string) (Entry, bool) {
	c.mu.RLock()
	e, ok := c.data[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.Expiry) {
		return Entry{}, false
	}
	return e, true
}

func (c *Cache) Set(key string, bal uint64, err error) {
	c.mu.Lock()
	c.data[key] = Entry{Balance: bal, Err: err, Expiry: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

func (c *Cache) Do(key string, fn func() (uint64, error)) (uint64, error) {
	// singleflight ensures concurrent calls for same key share work
	v, err, _ := c.sfg.Do(key, func() (interface{}, error) {
		// double-check cache inside the flight
		if e, ok := c.Get(key); ok {
			return e.Balance, e.Err
		}
		bal, e := fn()
		c.Set(key, bal, e)
		return bal, e
	})
	if err != nil {
		return 0, err
	}
	return v.(uint64), nil
}

