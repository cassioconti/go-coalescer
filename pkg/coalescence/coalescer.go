package coalescence

import (
	"sync"
	"time"
)

type coalescer struct {
	cache map[string]*content
	lock  sync.RWMutex
}

type content struct {
	response interface{}
	err      error
	cached   time.Time
	expire   time.Duration
}

type Coalescer interface {
	Do(operation func() (interface{}, error), key string, expire time.Duration) (interface{}, error)
}

func NewCoalescer() Coalescer {
	return &coalescer{
		cache: make(map[string]*content),
		lock:  sync.RWMutex{},
	}
}

// Do implements Coalescer.
func (c *coalescer) Do(operation func() (interface{}, error), key string, expire time.Duration) (interface{}, error) {
	cont := c.get(key)
	if invalidCachedValue(cont) {
		cont = c.set(operation, key, expire)
	}

	return cont.response, cont.err
}

func invalidCachedValue(cont *content) bool {
	return cont == nil || time.Since(cont.cached) > cont.expire
}

func (c *coalescer) get(key string) *content {
	// Read locks allow reads in parallel. It blocks a write if a read in happening.
	// This is more efficient if you have more reads than writes.
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.cache[key]
}

func (c *coalescer) set(operation func() (interface{}, error), key string, expire time.Duration) *content {
	// When you hit the condition to write, you need a full lock
	c.lock.Lock()
	defer c.lock.Unlock()
	// Read one more time, because multiple threads could have come to this method before the full lock
	// and are not aware the state was already update.
	cont := c.cache[key]
	// Check the condition again because another thread could have already updated it in the meantime.
	if invalidCachedValue(cont) {
		resp, err := operation()
		cont = &content{
			response: resp,
			err:      err,
			cached:   time.Now(),
			expire:   expire,
		}

		c.cache[key] = cont
	}

	return cont
}
