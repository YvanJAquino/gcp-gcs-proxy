package basic

import (
	"bytes"
	"io"
	"log"
	"sync"
	"sync/atomic"

	"cloud.google.com/go/storage"
	"github.com/YvanJAquino/gcp-gcs-proxy/pkg/caching"
)

var (
	AddU64  = atomic.AddUint64
	LoadU64 = atomic.LoadUint64
)

type Cache struct {
	items            map[uint32]*caching.Object
	mu               sync.RWMutex
	cap, size, bound uint64
}

// Constructors

func New(cap, ratio uint64) *Cache {
	bound := (ratio * cap) / 100
	log.Printf("CACHE ADM new cache created BOUND: %v / CAP: %v", bound, cap)
	return &Cache{
		items: make(map[uint32]*caching.Object),
		cap:   cap,
		bound: bound,
	}
}

// Private methods

// Random eviction
func (c *Cache) evict() {
	for key, item := range c.items {
		delete(c.items, key)
		if AddU64(&c.size, -item.Size) < LoadU64(&c.bound) {
			return
		}
	}
}

// Public methods

func (c *Cache) Contains(hash uint32) bool {
	c.mu.RLock()
	_, ok := c.items[hash]
	c.mu.RUnlock()
	return ok
}

func (c *Cache) Set(o *caching.Object) {
	if c.Contains(o.Hash) {
		return
	}

	if AddU64(&c.size, o.Size) > LoadU64(&c.cap) {
		c.evict()
	}
	c.mu.Lock()
	c.items[o.Hash] = o
	c.mu.Unlock()
	log.Printf("CACHE SET %s identified by hash %v - object size %v - size %v", o.Name, o.Hash, o.Size, c.size)
}

func (c *Cache) SetFrom(attrs *storage.ObjectAttrs, reader io.Reader) (*caching.Object, error) {
	obj, err := caching.ObjectFrom(attrs, reader)
	if err != nil {
		return nil, err
	}
	c.Set(obj)
	return obj, nil
}

func (c *Cache) Get(hash uint32) (*caching.Object, bool) {
	c.mu.RLock()
	obj, ok := c.items[hash]
	c.mu.RUnlock()
	if ok {
		log.Printf("CACHE GET [HIT] hash %v identified by name %s", hash, obj.Name)
	} else {
		log.Printf("CACHE GET [MISS] unable to find hash %v in cache", hash)
	}
	return obj, ok
}

func (c *Cache) Data(hash uint32) (*bytes.Reader, bool) {
	obj, ok := c.Get(hash)
	if ok {
		return bytes.NewReader(obj.Data), ok
	}
	return nil, ok
}
