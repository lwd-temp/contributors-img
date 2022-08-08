package cache

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/storage"
	"contrib.rocks/apps/api/internal/config"
	"contrib.rocks/apps/api/internal/tracing"
	"contrib.rocks/libs/goutils/model"
	"go.opentelemetry.io/otel/attribute"
)

type Service struct {
	bucket *storage.BucketHandle
}

func New(cfg *config.Config, storageClient *storage.Client) *Service {
	var cs Service
	if cfg.CacheBucketName != "" {
		cs.bucket = storageClient.Bucket(cfg.CacheBucketName)
	}
	return &cs
}

func (s *Service) Get(c context.Context, name string) (model.FileHandle, error) {
	return getFile(s.bucket, c, name)
}

func (s *Service) GetJSON(c context.Context, name string, v any) error {
	o, err := getFile(s.bucket, c, name)
	if err != nil {
		return err
	}
	if o == nil {
		v = nil
		return nil
	}
	r := o.Reader()
	defer r.Close()
	return json.NewDecoder(r).Decode(&v)
}

func (s *Service) Save(c context.Context, name string, data []byte, contentType string) error {
	return saveFile(s.bucket, c, name, data, contentType)
}

func (s *Service) SaveJSON(c context.Context, name string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return saveFile(s.bucket, c, name, data, "application/json")
}

func getFile(bucket *storage.BucketHandle, c context.Context, name string) (model.FileHandle, error) {
	if bucket == nil {
		return nil, nil
	}
	ctx, span := tracing.DefaultTracer.Start(c, "cache.Service.getFile")
	defer span.End()
	span.SetAttributes(attribute.String("cache.object.name", name))

	obj := bucket.Object(name)
	// TODO: concurrent read
	attrs, err := obj.Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	or, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}

	return &cacheFileHandle{or, attrs}, nil
}

func saveFile(bucket *storage.BucketHandle, c context.Context, name string, data []byte, contentType string) error {
	if bucket == nil {
		return nil
	}
	ctx, span := tracing.DefaultTracer.Start(c, "cache.Service.saveFile")
	defer span.End()
	span.SetAttributes(attribute.String("cache.object.name", name))

	w := bucket.Object(name).NewWriter(ctx)
	defer w.Close()
	w.ContentType = contentType
	_, err := w.Write(data)
	return err
}
