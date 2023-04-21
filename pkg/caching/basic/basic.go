package basic

import (
	"bytes"
	"io"
	"sync"
	"sync/atomic"

	"cloud.google.com/go/storage"
)

const (
	Kilobyte uint64 = 1024
	Megabyte        = Kilobyte * 1024
	Gigabyte        = Megabyte * 1024
)

type Object struct {
	hash uint32
	name string
	data []byte
	size uint64
}

// Constructors

func ObjectFrom(attrs *storage.ObjectAttrs, reader io.Reader) (*Object, error) {
	return new(Object).From(attrs, reader)
}

// Public Methods

func (o *Object) From(attrs *storage.ObjectAttrs, reader io.Reader) (*Object, error) {
	o.hash = attrs.CRC32C
	o.name = attrs.Name
	o.size = uint64(attrs.Size)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	o.data = data
	return o, nil
}

type Cache struct {
	cache map[uint32]*Object
	mu    sync.RWMutex
	max   uint64
	size  uint64
}

// Constructors

func New() *Cache {
	return &Cache{
		cache: make(map[uint32]*Object),
		max:   Megabyte * 250,
	}
}

// Private methods

func (c *Cache) hasSpace() bool {
	return atomic.LoadUint64(&c.size) <= atomic.LoadUint64(&c.max)
}

func (c *Cache) get(hash uint32) (*Object, bool) {
	c.mu.RLock()
	o, ok := c.cache[hash]
	c.mu.RUnlock()
	return o, ok
}

func (c *Cache) contains(o *Object) bool {
	_, ok := c.get(o.hash)
	return ok
}

func (c *Cache) containsHash(hash uint32) bool {
	_, ok := c.get(hash)
	return ok
}

func (c *Cache) set(o *Object) {
	if !c.hasSpace() {
		panic("cache out of space")
	}
	c.mu.Lock()
	c.cache[o.hash] = o
	atomic.AddUint64(&c.size, o.size)
	c.mu.Unlock()
}

// Public methods

func (c *Cache) Contains(hash uint32) bool {
	return c.containsHash(hash)
}

func (c *Cache) Set(o *Object) *Object {
	if !c.contains(o) {
		c.set(o)
	}
	return o
}

func (c *Cache) SetFrom(attrs *storage.ObjectAttrs, reader io.Reader) (*Object, error) {
	obj, err := ObjectFrom(attrs, reader)
	if err != nil {
		return nil, err
	}
	c.Set(obj)
	return obj, nil
}

func (c *Cache) Get(hash uint32) (*Object, bool) {
	return c.get(hash)
}

func (c *Cache) Data(hash uint32) *bytes.Reader {
	obj, ok := c.get(hash)
	if !ok {
		panic("hash not in cache")
	}
	return bytes.NewReader(obj.data)
}
