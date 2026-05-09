package cache

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type entry[V any] struct {
	value    V
	expireAt time.Time
}

type Cache[K comparable, V any] struct {
	mu      sync.RWMutex
	entries map[K]entry[V]
	ttl     time.Duration
	group   singleflight.Group
	stopCh  chan struct{}
}

func New[K comparable, V any](ttl time.Duration) *Cache[K, V] {
	c := &Cache[K, V]{
		entries: make(map[K]entry[V]),
		ttl:     ttl,
		stopCh:  make(chan struct{}),
	}
	go c.cleanup()
	return c
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	e, ok := c.entries[key]
	if !ok {
		var zero V
		return zero, false
	}
	if time.Now().After(e.expireAt) {
		var zero V
		return zero, false
	}
	return e.value, true
}

// GetOrLoad 缓存命中直接返回，未命中时通过 loadFn 加载并回填
// 内置 singleflight：同一 key 的并发请求只会触发一次 loadFn，其余请求共享结果
func (c *Cache[K, V]) GetOrLoad(key K, loadFn func() (V, error)) (V, error) {
	if v, ok := c.Get(key); ok {
		return v, nil
	}

	sfKey := fmt.Sprintf("%v", key)

	result, err, _ := c.group.Do(sfKey, func() (interface{}, error) {
		v, err := loadFn()
		if err != nil {
			return nil, err
		}
		c.Set(key, v)
		return v, nil
	})
	if err != nil {
		var zero V
		return zero, err
	}
	return result.(V), nil
}

func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = entry[V]{
		value:    value,
		expireAt: time.Now().Add(c.ttl),
	}
}

func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

func (c *Cache[K, V]) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for k, e := range c.entries {
				if now.After(e.expireAt) {
					delete(c.entries, k)
				}
			}
			c.mu.Unlock()
		case <-c.stopCh:
			return
		}
	}
}

func (c *Cache[K, V]) Stop() {
	close(c.stopCh)
}
