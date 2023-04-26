package caching

import (
	"bytes"
	"io"

	"cloud.google.com/go/storage"
)

type Cache interface {
	Contains(hash uint32) bool
	Set(o *Object)
	SetFrom(attrs *storage.ObjectAttrs, reader io.Reader) (*Object, error)
	Get(hash uint32) (*Object, bool)
	Data(hash uint32) (*bytes.Reader, bool)
}

const (
	Kilobyte uint64 = 1024
	Megabyte        = Kilobyte * 1024
	Gigabyte        = Megabyte * 1024
)

type Object struct {
	Hash uint32
	Name string
	Data []byte
	Size uint64
}

// Constructors

func ObjectFrom(attrs *storage.ObjectAttrs, reader io.Reader) (*Object, error) {
	return new(Object).From(attrs, reader)
}

// Public Methods

func (o *Object) From(attrs *storage.ObjectAttrs, reader io.Reader) (*Object, error) {
	o.Hash = attrs.CRC32C
	o.Name = attrs.Name
	o.Size = uint64(attrs.Size)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	o.Data = data
	return o, nil
}
