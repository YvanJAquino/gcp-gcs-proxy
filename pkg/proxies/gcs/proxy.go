package gcs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	caching "github.com/YvanJAquino/gcp-gcs-proxy/pkg/caching"
	cache "github.com/YvanJAquino/gcp-gcs-proxy/pkg/caching/lru"
)

const (
	basePath    = "storage/v1/b"
	basePathLen = 12
)

type StorageProxy struct {
	client    *storage.Client
	projectID string
	cache     caching.Cache
}

// Constructors

func New(client *storage.Client) *StorageProxy {
	return &StorageProxy{
		client: client,
		cache:  cache.New(caching.Megabyte*500, 70),
	}
}

func Default(ctx context.Context) (*StorageProxy, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	projectID, err := metadata.ProjectID()
	if err != nil {
		return nil, err
	}
	proxy := New(client)
	proxy.projectID = projectID
	return proxy, nil
}

func Must() *StorageProxy {
	proxy, err := Default(context.Background())
	if err != nil {
		panic(err)
	}
	return proxy
}

// Methods

// Method ServeHTTP satisfies the HTTP Handler interface.
func (p *StorageProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[1:]
	tokens := strings.Split(path, "/")
	if len(path) < basePathLen || len(tokens) < 3 {
		HTTPErrBadRequest(w)
		return
	}
	if path[:basePathLen] != basePath {
		HTTPErrBadRequest(w)
		return
	}

	switch {
	case path == basePath:
		p.listBuckets(w, r)
	case len(tokens) == 5 && tokens[4] == "o":
		bkt := tokens[3]
		p.listObjects(w, r, bkt)
	case len(tokens) >= 6 && tokens[4] == "o":
		bkt := tokens[3]
		obj := strings.Join(tokens[5:], "/")
		alt := r.URL.Query().Get("alt")
		if alt == "media" {
			p.getObject(w, r, bkt, obj)
		} else {
			p.getObjectMetadata(w, r, bkt, obj)
		}
	default:
		HTTPErrBadRequest(w)
		return
	}
}

func (p *StorageProxy) listBuckets(w http.ResponseWriter, r *http.Request) {
	bktsIter := p.client.Buckets(r.Context(), p.projectID)
	bkts := make([]string, 0)
	for {
		attrs, err := bktsIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			HTTPErrInternalServerError(w, err)
			return
		}
		bkts = append(bkts, attrs.Name)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")
	err := enc.Encode(&bkts)
	if err != nil {
		HTTPErrInternalServerError(w, err)
		return
	}
}

func (p *StorageProxy) listObjects(w http.ResponseWriter, r *http.Request, b string) {
	ctx := r.Context()
	bkt := p.client.Bucket(b)
	_, err := bkt.Attrs(ctx)
	if err == storage.ErrBucketNotExist {
		HTTPErrBucketDoesNotExist(w)
		return
	}

	objs := make([]string, 0)
	objIter := bkt.Objects(r.Context(), nil)
	for {
		attrs, err := objIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			HTTPErrInternalServerError(w, err)
			return
		}
		objs = append(objs, attrs.Name)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")
	err = enc.Encode(&objs)
	if err != nil {
		HTTPErrInternalServerError(w, err)
		return
	}
}

func (p *StorageProxy) getObjectMetadata(w http.ResponseWriter, r *http.Request, b, o string) {
	bkt := p.client.Bucket(b)
	obj := bkt.Object(o)
	attrs, err := obj.Attrs(r.Context())
	if err != nil {
		if err == storage.ErrObjectNotExist {
			HTTPErrObjectDoesNotExist(w)
			return
		} else {
			HTTPErrInternalServerError(w, err)
			return
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")
	err = enc.Encode(attrs)
	if err != nil {
		HTTPErrInternalServerError(w, err)
		return
	}
}

func (p *StorageProxy) getObject(w http.ResponseWriter, r *http.Request, b, o string) {
	bkt := p.client.Bucket(b)
	obj := bkt.Object(o)
	attrs, err := obj.Attrs(r.Context())
	if err != nil {
		if err == storage.ErrObjectNotExist {
			http.Error(w, ErrObjectDoesNotExist.Error(), http.StatusInternalServerError)
			return
		} else {
			log.Fatal(err)
		}
	}

	w.Header().Set("Content-Type", attrs.ContentType)

	// If the cache contains the CRC32c hash already, use the cache.
	if p.cache.Contains(attrs.CRC32C) {
		cr, ok := p.cache.Data(attrs.CRC32C)
		if !ok {
			http.Error(w, fmt.Sprintf("Hash %v not in cache", attrs.CRC32C), http.StatusInternalServerError)
			log.Println(err)
			return

		}
		_, err = io.Copy(w, cr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
		}
		return
	}

	or, err := obj.NewReader(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
	}

	tr := io.TeeReader(or, w)
	_, err = p.cache.SetFrom(attrs, tr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
	}
}
