package lru

import (
	"bytes"
	"container/list"
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
	items   map[uint32]*list.Element
	recency *list.List
	size    uint64
	cap     uint64
	bound   uint64
	mu      sync.RWMutex
}

// Constructors

func New(cap, ratio uint64) *Cache {
	bound := (ratio * cap) / 100
	log.Printf("LRU CACHE ADM new cache created BOUND: %v / CAP: %v", bound, cap)
	return &Cache{
		items:   make(map[uint32]*list.Element),
		recency: list.New(),
		size:    0,
		cap:     cap,
		bound:   bound,
	}
}

// Private methods

// LRU Eviction
func (c *Cache) evict() {
	for LoadU64(&c.size) > LoadU64(&c.bound) {
		item := c.recency.Remove(c.recency.Back()).(*caching.Object)
		AddU64(&c.size, -item.Size)
		delete(c.items, item.Hash)
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
	c.items[o.Hash] = c.recency.PushFront(o)
	c.mu.Unlock()

	log.Printf("LRU CACHE SET %s identified by hash %v - object size %v - size %v", o.Name, o.Hash, o.Size, c.size)
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
	elem, ok := c.items[hash]
	c.mu.RUnlock()
	if !ok {
		return nil, ok
	}
	// Is this necessary?
	c.mu.Lock()
	c.recency.MoveToFront(elem)
	c.mu.Unlock()

	if ok {
		log.Printf("CACHE GET [HIT] hash %v identified by name %s", hash, elem.Value.(*caching.Object).Name)
	} else {
		log.Printf("CACHE GET [MISS] unable to find hash %v in cache", hash)
	}
	return elem.Value.(*caching.Object), ok
}

func (c *Cache) Data(hash uint32) (*bytes.Reader, bool) {
	obj, ok := c.Get(hash)
	if ok {
		return bytes.NewReader(obj.Data), ok
	}
	return nil, ok
}
