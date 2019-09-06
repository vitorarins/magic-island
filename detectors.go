package main

import (
	"cloud.google.com/go/datastore"
	"context"
)

type Detector struct {
	Name   string `datastore:"name"`
	Status string `datastore:"status"`
	id     int64  // The integer ID used in the datastore.
}

type Storer interface {
	PutDetector(id int64, name, status string) error
	GetDetector(detectorID int64) (Detector, error)
}

type storerImpl struct {
	ctx    context.Context
	client *datastore.Client
}

func NewStorer(ctx context.Context, client *datastore.Client) Storer {
	return &storerImpl{
		ctx:    ctx,
		client: client,
	}
}

// PutDetector adds or updates a detector with the given name and id
// to the datastore, returning the key of the newly created entity.
func (s *storerImpl) PutDetector(id int64, name, status string) error {
	detector := &Detector{
		Name:   name,
		Status: status,
	}
	key := datastore.IDKey("Detector", id, nil)
	_, err := s.client.Put(s.ctx, key, detector)
	return err
}

func (s *storerImpl) GetDetector(detectorID int64) (Detector, error) {
	// Create a key using the given integer ID.
	key := datastore.IDKey("Detector", detectorID, nil)

	var detector Detector
	return detector, s.client.Get(s.ctx, key, &detector)
}
